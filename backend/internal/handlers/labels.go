package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"message-flow/backend/internal/models"
)

type createLabelRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type addLabelRequest struct {
	LabelID int64 `json:"label_id"`
}

func (a *API) CreateLabel(w http.ResponseWriter, r *http.Request) {
	var req createLabelRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Name == "" || req.Color == "" {
		writeError(w, http.StatusBadRequest, "name and color are required")
		return
	}

	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var label models.Label
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		query := `
			INSERT INTO labels (tenant_id, name, color, created_at)
			VALUES ($1,$2,$3,$4)
			RETURNING id, tenant_id, name, color, created_at`
		return conn.QueryRow(ctx, query, tenantID, req.Name, req.Color, time.Now().UTC()).Scan(
			&label.ID, &label.TenantID, &label.Name, &label.Color, &label.CreatedAt,
		)
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create label")
		return
	}

	a.logAudit(ctx, r, tenantID, authUserIDPtr(r), "label.create", stringPtr("label"), &label.ID, nil, map[string]any{
		"name": label.Name,
	})
	writeJSON(w, http.StatusCreated, label)
}

func (a *API) AddLabelToMessage(w http.ResponseWriter, r *http.Request, messageID int64) {
	var req addLabelRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.LabelID == 0 {
		writeError(w, http.StatusBadRequest, "label_id is required")
		return
	}

	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		_, err := conn.Exec(ctx, `
			INSERT INTO message_labels (message_id, label_id, tenant_id, created_at)
			VALUES ($1,$2,$3,$4)
			ON CONFLICT DO NOTHING`, messageID, req.LabelID, tenantID, time.Now().UTC())
		return err
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add label")
		return
	}

	a.logAudit(ctx, r, tenantID, authUserIDPtr(r), "message.label_added", stringPtr("message"), &messageID, nil, map[string]any{
		"label_id": req.LabelID,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "labeled"})
}
