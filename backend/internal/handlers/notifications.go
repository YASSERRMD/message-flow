package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"message-flow/backend/internal/models"
)

type listNotificationsRequest struct {
	Read  *bool `json:"read"`
	Limit *int  `json:"limit"`
}

func (a *API) ListNotifications(w http.ResponseWriter, r *http.Request) {
	var req listNotificationsRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	tenantID := a.tenantID(r)
	userID := authUserIDPtr(r)
	if userID == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := 50
	if req.Limit != nil && *req.Limit > 0 && *req.Limit <= 200 {
		limit = *req.Limit
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items := []models.Notification{}
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `
			SELECT id, tenant_id, user_id, type, content, read, created_at
			FROM notifications
			WHERE tenant_id=$1 AND user_id=$2
			  AND ($3::BOOLEAN IS NULL OR read = $3)
			ORDER BY created_at DESC
			LIMIT $4`, tenantID, *userID, req.Read, limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item models.Notification
			if err := rows.Scan(&item.ID, &item.TenantID, &item.UserID, &item.Type, &item.Content, &item.Read, &item.CreatedAt); err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load notifications")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (a *API) MarkNotificationRead(w http.ResponseWriter, r *http.Request, notificationID int64) {
	tenantID := a.tenantID(r)
	userID := authUserIDPtr(r)
	if userID == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		command, err := conn.Exec(ctx, `
			UPDATE notifications
			SET read=TRUE
			WHERE tenant_id=$1 AND user_id=$2 AND id=$3`, tenantID, *userID, notificationID)
		if err != nil {
			return err
		}
		if command.RowsAffected() == 0 {
			return errNotFound
		}
		return nil
	}); err != nil {
		writeError(w, http.StatusNotFound, "notification not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "read"})
}

func (a *API) createNotification(ctx context.Context, tenantID, userID int64, notifType, content string) {
	_ = a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		_, err := conn.Exec(ctx, `
			INSERT INTO notifications (tenant_id, user_id, type, content, read, created_at)
			VALUES ($1,$2,$3,$4,FALSE,$5)`, tenantID, userID, notifType, content, time.Now().UTC())
		return err
	})
}
