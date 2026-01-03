package llm

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"message-flow/backend/internal/crypto"
	"message-flow/backend/internal/db"
)

type Store struct {
	DB        *db.Store
	MasterKey string
}

func NewStore(store *db.Store, masterKey string) *Store {
	return &Store{DB: store, MasterKey: masterKey}
}

func (s *Store) ListProviders(ctx context.Context, tenantID int64) ([]ProviderConfig, error) {
	var configs []ProviderConfig
	err := s.DB.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `
			SELECT id, provider_name, api_key, model_name, temperature, max_tokens, cost_per_1k_input, cost_per_1k_output, max_requests_per_minute
			FROM llm_providers
			WHERE tenant_id=$1 AND is_active=TRUE
			ORDER BY is_default DESC, id ASC`, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var cfg ProviderConfig
			if err := rows.Scan(&cfg.ID, &cfg.ProviderName, &cfg.APIKey, &cfg.ModelName, &cfg.Temperature, &cfg.MaxTokens, &cfg.CostPer1KInput, &cfg.CostPer1KOutput, &cfg.MaxRequestsPerMinute); err != nil {
				return err
			}
			if decrypted, err := crypto.Decrypt(s.MasterKey, cfg.APIKey); err == nil {
				cfg.APIKey = decrypted
			}
			configs = append(configs, cfg)
		}
		return rows.Err()
	})
	return configs, err
}

func (s *Store) GetDefaultProvider(ctx context.Context, tenantID int64) (*ProviderConfig, error) {
	var cfg ProviderConfig
	err := s.DB.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		row := conn.QueryRow(ctx, `
			SELECT id, provider_name, api_key, model_name, temperature, max_tokens, cost_per_1k_input, cost_per_1k_output, max_requests_per_minute
			FROM llm_providers
			WHERE tenant_id=$1 AND is_default=TRUE AND is_active=TRUE
			LIMIT 1`, tenantID)
		if err := row.Scan(&cfg.ID, &cfg.ProviderName, &cfg.APIKey, &cfg.ModelName, &cfg.Temperature, &cfg.MaxTokens, &cfg.CostPer1KInput, &cfg.CostPer1KOutput, &cfg.MaxRequestsPerMinute); err != nil {
			return err
		}
		if decrypted, err := crypto.Decrypt(s.MasterKey, cfg.APIKey); err == nil {
			cfg.APIKey = decrypted
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (s *Store) GetProviderByID(ctx context.Context, tenantID int64, providerID int64) (*ProviderConfig, error) {
	var cfg ProviderConfig
	err := s.DB.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		row := conn.QueryRow(ctx, `
			SELECT id, provider_name, api_key, model_name, temperature, max_tokens, cost_per_1k_input, cost_per_1k_output, max_requests_per_minute
			FROM llm_providers
			WHERE tenant_id=$1 AND id=$2`, tenantID, providerID)
		if err := row.Scan(&cfg.ID, &cfg.ProviderName, &cfg.APIKey, &cfg.ModelName, &cfg.Temperature, &cfg.MaxTokens, &cfg.CostPer1KInput, &cfg.CostPer1KOutput, &cfg.MaxRequestsPerMinute); err != nil {
			return err
		}
		if decrypted, err := crypto.Decrypt(s.MasterKey, cfg.APIKey); err == nil {
			cfg.APIKey = decrypted
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (s *Store) InsertUsage(ctx context.Context, tenantID, providerID int64, messageID *int64, record UsageRecord, costIn, costOut float64) error {
	return s.DB.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		_, err := conn.Exec(ctx, `
			INSERT INTO llm_usage_logs (tenant_id, provider_id, message_id, input_tokens, output_tokens, total_tokens, input_cost, output_cost, total_cost, response_time_ms, success, error_message, feature_used, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
			tenantID, providerID, messageID, record.InputTokens, record.OutputTokens, record.TotalTokens,
			record.InputCost(costIn), record.OutputCost(costOut), record.TotalCost(costIn, costOut), record.Latency.Milliseconds(), record.Success, record.ErrorMessage, record.Feature, time.Now().UTC())
		return err
	})
}

func (s *Store) InsertHealth(ctx context.Context, tenantID, providerID int64, status string, latency time.Duration, errorMessage *string, httpStatus *int) error {
	return s.DB.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		_, err := conn.Exec(ctx, `
			INSERT INTO llm_provider_health (provider_id, tenant_id, check_time, status, latency_ms, error_message, http_status_code, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			providerID, tenantID, time.Now().UTC(), status, latency.Milliseconds(), errorMessage, httpStatus, time.Now().UTC())
		if err != nil {
			return err
		}
		_, err = conn.Exec(ctx, `
			UPDATE llm_providers
			SET health_status=$1, last_health_check=$2
			WHERE id=$3 AND tenant_id=$4`, status, time.Now().UTC(), providerID, tenantID)
		return err
	})
}

func (s *Store) RecentHealthFailures(ctx context.Context, tenantID, providerID int64) (int, error) {
	failures := 0
	err := s.DB.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `
			SELECT status FROM llm_provider_health
			WHERE provider_id=$1 AND tenant_id=$2
			ORDER BY check_time DESC
			LIMIT 3`, providerID, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var status string
			if err := rows.Scan(&status); err != nil {
				return err
			}
			if status != "ok" {
				failures++
			}
		}
		return rows.Err()
	})
	return failures, err
}

func (s *Store) ListProviderIDs(ctx context.Context, tenantID int64) ([]int64, error) {
	var ids []int64
	err := s.DB.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `SELECT id FROM llm_providers WHERE tenant_id=$1 AND is_active=TRUE`, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				return err
			}
			ids = append(ids, id)
		}
		return rows.Err()
	})
	return ids, err
}

func (s *Store) SetProviderHealth(ctx context.Context, tenantID, providerID int64, status string) error {
	return s.DB.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		_, err := conn.Exec(ctx, `
			UPDATE llm_providers
			SET health_status=$1, last_health_check=$2
			WHERE id=$3 AND tenant_id=$4`, status, time.Now().UTC(), providerID, tenantID)
		return err
	})
}
