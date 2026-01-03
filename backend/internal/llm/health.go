package llm

import (
	"context"
	"time"
)

type HealthMonitor struct {
	Router *Router
	Store  *Store
}

func (h *HealthMonitor) Run(ctx context.Context, tenantID int64) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	h.runOnce(ctx, tenantID)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.runOnce(ctx, tenantID)
		}
	}
}

func (h *HealthMonitor) runOnce(ctx context.Context, tenantID int64) {
	providerIDs, err := h.Store.ListProviderIDs(ctx, tenantID)
	if err != nil {
		return
	}
	for _, providerID := range providerIDs {
		provider, err := h.Router.GetProvider(ctx, tenantID, providerID)
		if err != nil {
			continue
		}
		result, err := provider.HealthCheck(ctx)
		status := "ok"
		if err != nil || result == nil {
			status = "error"
		} else if result.Latency > 3*time.Second {
			status = "slow"
		}
		var errMsg *string
		if err != nil {
			msg := err.Error()
			errMsg = &msg
		}
		latency := time.Duration(0)
		if result != nil {
			latency = result.Latency
		}
		_ = h.Store.InsertHealth(ctx, tenantID, providerID, status, latency, errMsg, nil)
		if status == "error" {
			failures, err := h.Store.RecentHealthFailures(ctx, tenantID, providerID)
			if err == nil && failures >= 3 {
				_ = h.Store.SetProviderHealth(ctx, tenantID, providerID, "unhealthy")
			}
		}
	}
}
