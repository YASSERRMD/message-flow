package handlers

import (
	"context"
	"net/http"
	"time"

	"message-flow/backend/internal/models"
)

type createActionItemRequest struct {
	ConversationID int64   `json:"conversation_id"`
	Description    string  `json:"description"`
	Status         string  `json:"status"`
	AssignedTo     *int64  `json:"assigned_to"`
	DueDate        *string `json:"due_date"`
}

type updateActionItemRequest struct {
	Description *string `json:"description"`
	Status      *string `json:"status"`
	AssignedTo  *int64  `json:"assigned_to"`
	DueDate     *string `json:"due_date"`
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
		status = "open"
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

	var item models.ActionItem
	query := `
		INSERT INTO action_items (tenant_id, conversation_id, description, status, assigned_to, due_date, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, tenant_id, conversation_id, description, status, assigned_to, due_date, created_at`

	if err := a.Store.Pool.QueryRow(ctx, query, tenantID, req.ConversationID, req.Description, status, req.AssignedTo, dueDate, time.Now().UTC()).Scan(
		&item.ID, &item.TenantID, &item.ConversationID, &item.Description, &item.Status, &item.AssignedTo, &item.DueDate, &item.CreatedAt,
	); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create action item")
		return
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

	var item models.ActionItem
	query := `
		UPDATE action_items
		SET description=COALESCE($1, description),
		    status=COALESCE($2, status),
		    assigned_to=COALESCE($3, assigned_to),
		    due_date=COALESCE($4, due_date)
		WHERE id=$5 AND tenant_id=$6
		RETURNING id, tenant_id, conversation_id, description, status, assigned_to, due_date, created_at`

	if err := a.Store.Pool.QueryRow(ctx, query, req.Description, req.Status, req.AssignedTo, dueDate, actionItemID, tenantID).Scan(
		&item.ID, &item.TenantID, &item.ConversationID, &item.Description, &item.Status, &item.AssignedTo, &item.DueDate, &item.CreatedAt,
	); err != nil {
		writeError(w, http.StatusNotFound, "action item not found")
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (a *API) DeleteActionItem(w http.ResponseWriter, r *http.Request, actionItemID int64) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	command, err := a.Store.Pool.Exec(ctx, `
		DELETE FROM action_items WHERE id=$1 AND tenant_id=$2`, actionItemID, tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete action item")
		return
	}
	if command.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "action item not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (a *API) ListActionItems(w http.ResponseWriter, r *http.Request) {
	tenantID := a.tenantID(r)
	page, limit := parsePagination(r)
	offset := (page - 1) * limit

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := a.Store.Pool.Query(ctx, `
		SELECT id, tenant_id, conversation_id, description, status, assigned_to, due_date, created_at
		FROM action_items
		WHERE tenant_id=$1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`, tenantID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list action items")
		return
	}
	defer rows.Close()

	items := []models.ActionItem{}
	for rows.Next() {
		var item models.ActionItem
		if err := rows.Scan(&item.ID, &item.TenantID, &item.ConversationID, &item.Description, &item.Status, &item.AssignedTo, &item.DueDate, &item.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read action items")
			return
		}
		items = append(items, item)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  items,
		"page":  page,
		"limit": limit,
	})
}
