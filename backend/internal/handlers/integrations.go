package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"message-flow/backend/internal/models"
)

type integrationRequest struct {
	Config map[string]any `json:"config"`
}

func (a *API) ConnectIntegration(w http.ResponseWriter, r *http.Request, integrationType string) {
	var req integrationRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if len(req.Config) == 0 {
		writeError(w, http.StatusBadRequest, "config is required")
		return
	}

	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	raw, err := json.Marshal(req.Config)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid config")
		return
	}

	var integration models.Integration
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		query := `
			INSERT INTO integrations (tenant_id, type, config_json, status, created_at, updated_at)
			VALUES ($1,$2,$3,'active',$4,$5)
			ON CONFLICT (tenant_id, type)
			DO UPDATE SET config_json=EXCLUDED.config_json, status='active', updated_at=EXCLUDED.updated_at
			RETURNING id, tenant_id, type, config_json, status, created_at, updated_at`
		return conn.QueryRow(ctx, query, tenantID, integrationType, string(raw), time.Now().UTC(), time.Now().UTC()).Scan(
			&integration.ID, &integration.TenantID, &integration.Type, &integration.Config, &integration.Status, &integration.CreatedAt, &integration.UpdatedAt,
		)
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to connect integration")
		return
	}

	a.logAudit(ctx, r, tenantID, authUserIDPtr(r), "integration.connected", stringPtr("integration"), &integration.ID, nil, map[string]any{
		"type": integrationType,
	})
	writeJSON(w, http.StatusCreated, integration)
}

func (a *API) ListIntegrations(w http.ResponseWriter, r *http.Request) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items := []models.Integration{}
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `
			SELECT id, tenant_id, type, config_json, status, created_at, updated_at
			FROM integrations
			WHERE tenant_id=$1
			ORDER BY created_at DESC`, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item models.Integration
			if err := rows.Scan(&item.ID, &item.TenantID, &item.Type, &item.Config, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list integrations")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (a *API) GetIntegrationConfig(w http.ResponseWriter, r *http.Request, integrationID int64) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var integration models.Integration
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		query := `
			SELECT id, tenant_id, type, config_json, status, created_at, updated_at
			FROM integrations
			WHERE tenant_id=$1 AND id=$2`
		return conn.QueryRow(ctx, query, tenantID, integrationID).Scan(
			&integration.ID, &integration.TenantID, &integration.Type, &integration.Config, &integration.Status, &integration.CreatedAt, &integration.UpdatedAt,
		)
	}); err != nil {
		writeError(w, http.StatusNotFound, "integration not found")
		return
	}

	writeJSON(w, http.StatusOK, integration)
}

func (a *API) DisconnectIntegration(w http.ResponseWriter, r *http.Request, integrationID int64) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		command, err := conn.Exec(ctx, `DELETE FROM integrations WHERE tenant_id=$1 AND id=$2`, tenantID, integrationID)
		if err != nil {
			return err
		}
		if command.RowsAffected() == 0 {
			return errNotFound
		}
		return nil
	}); err != nil {
		writeError(w, http.StatusNotFound, "integration not found")
		return
	}

	a.logAudit(ctx, r, tenantID, authUserIDPtr(r), "integration.disconnected", stringPtr("integration"), &integrationID, nil, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (a *API) ReceiveWebhook(w http.ResponseWriter, r *http.Request) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	secret, err := a.lookupWebhookSecret(ctx, tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "webhook not configured")
		return
	}
	if secret != "" {
		signature := r.Header.Get("X-Webhook-Signature")
		if signature == "" || !verifyHMAC(body, signature, secret) {
			writeError(w, http.StatusUnauthorized, "invalid signature")
			return
		}
	}

	a.logAudit(ctx, r, tenantID, authUserIDPtr(r), "webhook.received", stringPtr("webhook"), nil, nil, map[string]any{
		"size": len(body),
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "received"})
}

func (a *API) lookupWebhookSecret(ctx context.Context, tenantID int64) (string, error) {
	var config string
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		return conn.QueryRow(ctx, `
			SELECT config_json
			FROM integrations
			WHERE tenant_id=$1 AND type='webhook'
			ORDER BY updated_at DESC
			LIMIT 1`, tenantID).Scan(&config)
	}); err != nil {
		return "", err
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(config), &payload); err != nil {
		return "", nil
	}
	if value, ok := payload["secret"].(string); ok {
		return value, nil
	}
	return "", nil
}

func verifyHMAC(payload []byte, signature, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expected))
}
