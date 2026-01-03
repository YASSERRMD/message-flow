package whatsapp

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"message-flow/backend/internal/db"
	"message-flow/backend/internal/llm"
	"message-flow/backend/internal/realtime"
)

type Syncer struct {
	Store *db.Store
	Queue *llm.Queue
	Hub   *realtime.Hub
}

func NewSyncer(store *db.Store, queue *llm.Queue, hub *realtime.Hub) *Syncer {
	return &Syncer{Store: store, Queue: queue, Hub: hub}
}

func (s *Syncer) Attach(tenantID int64, client *whatsmeow.Client) {
	if s == nil || client == nil || s.Store == nil {
		return
	}
	client.AddEventHandler(func(evt any) {
		ctx := context.Background()
		switch event := evt.(type) {
		case *events.Message:
			s.handleMessage(ctx, tenantID, client, event.Info, event.Message, event.Info.Chat, event.Info.PushName)
		case events.Message:
			s.handleMessage(ctx, tenantID, client, event.Info, event.Message, event.Info.Chat, event.Info.PushName)
		case *events.HistorySync:
			s.handleHistorySync(ctx, tenantID, client, event)
		case events.HistorySync:
			s.handleHistorySync(ctx, tenantID, client, &event)
		}
	})
}

func (s *Syncer) handleHistorySync(ctx context.Context, tenantID int64, client *whatsmeow.Client, evt *events.HistorySync) {
	if evt == nil || evt.Data == nil || client == nil {
		return
	}
	for _, conv := range evt.Data.GetConversations() {
		chatJID, err := types.ParseJID(conv.GetID())
		if err != nil {
			continue
		}
		contactName := strings.TrimSpace(conv.GetName())
		for _, historyMsg := range conv.GetMessages() {
			webMsg := historyMsg.GetMessage()
			if webMsg == nil {
				continue
			}
			msgEvt, err := client.ParseWebMessage(chatJID, webMsg)
			if err != nil {
				continue
			}
			s.handleMessage(ctx, tenantID, client, msgEvt.Info, msgEvt.Message, chatJID, contactName)
		}
	}
}

func (s *Syncer) handleMessage(ctx context.Context, tenantID int64, client *whatsmeow.Client, info types.MessageInfo, msg *waE2E.Message, chatJID types.JID, contactName string) {
	if s == nil || s.Store == nil {
		return
	}
	content := extractText(msg)
	if content == "" || info.ID == "" {
		return
	}

	contactNumber := chatJID.String()
	if chatJID.User != "" {
		contactNumber = chatJID.User
	}
	if contactName == "" {
		contactName = strings.TrimSpace(info.PushName)
	}

	conversationID, err := s.upsertConversation(ctx, tenantID, contactNumber, contactName, info.Timestamp)
	if err != nil || conversationID == 0 {
		return
	}

	messageID, inserted, err := s.insertMessage(ctx, tenantID, conversationID, info, content, chatJID)
	if err != nil || !inserted {
		return
	}

	if s.Queue != nil {
		_ = s.Queue.Enqueue(ctx, llm.QueueMessage{
			TenantID:  tenantID,
			MessageID: messageID,
			Content:   content,
			Feature:   "analysis",
			CreatedAt: time.Now().UTC(),
		})
	}

	if s.Hub != nil {
		s.Hub.Broadcast(tenantID, map[string]any{
			"type":            "message.received",
			"message_id":      messageID,
			"conversation_id": conversationID,
		})
	}
}

func (s *Syncer) upsertConversation(ctx context.Context, tenantID int64, contactNumber, contactName string, lastMessageAt time.Time) (int64, error) {
	var id int64
	var existingName string
	err := s.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		err := conn.QueryRow(ctx, `
			SELECT id, contact_name
			FROM conversations
			WHERE tenant_id=$1 AND contact_number=$2`, tenantID, contactNumber).Scan(&id, &existingName)
		if err != nil {
			if err == pgx.ErrNoRows {
				name := contactName
				if name == "" {
					name = contactNumber
				}
				return conn.QueryRow(ctx, `
					INSERT INTO conversations (tenant_id, contact_number, contact_name, last_message_at, created_at)
					VALUES ($1, $2, $3, $4, $5)
					RETURNING id`, tenantID, contactNumber, name, lastMessageAt, time.Now().UTC()).Scan(&id)
			}
			return err
		}
		name := existingName
		if name == "" && contactName != "" {
			name = contactName
		}
		_, err = conn.Exec(ctx, `
			UPDATE conversations
			SET last_message_at=$1, contact_name=$2
			WHERE id=$3 AND tenant_id=$4`, lastMessageAt, name, id, tenantID)
		return err
	})
	return id, err
}

func (s *Syncer) insertMessage(ctx context.Context, tenantID, conversationID int64, info types.MessageInfo, content string, chatJID types.JID) (int64, bool, error) {
	var id int64
	inserted := false
	err := s.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		var existingID int64
		err := conn.QueryRow(ctx, `
			SELECT id
			FROM messages
			WHERE tenant_id=$1 AND metadata_json->>'whatsapp_id'=$2
			LIMIT 1`, tenantID, info.ID).Scan(&existingID)
		if err == nil {
			id = existingID
			return nil
		}
		if err != pgx.ErrNoRows {
			return err
		}

		sender := info.Sender.String()
		if info.IsFromMe {
			sender = "me"
		}
		meta := map[string]any{
			"source":      "whatsapp",
			"whatsapp_id": info.ID,
			"sender":      sender,
			"chat":        chatJID.String(),
		}
		metaBytes, _ := json.Marshal(meta)
		now := time.Now().UTC()
		return conn.QueryRow(ctx, `
			INSERT INTO messages (tenant_id, conversation_id, sender, content, timestamp, metadata_json, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id`, tenantID, conversationID, sender, content, info.Timestamp, string(metaBytes), now).Scan(&id)
	})
	if err == nil {
		inserted = true
	}
	return id, inserted, err
}

func extractText(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}
	if text := msg.GetConversation(); text != "" {
		return text
	}
	if ext := msg.GetExtendedTextMessage(); ext != nil && ext.GetText() != "" {
		return ext.GetText()
	}
	if img := msg.GetImageMessage(); img != nil && img.GetCaption() != "" {
		return img.GetCaption()
	}
	if vid := msg.GetVideoMessage(); vid != nil && vid.GetCaption() != "" {
		return vid.GetCaption()
	}
	if doc := msg.GetDocumentMessage(); doc != nil && doc.GetCaption() != "" {
		return doc.GetCaption()
	}
	if buttons := msg.GetButtonsResponseMessage(); buttons != nil && buttons.GetSelectedDisplayText() != "" {
		return buttons.GetSelectedDisplayText()
	}
	if list := msg.GetListResponseMessage(); list != nil && list.GetTitle() != "" {
		return list.GetTitle()
	}
	if template := msg.GetTemplateButtonReplyMessage(); template != nil && template.GetSelectedDisplayText() != "" {
		return template.GetSelectedDisplayText()
	}
	if poll := msg.GetPollCreationMessage(); poll != nil && poll.GetName() != "" {
		return poll.GetName()
	}
	return ""
}
