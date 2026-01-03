package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"message-flow/backend/internal/auth"
	"message-flow/backend/internal/models"
)

type createActionItemRequest struct {
	ConversationID int64   `json:"conversation_id"`
	Description    string  `json:"description"`
	Status         string  `json:"status"`
	AssignedTo     *int64  `json:"assigned_to"`
	DueDate        *string `json:"due_date"`
	Watchers       []int64 `json:"watchers"`
}

type updateActionItemRequest struct {
	Description *string `json:"description"`
	Status      *string `json:"status"`
	AssignedTo  *int64  `json:"assigned_to"`
	DueDate     *string `json:"due_date"`
	Watchers    []int64 `json:"watchers"`
}

func (a *API) CreateActionItem(w http.ResponseWriter, r *http.Request) {
	var req createActionItemRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.ConversationID == 0 || req.Description == "" {
		writeError(w, http.StatusBadRequest, "conversation_id and description are required")
		return
	}
	status := req.Status
	if status == "" {
		status = "new"
	}

	var dueDate *time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		parsed, err := time.Parse("2006-01-02", *req.DueDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "due_date must be YYYY-MM-DD")
			return
		}
		dueDate = &parsed
	}

	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var watchersJSON *string
	if len(req.Watchers) > 0 {
		raw, err := json.Marshal(req.Watchers)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid watchers list")
			return
		}
		value := string(raw)
		watchersJSON = &value
	}

	var item models.ActionItem
	query := `
		INSERT INTO action_items (tenant_id, conversation_id, description, status, assigned_to, due_date, watchers_json, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, tenant_id, conversation_id, description, status, assigned_to, due_date, watchers_json, created_at`

	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		return conn.QueryRow(ctx, query, tenantID, req.ConversationID, req.Description, status, req.AssignedTo, dueDate, watchersJSON, time.Now().UTC()).Scan(
			&item.ID, &item.TenantID, &item.ConversationID, &item.Description, &item.Status, &item.AssignedTo, &item.DueDate, &item.WatchersJSON, &item.CreatedAt,
		)
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create action item")
		return
	}

	if user, ok := auth.UserFromContext(r.Context()); ok {
		a.logActivity(ctx, tenantID, user, "action_item.create", map[string]any{
			"action_item_id":  item.ID,
			"conversation_id": item.ConversationID,
		})
	}
	a.logAudit(ctx, r, tenantID, authUserIDPtr(r), "action_item.create", stringPtr("action_item"), &item.ID, nil, map[string]any{
		"conversation_id": item.ConversationID,
		"status":          item.Status,
	})
	if item.AssignedTo != nil {
		a.createNotification(ctx, tenantID, *item.AssignedTo, "action_item.assigned", "You have been assigned a new action item.")
	}
	if a.Hub != nil {
		a.Hub.Broadcast(tenantID, map[string]any{
			"type": "action_item.create",
			"id":   item.ID,
		})
	}

	writeJSON(w, http.StatusCreated, item)
}

func (a *API) UpdateActionItem(w http.ResponseWriter, r *http.Request, actionItemID int64) {
	var req updateActionItemRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	var dueDate *time.Time
	if req.DueDate != nil {
		if *req.DueDate == "" {
			dueDate = nil
		} else {
			parsed, err := time.Parse("2006-01-02", *req.DueDate)
			if err != nil {
				writeError(w, http.StatusBadRequest, "due_date must be YYYY-MM-DD")
				return
			}
			dueDate = &parsed
		}
	}

	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var watchersJSON *string
	if req.Watchers != nil {
		raw, err := json.Marshal(req.Watchers)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid watchers list")
			return
		}
		value := string(raw)
		watchersJSON = &value
	}

	var item models.ActionItem
	query := `
		UPDATE action_items
		SET description=COALESCE($1, description),
		    status=COALESCE($2, status),
		    assigned_to=COALESCE($3, assigned_to),
		    due_date=COALESCE($4, due_date),
		    watchers_json=COALESCE($5, watchers_json)
		WHERE id=$6 AND tenant_id=$7
		RETURNING id, tenant_id, conversation_id, description, status, assigned_to, due_date, watchers_json, created_at`

	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		return conn.QueryRow(ctx, query, req.Description, req.Status, req.AssignedTo, dueDate, watchersJSON, actionItemID, tenantID).Scan(
			&item.ID, &item.TenantID, &item.ConversationID, &item.Description, &item.Status, &item.AssignedTo, &item.DueDate, &item.WatchersJSON, &item.CreatedAt,
		)
	}); err != nil {
		writeError(w, http.StatusNotFound, "action item not found")
		return
	}

	if user, ok := auth.UserFromContext(r.Context()); ok {
		a.logActivity(ctx, tenantID, user, "action_item.update", map[string]any{
			"action_item_id":  item.ID,
			"conversation_id": item.ConversationID,
		})
	}
	a.logAudit(ctx, r, tenantID, authUserIDPtr(r), "action_item.update", stringPtr("action_item"), &item.ID, nil, map[string]any{
		"status": item.Status,
	})
	if item.AssignedTo != nil {
		a.createNotification(ctx, tenantID, *item.AssignedTo, "action_item.reassigned", "An action item has been assigned to you.")
	}
	if a.Hub != nil {
		a.Hub.Broadcast(tenantID, map[string]any{
			"type": "action_item.update",
			"id":   item.ID,
		})
	}

	writeJSON(w, http.StatusOK, item)
}

func (a *API) DeleteActionItem(w http.ResponseWriter, r *http.Request, actionItemID int64) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var rowsAffected int64
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		command, err := conn.Exec(ctx, `
			DELETE FROM action_items WHERE id=$1 AND tenant_id=$2`, actionItemID, tenantID)
		if err != nil {
			return err
		}
		rowsAffected = command.RowsAffected()
		return nil
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete action item")
		return
	}
	if rowsAffected == 0 {
		writeError(w, http.StatusNotFound, "action item not found")
		return
	}

	if user, ok := auth.UserFromContext(r.Context()); ok {
		a.logActivity(ctx, tenantID, user, "action_item.delete", map[string]any{
			"action_item_id": actionItemID,
		})
	}
	a.logAudit(ctx, r, tenantID, authUserIDPtr(r), "action_item.delete", stringPtr("action_item"), &actionItemID, nil, nil)
	if a.Hub != nil {
		a.Hub.Broadcast(tenantID, map[string]any{
			"type": "action_item.delete",
			"id":   actionItemID,
		})
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (a *API) ListActionItems(w http.ResponseWriter, r *http.Request) {
	tenantID := a.tenantID(r)
	page, limit := parsePagination(r)
	offset := (page - 1) * limit

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items := []models.ActionItem{}
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `
			SELECT id, tenant_id, conversation_id, description, status, assigned_to, due_date, watchers_json, created_at
			FROM action_items
			WHERE tenant_id=$1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3`, tenantID, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var item models.ActionItem
			if err := rows.Scan(&item.ID, &item.TenantID, &item.ConversationID, &item.Description, &item.Status, &item.AssignedTo, &item.DueDate, &item.WatchersJSON, &item.CreatedAt); err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list action items")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  items,
		"page":  page,
		"limit": limit,
	})
}
