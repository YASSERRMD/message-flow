package llm

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"message-flow/backend/internal/db"
)

func StoreAnalysis(ctx context.Context, store *db.Store, tenantID, messageID int64, result *AnalysisResult) error {
	return store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		var existing string
		_ = conn.QueryRow(ctx, `
			SELECT metadata_json FROM messages WHERE id=$1 AND tenant_id=$2`, messageID, tenantID).Scan(&existing)

		payload := map[string]any{}
		if existing != "" {
			_ = json.Unmarshal([]byte(existing), &payload)
		}
		payload["analysis"] = result

		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		_, err = conn.Exec(ctx, `
			UPDATE messages SET metadata_json=$1 WHERE id=$2 AND tenant_id=$3`, string(encoded), messageID, tenantID)
		if err != nil {
			return err
		}
		if result.IsImportant {
			_, err = conn.Exec(ctx, `
				INSERT INTO important_messages (tenant_id, message_id, priority, reason, created_at)
				VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT DO NOTHING`, tenantID, messageID, result.Priority, result.Reason, time.Now().UTC())
			if err != nil {
				return err
			}
		}
		return nil
	})
}
