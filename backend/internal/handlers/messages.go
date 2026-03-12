package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"message-flow/backend/internal/auth"
	"message-flow/backend/internal/models"
)

type replyRequest struct {
	ConversationID int64  `json:"conversation_id"`
	Content        string `json:"content"`
	Sender         string `json:"sender"`
}

type forwardRequest struct {
	MessageID            int64  `json:"message_id"`
	TargetConversationID int64  `json:"target_conversation_id"`
	Sender               string `json:"sender"`
}

func (a *API) GetMessage(w http.ResponseWriter, r *http.Request, messageID int64) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var msg models.Message
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		return conn.QueryRow(ctx, `
			SELECT id, tenant_id, conversation_id, sender, content, timestamp, metadata_json, created_at
			FROM messages
			WHERE tenant_id=$1 AND id=$2`, tenantID, messageID).Scan(
			&msg.ID,
			&msg.TenantID,
			&msg.ConversationID,
			&msg.Sender,
			&msg.Content,
			&msg.Timestamp,
			&msg.MetadataJSON,
			&msg.CreatedAt,
		)
	}); err != nil {
		writeError(w, http.StatusNotFound, "message not found")
		return
	}

	if msg.Sender == "agent" || msg.Sender == "me" {
		msg.IsOutbound = true
	} else {
		if msg.MetadataJSON != nil && *msg.MetadataJSON != "" {
			var meta map[string]any
			if err := json.Unmarshal([]byte(*msg.MetadataJSON), &meta); err == nil {
				if pn, ok := meta["push_name"].(string); ok && strings.TrimSpace(pn) != "" {
					value := strings.TrimSpace(pn)
					msg.SenderName = &value
				}
			}
		}
		if msg.SenderName == nil {
			name := msg.Sender
			if len(name) > 12 {
				name = name[:12] + "..."
			}
			msg.SenderName = &name
		}
	}

	writeJSON(w, http.StatusOK, msg)
}

func (a *API) GetMessageMedia(w http.ResponseWriter, r *http.Request, messageID int64) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var metadataJSON *string
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		return conn.QueryRow(ctx, `
			SELECT metadata_json
			FROM messages
			WHERE tenant_id=$1 AND id=$2`, tenantID, messageID).Scan(&metadataJSON)
	}); err != nil {
		writeError(w, http.StatusNotFound, "message not found")
		return
	}

	if metadataJSON == nil || *metadataJSON == "" {
		writeError(w, http.StatusNotFound, "no media for this message")
		return
	}

	var meta map[string]any
	if err := json.Unmarshal([]byte(*metadataJSON), &meta); err != nil {
		writeError(w, http.StatusInternalServerError, "invalid metadata")
		return
	}

	mediaMap, _ := meta["media"].(map[string]any)
	if mediaMap == nil {
		writeError(w, http.StatusNotFound, "no media for this message")
		return
	}
	mediaPath, _ := mediaMap["media_path"].(string)
	if mediaPath == "" {
		writeError(w, http.StatusNotFound, "media file not available")
		return
	}

	absPath := a.MediaDir + "/" + mediaPath
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "media file not found on disk")
		return
	}

	// Set Content-Type from metadata
	if mimeType, ok := mediaMap["mime_type"].(string); ok && mimeType != "" {
		w.Header().Set("Content-Type", mimeType)
	}

	http.ServeFile(w, r, absPath)
}

