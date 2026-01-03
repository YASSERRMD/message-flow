package llm

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"message-flow/backend/internal/db"
)

func StoreAnalysis(ctx context.Context, store *db.Store, tenantID, messageID int64, result *AnalysisResult) error {
	payload, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		_, err := conn.Exec(ctx, `
			UPDATE messages SET metadata_json=$1 WHERE id=$2 AND tenant_id=$3`, string(payload), messageID, tenantID)
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
