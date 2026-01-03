package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"message-flow/backend/internal/models"
)

type createCommentRequest struct {
	Content string `json:"content"`
}

func (a *API) ListActionItemComments(w http.ResponseWriter, r *http.Request, actionItemID int64) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items := []models.Comment{}
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `
			SELECT id, tenant_id, action_item_id, user_id, content, created_at
			FROM comments
			WHERE tenant_id=$1 AND action_item_id=$2
			ORDER BY created_at ASC`, tenantID, actionItemID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item models.Comment
			if err := rows.Scan(&item.ID, &item.TenantID, &item.ActionItemID, &item.UserID, &item.Content, &item.CreatedAt); err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load comments")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (a *API) CreateActionItemComment(w http.ResponseWriter, r *http.Request, actionItemID int64) {
	var req createCommentRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}

	tenantID := a.tenantID(r)
	userID := authUserIDPtr(r)
	if userID == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var comment models.Comment
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		query := `
			INSERT INTO comments (tenant_id, action_item_id, user_id, content, created_at)
			VALUES ($1,$2,$3,$4,$5)
			RETURNING id, tenant_id, action_item_id, user_id, content, created_at`
		return conn.QueryRow(ctx, query, tenantID, actionItemID, *userID, req.Content, time.Now().UTC()).Scan(
			&comment.ID, &comment.TenantID, &comment.ActionItemID, &comment.UserID, &comment.Content, &comment.CreatedAt,
		)
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add comment")
		return
	}

	a.logAudit(ctx, r, tenantID, userID, "comment.create", stringPtr("comment"), &comment.ID, nil, map[string]any{
		"action_item_id": actionItemID,
	})
	writeJSON(w, http.StatusCreated, comment)
}

func (a *API) DeleteComment(w http.ResponseWriter, r *http.Request, commentID int64) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		command, err := conn.Exec(ctx, `DELETE FROM comments WHERE tenant_id=$1 AND id=$2`, tenantID, commentID)
		if err != nil {
			return err
		}
		if command.RowsAffected() == 0 {
			return errNotFound
		}
		return nil
	}); err != nil {
		writeError(w, http.StatusNotFound, "comment not found")
		return
	}

	a.logAudit(ctx, r, tenantID, authUserIDPtr(r), "comment.delete", stringPtr("comment"), &commentID, nil, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