func (a *API) ReplyMessage(w http.ResponseWriter, r *http.Request) {
	var req replyRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.ConversationID == 0 || req.Content == "" {
		writeError(w, http.StatusBadRequest, "conversation_id and content are required")
		return
	}
	sender := req.Sender
	if sender == "" {
		sender = "agent"
	}

	tenantID := a.tenantID(r)
	now := time.Now().UTC()

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var message models.Message

	// Get recipient number from conversation
	var recipient string
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		return conn.QueryRow(ctx, `
			SELECT COALESCE(NULLIF(whatsapp_jid,''), contact_number)
			FROM conversations
			WHERE id=$1 AND tenant_id=$2`, req.ConversationID, tenantID).Scan(&recipient)
	}); err != nil {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}

	// Send via WhatsApp
	if a.WhatsApp != nil {
		if err := a.WhatsApp.SendMessage(ctx, tenantID, recipient, req.Content); err != nil {
			// Log error but continue to save (or should we fail? usually better to fail if send fails)
			// But for now, let's return error so user knows
			writeError(w, http.StatusInternalServerError, "failed to send whatsapp message: "+err.Error())
			return
		}
	}

	query := `
		INSERT INTO messages (tenant_id, conversation_id, sender, content, timestamp, metadata_json, created_at)
		VALUES ($1, $2, $3, $4, $5, NULL, $6)
		RETURNING id, tenant_id, conversation_id, sender, content, timestamp, metadata_json, created_at`

	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		return conn.QueryRow(ctx, query, tenantID, req.ConversationID, sender, req.Content, now, now).Scan(
			&message.ID, &message.TenantID, &message.ConversationID, &message.Sender, &message.Content, &message.Timestamp, &message.MetadataJSON, &message.CreatedAt,
		)
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save message")
		return
	}
	if message.Sender == "agent" || message.Sender == "me" {
		message.IsOutbound = true
	}

	_ = a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		_, err := conn.Exec(ctx, `
			UPDATE conversations SET last_message_at=$1
			WHERE id=$2 AND tenant_id=$3`, now, req.ConversationID, tenantID)
		return err
	})

	if user, ok := auth.UserFromContext(r.Context()); ok {
		a.logActivity(ctx, tenantID, user, "message.reply", map[string]any{
			"conversation_id": req.ConversationID,
			"message_id":      message.ID,
		})
	}
	if a.Hub != nil {
		var unreadCount int64
		_ = a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
			return conn.QueryRow(ctx, `
				SELECT unread_count
				FROM conversations
				WHERE tenant_id=$1 AND id=$2`, tenantID, req.ConversationID).Scan(&unreadCount)
		})
		a.Hub.Broadcast(tenantID, map[string]any{
			"type":            "message.reply",
			"message_id":      message.ID,
			"conversation_id": req.ConversationID,
			"message":         message,
			"conversation": map[string]any{
				"id":              req.ConversationID,
				"last_message":    message.Content,
				"last_message_at": message.Timestamp,
				"unread_count":    unreadCount,
			},
		})
	}

	writeJSON(w, http.StatusCreated, message)
}

func (a *API) ForwardMessage(w http.ResponseWriter, r *http.Request) {
	var req forwardRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.MessageID == 0 || req.TargetConversationID == 0 {
		writeError(w, http.StatusBadRequest, "message_id and target_conversation_id are required")
		return
	}
	sender := req.Sender
	if sender == "" {
		sender = "agent"
	}

	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var content string
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		return conn.QueryRow(ctx, `
			SELECT content FROM messages WHERE id=$1 AND tenant_id=$2`, req.MessageID, tenantID).Scan(&content)
	}); err != nil {
		writeError(w, http.StatusNotFound, "message not found")
		return
	}

	now := time.Now().UTC()
	var message models.Message
	query := `
		INSERT INTO messages (tenant_id, conversation_id, sender, content, timestamp, metadata_json, created_at)
		VALUES ($1, $2, $3, $4, $5, NULL, $6)
		RETURNING id, tenant_id, conversation_id, sender, content, timestamp, metadata_json, created_at`

	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		return conn.QueryRow(ctx, query, tenantID, req.TargetConversationID, sender, content, now, now).Scan(
			&message.ID, &message.TenantID, &message.ConversationID, &message.Sender, &message.Content, &message.Timestamp, &message.MetadataJSON, &message.CreatedAt,
		)
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to forward message")
		return
	}
	if message.Sender == "agent" || message.Sender == "me" {
		message.IsOutbound = true
	}

	_ = a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		_, err := conn.Exec(ctx, `
			UPDATE conversations SET last_message_at=$1
			WHERE id=$2 AND tenant_id=$3`, now, req.TargetConversationID, tenantID)
		return err
	})

	if user, ok := auth.UserFromContext(r.Context()); ok {
		a.logActivity(ctx, tenantID, user, "message.forward", map[string]any{
			"source_message_id": req.MessageID,
			"target_message_id": message.ID,
			"conversation_id":   req.TargetConversationID,
		})
	}
	if a.Hub != nil {
		var unreadCount int64
		_ = a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
			return conn.QueryRow(ctx, `
				SELECT unread_count
				FROM conversations
				WHERE tenant_id=$1 AND id=$2`, tenantID, req.TargetConversationID).Scan(&unreadCount)
		})
		a.Hub.Broadcast(tenantID, map[string]any{
			"type":            "message.forward",
			"message_id":      message.ID,
			"conversation_id": req.TargetConversationID,
			"message":         message,
			"conversation": map[string]any{
				"id":              req.TargetConversationID,
				"last_message":    message.Content,
				"last_message_at": message.Timestamp,
				"unread_count":    unreadCount,
			},
		})
	}

	writeJSON(w, http.StatusCreated, message)
}
