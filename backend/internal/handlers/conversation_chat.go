package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"message-flow/backend/internal/models"
)

type conversationChatRequest struct {
	Question   string `json:"question"`
	ProviderID *int64 `json:"provider_id"`
	Limit      *int   `json:"limit"`
}

type conversationChatResponse struct {
	Answer       string `json:"answer"`
	ProviderID   int64  `json:"provider_id"`
	MessageCount int    `json:"message_count"`
}

func (a *API) ChatConversation(w http.ResponseWriter, r *http.Request, conversationID int64) {
	if a.LLM == nil || a.LLM.Router == nil {
		writeError(w, http.StatusServiceUnavailable, "llm not configured")
		return
	}

	var req conversationChatRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	req.Question = strings.TrimSpace(req.Question)
	if req.Question == "" {
		writeError(w, http.StatusBadRequest, "question is required")
		return
	}

	limit := 120
	if req.Limit != nil && *req.Limit > 0 {
		limit = *req.Limit
	}
	if limit > 400 {
		limit = 400
	}

	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	// Load last N messages (newest-first), then reverse to chronological order.
	msgsDesc := []models.Message{}
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `
			SELECT id, tenant_id, conversation_id, sender, content, timestamp, metadata_json, created_at
			FROM messages
			WHERE tenant_id=$1 AND conversation_id=$2
			ORDER BY timestamp DESC
			LIMIT $3`, tenantID, conversationID, limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var msg models.Message
			if err := rows.Scan(&msg.ID, &msg.TenantID, &msg.ConversationID, &msg.Sender, &msg.Content, &msg.Timestamp, &msg.MetadataJSON, &msg.CreatedAt); err != nil {
				return err
			}
			msgsDesc = append(msgsDesc, msg)
		}
		return rows.Err()
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load messages")
		return
	}

	for i, j := 0, len(msgsDesc)-1; i < j; i, j = i+1, j-1 {
		msgsDesc[i], msgsDesc[j] = msgsDesc[j], msgsDesc[i]
	}

	transcript := buildTranscript(msgsDesc)

	providerID := int64(0)
	if req.ProviderID != nil {
		providerID = *req.ProviderID
	}
	if providerID == 0 {
		provider, err := a.LLM.Router.GetDefaultProvider(ctx, tenantID)
		if err == nil && provider != nil && provider.GetConfig() != nil {
			providerID = provider.GetConfig().ID
		}
	}
	if providerID == 0 {
		writeError(w, http.StatusServiceUnavailable, "no llm provider configured")
		return
	}

	prompt := strings.TrimSpace(`
You are a helpful assistant. You will be given a WhatsApp chat transcript and a question.

Rules:
- Use only the transcript to answer.
- If the answer is not in the transcript, say you don't know based on the messages provided.
- Keep the answer concise and practical.

Transcript:
` + transcript + `

Question:
` + req.Question + `
`)

	answer, err := a.LLM.Chat(ctx, tenantID, providerID, prompt)
	if err != nil {
		// Fallback: return a safe response when LLM fails.
		answer = "AI chat is temporarily unavailable. Please try again after configuring or verifying your LLM provider."
	}

	writeJSON(w, http.StatusOK, conversationChatResponse{
		Answer:       answer,
		ProviderID:   providerID,
		MessageCount: len(msgsDesc),
	})
}

func buildTranscript(messages []models.Message) string {
	var b strings.Builder
	for _, m := range messages {
		sender := senderLabel(m)
		content := strings.TrimSpace(m.Content)
		if content == "" {
			continue
		}
		b.WriteString(m.Timestamp.UTC().Format("2006-01-02 15:04"))
		b.WriteString(" ")
		b.WriteString(sender)
		b.WriteString(": ")
		b.WriteString(content)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func senderLabel(m models.Message) string {
	if m.Sender == "me" || m.Sender == "agent" {
		return "Me"
	}

	// If we stored a push_name in metadata_json, prefer it.
	if m.MetadataJSON != nil && *m.MetadataJSON != "" {
		var meta map[string]any
		if err := json.Unmarshal([]byte(*m.MetadataJSON), &meta); err == nil {
			if pn, ok := meta["push_name"].(string); ok && strings.TrimSpace(pn) != "" {
				return strings.TrimSpace(pn)
			}
		}
	}

	// Fallback to sender (jid) prefix.
	if idx := strings.Index(m.Sender, "@"); idx > 0 {
		return m.Sender[:idx]
	}
	return m.Sender
}
