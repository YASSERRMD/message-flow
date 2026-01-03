package handlers

import (
	"context"
	"net/http"
	"time"

	"message-flow/backend/internal/models"
)

func (a *API) ListConversations(w http.ResponseWriter, r *http.Request) {
	tenantID := a.tenantID(r)
	page, limit := parsePagination(r)
	offset := (page - 1) * limit

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := a.Store.Pool.Query(ctx, `
		SELECT id, tenant_id, contact_number, contact_name, last_message_at, created_at
		FROM conversations
		WHERE tenant_id=$1
		ORDER BY last_message_at DESC NULLS LAST, created_at DESC
		LIMIT $2 OFFSET $3`, tenantID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list conversations")
		return
	}
	defer rows.Close()

	conversations := []models.Conversation{}
	for rows.Next() {
		var convo models.Conversation
		if err := rows.Scan(&convo.ID, &convo.TenantID, &convo.ContactNumber, &convo.ContactName, &convo.LastMessageAt, &convo.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read conversations")
			return
		}
		conversations = append(conversations, convo)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  conversations,
		"page":  page,
		"limit": limit,
	})
}

func (a *API) GetConversationMessages(w http.ResponseWriter, r *http.Request, conversationID int64) {
	tenantID := a.tenantID(r)
	page, limit := parsePagination(r)
	offset := (page - 1) * limit

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := a.Store.Pool.Query(ctx, `
		SELECT id, tenant_id, conversation_id, sender, content, timestamp, metadata_json, created_at
		FROM messages
		WHERE tenant_id=$1 AND conversation_id=$2
		ORDER BY timestamp ASC
		LIMIT $3 OFFSET $4`, tenantID, conversationID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list messages")
		return
	}
	defer rows.Close()

	messages := []models.Message{}
	for rows.Next() {
		var msg models.Message
		if err := rows.Scan(&msg.ID, &msg.TenantID, &msg.ConversationID, &msg.Sender, &msg.Content, &msg.Timestamp, &msg.MetadataJSON, &msg.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read messages")
			return
		}
		messages = append(messages, msg)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  messages,
		"page":  page,
		"limit": limit,
	})
}
