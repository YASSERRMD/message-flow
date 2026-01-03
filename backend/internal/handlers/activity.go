package handlers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"message-flow/backend/internal/auth"
)

func (a *API) logActivity(ctx context.Context, tenantID int64, user auth.User, action string, details any) {
	if user.ID == 0 {
		return
	}
	payload, err := json.Marshal(details)
	if err != nil {
		return
	}

	_ = a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		_, err := conn.Exec(ctx, `
			INSERT INTO user_activity_logs (tenant_id, user_id, action, details_json, created_at)
			VALUES ($1, $2, $3, $4, $5)`, tenantID, user.ID, action, string(payload), time.Now().UTC())
		return err
	})
}
