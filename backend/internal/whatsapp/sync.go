package whatsapp

import (
	"context"
	"encoding/json"
	"log"
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
		log.Printf("[Syncer] Cannot attach: s=%v, client=%v, store=%v", s != nil, client != nil, s != nil && s.Store != nil)
		return
	}
	log.Printf("[Syncer] Attaching event handler for tenant %d", tenantID)
	client.AddEventHandler(func(evt any) {
		ctx := context.Background()
		switch event := evt.(type) {
		case *events.Message:
			log.Printf("[Syncer] Received *events.Message from %s", event.Info.Chat.String())
			s.handleMessage(ctx, tenantID, client, event.Info, event.Message, event.Info.Chat, event.Info.PushName)
		case events.Message:
			log.Printf("[Syncer] Received events.Message from %s", event.Info.Chat.String())
			s.handleMessage(ctx, tenantID, client, event.Info, event.Message, event.Info.Chat, event.Info.PushName)
		case *events.HistorySync:
			log.Printf("[Syncer] Received *events.HistorySync with %d conversations", len(event.Data.GetConversations()))
			s.handleHistorySync(ctx, tenantID, client, event)
		case events.HistorySync:
			log.Printf("[Syncer] Received events.HistorySync with %d conversations", len(event.Data.GetConversations()))
			s.handleHistorySync(ctx, tenantID, client, &event)
		}
	})
}

func (s *Syncer) handleHistorySync(ctx context.Context, tenantID int64, client *whatsmeow.Client, evt *events.HistorySync) {
	if evt == nil || evt.Data == nil || client == nil {
		return
	}
	conversations := evt.Data.GetConversations()
	log.Printf("[Syncer] Processing history sync for %d conversations", len(conversations))
	for _, conv := range conversations {
		chatJID, err := types.ParseJID(conv.GetID())
		if err != nil {
			continue
		}
		contactName := strings.TrimSpace(conv.GetName())
		messages := conv.GetMessages()
		log.Printf("[Syncer] Processing %d messages for conversation %s", len(messages), chatJID.String())

		for _, historyMsg := range messages {
			webMsg := historyMsg.GetMessage()
			if webMsg == nil {
				continue
			}
			msgEvt, err := client.ParseWebMessage(chatJID, webMsg)
			if err != nil {
				log.Printf("[Syncer] Failed to parse message: %v", err)
				continue
			}
			s.handleMessage(ctx, tenantID, client, msgEvt.Info, msgEvt.Message, chatJID, contactName)
		}
	}
}

