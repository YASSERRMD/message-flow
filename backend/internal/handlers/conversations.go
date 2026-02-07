package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"message-flow/backend/internal/models"
	"go.mau.fi/whatsmeow/types"
)

func (a *API) ListConversations(w http.ResponseWriter, r *http.Request) {
	tenantID := a.tenantID(r)
	page, limit := parsePagination(r)
	offset := (page - 1) * limit

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	type conversationListItem struct {
		models.Conversation
		LastMessage *string `json:"last_message"`
		IsGroup     bool    `json:"is_group"`
	}

	conversations := []conversationListItem{}
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `
				SELECT
					c.id,
					c.tenant_id,
					c.contact_number,
					c.whatsapp_jid,
					c.contact_name,
					c.last_message_at,
					c.created_at,
					c.profile_picture_url,
					c.unread_count,
					lm.content
				FROM conversations c
				LEFT JOIN LATERAL (
					SELECT m.content
					FROM messages m
					WHERE m.tenant_id=c.tenant_id AND m.conversation_id=c.id
					ORDER BY m.timestamp DESC
					LIMIT 1
				) lm ON true
				WHERE c.tenant_id=$1
				ORDER BY c.last_message_at DESC NULLS LAST, c.created_at DESC
				LIMIT $2 OFFSET $3`, tenantID, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var convo conversationListItem
			if err := rows.Scan(
				&convo.ID,
				&convo.TenantID,
				&convo.ContactNumber,
				&convo.WhatsAppJID,
				&convo.ContactName,
				&convo.LastMessageAt,
				&convo.CreatedAt,
				&convo.ProfilePictureURL,
				&convo.UnreadCount,
				&convo.LastMessage,
			); err != nil {
				return err
			}
			jid := convo.ContactNumber
			if convo.WhatsAppJID != nil && strings.TrimSpace(*convo.WhatsAppJID) != "" {
				jid = *convo.WhatsAppJID
			}
			convo.IsGroup = strings.Contains(jid, "@g.us") || strings.HasPrefix(convo.ContactNumber, "12036")
			conversations = append(conversations, convo)
		}
		return rows.Err()
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list conversations")
		return
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

	// Best-effort sender name lookup for group conversations.
	// This makes older messages show names even if metadata_json lacks push_name.
	var contactNameCache map[string]string
	var waClientLookup func(jidStr string) string
	if a.WhatsApp != nil {
		if cli, err := a.WhatsApp.ClientForTenant(tenantID); err == nil && cli != nil && cli.Store != nil && cli.Store.Contacts != nil {
			contactNameCache = map[string]string{}
			waClientLookup = func(jidStr string) string {
				jidStr = strings.TrimSpace(jidStr)
				if jidStr == "" {
					return ""
				}
				if name, ok := contactNameCache[jidStr]; ok {
					return name
				}
				jid, err := types.ParseJID(jidStr)
				if err != nil {
					contactNameCache[jidStr] = ""
					return ""
				}
				info, err := cli.Store.Contacts.GetContact(ctx, jid)
				if err != nil || !info.Found {
					contactNameCache[jidStr] = ""
					return ""
				}
				name := strings.TrimSpace(info.FullName)
				if name == "" {
					name = strings.TrimSpace(info.PushName)
				}
				if name == "" {
					name = strings.TrimSpace(info.FirstName)
				}
				contactNameCache[jidStr] = name
				return name
			}
		}
	}

	// We query newest-first for efficient "latest messages" behavior, then reverse to return
	// chronological order for display.
	messagesDesc := []models.Message{}
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `
			SELECT id, tenant_id, conversation_id, sender, content, timestamp, metadata_json, created_at
			FROM messages
			WHERE tenant_id=$1 AND conversation_id=$2
			ORDER BY timestamp DESC
			LIMIT $3 OFFSET $4`, tenantID, conversationID, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var msg models.Message
			if err := rows.Scan(&msg.ID, &msg.TenantID, &msg.ConversationID, &msg.Sender, &msg.Content, &msg.Timestamp, &msg.MetadataJSON, &msg.CreatedAt); err != nil {
				return err
			}
			// Simple logic: if sender is "agent" or "system", it's outbound.
			// In a real app, you'd compare against the tenant's JID.
			if msg.Sender == "agent" || msg.Sender == "me" {
				msg.IsOutbound = true
			} else {
				// Prefer push_name from metadata if present.
				if msg.MetadataJSON != nil && *msg.MetadataJSON != "" {
					var meta map[string]any
					if err := json.Unmarshal([]byte(*msg.MetadataJSON), &meta); err == nil {
						if pn, ok := meta["push_name"].(string); ok && strings.TrimSpace(pn) != "" {
							value := strings.TrimSpace(pn)
							msg.SenderName = &value
						}
					}
				}
				if msg.SenderName == nil && waClientLookup != nil {
					if resolved := strings.TrimSpace(waClientLookup(msg.Sender)); resolved != "" {
						msg.SenderName = &resolved
					}
				}
				if msg.SenderName == nil {
					// Fallback: sender JID like 123456789@s.whatsapp.net -> 123456789...
					name := msg.Sender
					if len(name) > 12 {
						name = name[:12] + "..."
					}
					msg.SenderName = &name
				}
			}
			messagesDesc = append(messagesDesc, msg)
		}
		return rows.Err()
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list messages")
		return
	}

	// Reverse in-place to chronological order.
	for i, j := 0, len(messagesDesc)-1; i < j; i, j = i+1, j-1 {
		messagesDesc[i], messagesDesc[j] = messagesDesc[j], messagesDesc[i]
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  messagesDesc,
		"page":  page,
		"limit": limit,
	})
}
