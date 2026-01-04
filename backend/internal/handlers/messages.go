package handlers

import (
	"context"
	"net/http"
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
	var contactNumber string
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		return conn.QueryRow(ctx, "SELECT contact_number FROM conversations WHERE id=$1 AND tenant_id=$2", req.ConversationID, tenantID).Scan(&contactNumber)
	}); err != nil {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}

	// Send via WhatsApp
	if a.WhatsApp != nil {
		if err := a.WhatsApp.SendMessage(ctx, tenantID, contactNumber, req.Content); err != nil {
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
		a.Hub.Broadcast(tenantID, map[string]any{
			"type":       "message.reply",
			"message_id": message.ID,
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
		a.Hub.Broadcast(tenantID, map[string]any{
			"type":       "message.forward",
			"message_id": message.ID,
		})
	}

	writeJSON(w, http.StatusCreated, message)
}
