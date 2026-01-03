package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type importantMessageResponse struct {
	ID             int64     `json:"id"`
	MessageID      int64     `json:"message_id"`
	Priority       string    `json:"priority"`
	Reason         *string   `json:"reason"`
	CreatedAt      time.Time `json:"created_at"`
	ConversationID int64     `json:"conversation_id"`
	Sender         string    `json:"sender"`
	Content        string    `json:"content"`
	Timestamp      time.Time `json:"timestamp"`
}

func (a *API) ListImportantMessages(w http.ResponseWriter, r *http.Request) {
	tenantID := a.tenantID(r)
	page, limit := parsePagination(r)
	offset := (page - 1) * limit

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items := []importantMessageResponse{}
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `
			SELECT im.id, im.message_id, im.priority, im.reason, im.created_at,
			       m.conversation_id, m.sender, m.content, m.timestamp
			FROM important_messages im
			JOIN messages m ON m.id = im.message_id
			WHERE im.tenant_id=$1
			ORDER BY im.created_at DESC
			LIMIT $2 OFFSET $3`, tenantID, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var item importantMessageResponse
			if err := rows.Scan(&item.ID, &item.MessageID, &item.Priority, &item.Reason, &item.CreatedAt, &item.ConversationID, &item.Sender, &item.Content, &item.Timestamp); err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list important messages")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  items,
		"page":  page,
		"limit": limit,
	})
}