func (s *Syncer) handleMessage(ctx context.Context, tenantID int64, client *whatsmeow.Client, info types.MessageInfo, msg *waE2E.Message, chatJID types.JID, contactName string) {
	if s == nil || s.Store == nil {
		log.Printf("[Syncer] handleMessage: nil store")
		return
	}
	content := extractText(msg)
	mediaInfo := extractMediaInfo(msg)

	// Allow message if either has text content or has media
	if (content == "" && mediaInfo == nil) || info.ID == "" {
		return
	}

	// For media-only messages, set a placeholder content
	if content == "" && mediaInfo != nil {
		content = "[" + mediaInfo.Type + "]"
	}

	if contactName == "" {
		contactName = strings.TrimSpace(info.PushName)
	}

	// Ensure conversation exists and get ID
	conversationID, err := s.UpsertConversation(ctx, tenantID, client, chatJID, contactName, info.Timestamp)
	if err != nil {
		log.Printf("[Syncer] Failed to upsert conversation: %v", err)
		return
	}
	if conversationID == 0 {
		log.Printf("[Syncer] UpsertConversation returned 0 ID")
		return
	}

	messageID, inserted, err := s.insertMessage(ctx, tenantID, conversationID, info, content, chatJID, mediaInfo)
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

func (s *Syncer) UpsertConversation(ctx context.Context, tenantID int64, client *whatsmeow.Client, chatJID types.JID, contactName string, lastMessageAt time.Time) (int64, error) {
	contactNumber := chatJID.User
	if contactNumber == "" {
		// Fallback, but strip domain if possible
		str := chatJID.String()
		if idx := strings.Index(str, "@"); idx != -1 {
			contactNumber = str[:idx]
		} else {
			contactNumber = str
		}
	}

	// Try to get better name from store if missing
	if contactName == "" || contactName == contactNumber {
		if info, err := client.Store.Contacts.GetContact(ctx, chatJID); err == nil && info.Found {
			if info.PushName != "" {
				contactName = info.PushName
			} else if info.FullName != "" {
				contactName = info.FullName
			} else if info.FirstName != "" {
				contactName = info.FirstName
			}
		}
	}

	// Fetch profile picture
	var profilePicURL string
	if params, err := client.GetProfilePictureInfo(ctx, chatJID, &whatsmeow.GetProfilePictureParams{Preview: true}); err == nil && params != nil {
		profilePicURL = params.URL
	}

	var id int64

	// Use ON CONFLICT to handle concurrent updates and duplicates robustly
	err := s.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		// We use a transaction to ensure atomic upsert
		tx, err := conn.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)

		// 1. Try to INSERT
		// We use ON CONFLICT DO UPDATE to handle race conditions
		// Note: We need a unique constraint on (tenant_id, contact_number) for this to work
		// which we added in migration 009.
		var insertedID int64
		err = tx.QueryRow(ctx, `
			INSERT INTO conversations (tenant_id, contact_number, contact_name, last_message_at, profile_picture_url, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (tenant_id, contact_number) 
			DO UPDATE SET 
				last_message_at = GREATEST(conversations.last_message_at, EXCLUDED.last_message_at),
				contact_name = COALESCE(NULLIF(EXCLUDED.contact_name, ''), conversations.contact_name),
				profile_picture_url = COALESCE(NULLIF(EXCLUDED.profile_picture_url, ''), conversations.profile_picture_url)
			RETURNING id`,
			tenantID, contactNumber, contactName, lastMessageAt, profilePicURL, time.Now().UTC()).Scan(&insertedID)

		if err != nil {
			return err
		}

		id = insertedID
		return tx.Commit(ctx)
	})
	return id, err
}

func (s *Syncer) insertMessage(ctx context.Context, tenantID, conversationID int64, info types.MessageInfo, content string, chatJID types.JID, mediaInfo *MediaInfo) (int64, bool, error) {
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
		// Include media info if present
		if mediaInfo != nil {
			meta["media"] = mediaInfo
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

// MediaInfo holds extracted media metadata
type MediaInfo struct {
	Type     string `json:"media_type,omitempty"` // image, video, audio, document, sticker
	MimeType string `json:"mime_type,omitempty"`
	FileName string `json:"file_name,omitempty"`
	Caption  string `json:"caption,omitempty"`
	FileSize uint64 `json:"file_size,omitempty"`
	Seconds  uint32 `json:"duration_seconds,omitempty"`
	HasMedia bool   `json:"has_media"`
}

func extractMediaInfo(msg *waE2E.Message) *MediaInfo {
	if msg == nil {
		return nil
	}

	if img := msg.GetImageMessage(); img != nil {
		return &MediaInfo{
			Type:     "image",
			MimeType: img.GetMimetype(),
			Caption:  img.GetCaption(),
			FileSize: img.GetFileLength(),
			HasMedia: true,
		}
	}

	if vid := msg.GetVideoMessage(); vid != nil {
		return &MediaInfo{
			Type:     "video",
			MimeType: vid.GetMimetype(),
			Caption:  vid.GetCaption(),
			FileSize: vid.GetFileLength(),
			Seconds:  vid.GetSeconds(),
			HasMedia: true,
		}
	}

	if audio := msg.GetAudioMessage(); audio != nil {
		return &MediaInfo{
			Type:     "audio",
			MimeType: audio.GetMimetype(),
			FileSize: audio.GetFileLength(),
			Seconds:  audio.GetSeconds(),
			HasMedia: true,
		}
	}

	if doc := msg.GetDocumentMessage(); doc != nil {
		return &MediaInfo{
			Type:     "document",
			MimeType: doc.GetMimetype(),
			FileName: doc.GetFileName(),
			Caption:  doc.GetCaption(),
			FileSize: doc.GetFileLength(),
			HasMedia: true,
		}
	}

	if sticker := msg.GetStickerMessage(); sticker != nil {
		return &MediaInfo{
			Type:     "sticker",
			MimeType: sticker.GetMimetype(),
			FileSize: sticker.GetFileLength(),
			HasMedia: true,
		}
	}

	return nil
}
