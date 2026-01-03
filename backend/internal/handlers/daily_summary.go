package handlers

import (
	"context"
	"net/http"
	"time"

	"message-flow/backend/internal/models"
)

func (a *API) GetDailySummary(w http.ResponseWriter, r *http.Request) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var summary models.DailySummary
	query := `
		SELECT id, tenant_id, conversation_id, summary_text, key_points_json, created_at
		FROM daily_summaries
		WHERE tenant_id=$1
		ORDER BY created_at DESC
		LIMIT 1`

	if err := a.Store.Pool.QueryRow(ctx, query, tenantID).Scan(
		&summary.ID, &summary.TenantID, &summary.ConversationID, &summary.SummaryText, &summary.KeyPointsJSON, &summary.CreatedAt,
	); err != nil {
		writeError(w, http.StatusNotFound, "no summary available")
		return
	}

	writeJSON(w, http.StatusOK, summary)
}
