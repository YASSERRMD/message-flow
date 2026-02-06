package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func (a *API) MarkConversationRead(w http.ResponseWriter, r *http.Request, conversationID int64) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		_, err := conn.Exec(ctx, `
			UPDATE conversations
			SET unread_count = 0
			WHERE tenant_id=$1 AND id=$2`, tenantID, conversationID)
		return err
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to mark read")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}
