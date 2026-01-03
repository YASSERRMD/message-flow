package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"message-flow/backend/internal/models"
)

type workflowRequest struct {
	Name    string          `json:"name"`
	Trigger string          `json:"trigger"`
	Actions json.RawMessage `json:"actions"`
	Enabled *bool           `json:"enabled"`
}

func (a *API) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var req workflowRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Name == "" || req.Trigger == "" || len(req.Actions) == 0 {
		writeError(w, http.StatusBadRequest, "name, trigger, and actions are required")
		return
	}

	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	var workflow models.Workflow
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		query := `
			INSERT INTO workflows (tenant_id, name, trigger, actions_json, enabled, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
			RETURNING id, tenant_id, name, trigger, actions_json, enabled, created_at, updated_at`
		return conn.QueryRow(ctx, query, tenantID, req.Name, req.Trigger, string(req.Actions), enabled, time.Now().UTC(), time.Now().UTC()).Scan(
			&workflow.ID, &workflow.TenantID, &workflow.Name, &workflow.Trigger, &workflow.Actions, &workflow.Enabled, &workflow.CreatedAt, &workflow.UpdatedAt,
		)
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create workflow")
		return
	}

	a.logAudit(ctx, r, tenantID, authUserIDPtr(r), "workflow.create", stringPtr("workflow"), &workflow.ID, nil, map[string]any{
		"name": workflow.Name,
	})
	writeJSON(w, http.StatusCreated, workflow)
}

func (a *API) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items := []models.Workflow{}
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `
			SELECT id, tenant_id, name, trigger, actions_json, enabled, created_at, updated_at
			FROM workflows
			WHERE tenant_id=$1
			ORDER BY created_at DESC`, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item models.Workflow
			if err := rows.Scan(&item.ID, &item.TenantID, &item.Name, &item.Trigger, &item.Actions, &item.Enabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list workflows")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (a *API) UpdateWorkflow(w http.ResponseWriter, r *http.Request, workflowID int64) {
	var req workflowRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var workflow models.Workflow
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		query := `
			UPDATE workflows
			SET name=COALESCE(NULLIF($1, ''), name),
			    trigger=COALESCE(NULLIF($2, ''), trigger),
			    actions_json=COALESCE(NULLIF($3, ''), actions_json),
			    enabled=COALESCE($4, enabled),
			    updated_at=$5
			WHERE tenant_id=$6 AND id=$7
			RETURNING id, tenant_id, name, trigger, actions_json, enabled, created_at, updated_at`
		actions := ""
		if len(req.Actions) > 0 {
			actions = string(req.Actions)
		}
		return conn.QueryRow(ctx, query, req.Name, req.Trigger, actions, req.Enabled, time.Now().UTC(), tenantID, workflowID).Scan(
			&workflow.ID, &workflow.TenantID, &workflow.Name, &workflow.Trigger, &workflow.Actions, &workflow.Enabled, &workflow.CreatedAt, &workflow.UpdatedAt,
		)
	}); err != nil {
		writeError(w, http.StatusNotFound, "workflow not found")
		return
	}

	a.logAudit(ctx, r, tenantID, authUserIDPtr(r), "workflow.update", stringPtr("workflow"), &workflowID, nil, map[string]any{
		"name": workflow.Name,
	})
	writeJSON(w, http.StatusOK, workflow)
}

func (a *API) DeleteWorkflow(w http.ResponseWriter, r *http.Request, workflowID int64) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		command, err := conn.Exec(ctx, `DELETE FROM workflows WHERE tenant_id=$1 AND id=$2`, tenantID, workflowID)
		if err != nil {
			return err
		}
		if command.RowsAffected() == 0 {
			return errNotFound
		}
		return nil
	}); err != nil {
		writeError(w, http.StatusNotFound, "workflow not found")
		return
	}

	a.logAudit(ctx, r, tenantID, authUserIDPtr(r), "workflow.delete", stringPtr("workflow"), &workflowID, nil, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (a *API) TestWorkflow(w http.ResponseWriter, r *http.Request, workflowID int64) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var execution models.WorkflowExecution
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		query := `
			INSERT INTO workflow_executions (workflow_id, execution_time, success, created_at)
			VALUES ($1,$2,$3,$4)
			RETURNING id, workflow_id, execution_time, success, error_message, created_at`
		return conn.QueryRow(ctx, query, workflowID, time.Now().UTC(), true, time.Now().UTC()).Scan(
			&execution.ID, &execution.WorkflowID, &execution.ExecutionTime, &execution.Success, &execution.ErrorMessage, &execution.CreatedAt,
		)
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to test workflow")
		return
	}

	a.logAudit(ctx, r, tenantID, authUserIDPtr(r), "workflow.test", stringPtr("workflow"), &workflowID, nil, map[string]any{
		"execution_id": execution.ID,
	})
	writeJSON(w, http.StatusOK, execution)
}

func (a *API) GetWorkflowExecutions(w http.ResponseWriter, r *http.Request, workflowID int64) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items := []models.WorkflowExecution{}
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `
			SELECT id, workflow_id, execution_time, success, error_message, created_at
			FROM workflow_executions
			WHERE workflow_id=$1
			ORDER BY execution_time DESC
			LIMIT 50`, workflowID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item models.WorkflowExecution
			if err := rows.Scan(&item.ID, &item.WorkflowID, &item.ExecutionTime, &item.Success, &item.ErrorMessage, &item.CreatedAt); err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load executions")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}
