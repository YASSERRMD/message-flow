package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"message-flow/backend/internal/models"
)

func (a *API) GetDashboard(w http.ResponseWriter, r *http.Request) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	summary := models.DashboardSummary{}

	queries := []struct {
		query string
		dest  *int64
	}{
		{"SELECT COUNT(*) FROM conversations WHERE tenant_id=$1", &summary.TotalConversations},
		{"SELECT COUNT(*) FROM messages WHERE tenant_id=$1", &summary.TotalMessages},
		{"SELECT COUNT(*) FROM important_messages WHERE tenant_id=$1", &summary.ImportantMessages},
		{"SELECT COUNT(*) FROM action_items WHERE tenant_id=$1 AND status NOT IN ('done','completed')", &summary.OpenActionItems},
	}

	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		for _, q := range queries {
			if err := conn.QueryRow(ctx, q.query, tenantID).Scan(q.dest); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load dashboard")
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

func (a *API) StreamDashboard(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			writer := &responseBuffer{w: w}
			a.GetDashboard(writer, r)
			if writer.status == http.StatusOK {
				_, _ = w.Write([]byte("data: " + writer.body + "\n\n"))
				flusher.Flush()
			}
		}
	}
}

type responseBuffer struct {
	w       http.ResponseWriter
	body    string
	status  int
	headers http.Header
}

func (rb *responseBuffer) Header() http.Header {
	if rb.headers == nil {
		rb.headers = http.Header{}
	}
	return rb.headers
}

func (rb *responseBuffer) WriteHeader(status int) {
	rb.status = status
}

func (rb *responseBuffer) Write(b []byte) (int, error) {
	rb.body = string(b)
	return len(b), nil
}
