package handlers

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"message-flow/backend/internal/auth"
	"message-flow/backend/internal/models"
)

type auditLogRequest struct {
	UserID    *int64  `json:"user_id"`
	Action    *string `json:"action"`
	Resource  *string `json:"resource_type"`
	StartDate *string `json:"start_date"`
	EndDate   *string `json:"end_date"`
	Limit     *int    `json:"limit"`
}

func (a *API) logAudit(ctx context.Context, r *http.Request, tenantID int64, userID *int64, action string, resourceType *string, resourceID *int64, before any, after any) {
	beforeJSON := marshalOptional(before)
	afterJSON := marshalOptional(after)
	ip := clientIP(r)
	ua := r.UserAgent()

	_ = a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		_, err := conn.Exec(ctx, `
			INSERT INTO audit_logs (tenant_id, user_id, action, resource_type, resource_id, changes_before_json, changes_after_json, ip_address, user_agent, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, tenantID, userID, action, resourceType, resourceID, beforeJSON, afterJSON, ip, ua, time.Now().UTC())
		return err
	})
}

func (a *API) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	var req auditLogRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	limit := 100
	if req.Limit != nil && *req.Limit > 0 && *req.Limit <= 500 {
		limit = *req.Limit
	}

	var start *time.Time
	if req.StartDate != nil && *req.StartDate != "" {
		parsed, err := time.Parse("2006-01-02", *req.StartDate)
		if err == nil {
			start = &parsed
		}
	}
	var end *time.Time
	if req.EndDate != nil && *req.EndDate != "" {
		parsed, err := time.Parse("2006-01-02", *req.EndDate)
		if err == nil {
			value := parsed.Add(24 * time.Hour)
			end = &value
		}
	}

	items := []models.AuditLog{}
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `
			SELECT id, tenant_id, user_id, action, resource_type, resource_id, changes_before_json, changes_after_json, ip_address, user_agent, created_at
			FROM audit_logs
			WHERE tenant_id=$1
			  AND ($2::BIGINT IS NULL OR user_id = $2)
			  AND ($3::TEXT IS NULL OR action = $3)
			  AND ($4::TEXT IS NULL OR resource_type = $4)
			  AND ($5::TIMESTAMPTZ IS NULL OR created_at >= $5)
			  AND ($6::TIMESTAMPTZ IS NULL OR created_at < $6)
			ORDER BY created_at DESC
			LIMIT $7`, tenantID, req.UserID, req.Action, req.Resource, start, end, limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item models.AuditLog
			if err := rows.Scan(&item.ID, &item.TenantID, &item.UserID, &item.Action, &item.ResourceType, &item.ResourceID, &item.ChangesBeforeJSON, &item.ChangesAfterJSON, &item.IPAddress, &item.UserAgent, &item.CreatedAt); err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load audit logs")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func marshalOptional(value any) *string {
	if value == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	text := string(raw)
	return &text
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func authUserIDPtr(r *http.Request) *int64 {
	if user, ok := auth.UserFromContext(r.Context()); ok {
		return &user.ID
	}
	return nil
}
