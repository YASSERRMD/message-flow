package llm

import (
	"context"
	"sync"
)

type HealthScheduler struct {
	mu      sync.Mutex
	monitor *HealthMonitor
	store   *Store
	workers map[int64]context.CancelFunc
}

func NewHealthScheduler(monitor *HealthMonitor, store *Store) *HealthScheduler {
	return &HealthScheduler{monitor: monitor, store: store, workers: map[int64]context.CancelFunc{}}
}

func (s *HealthScheduler) EnsureTenant(ctx context.Context, tenantID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.workers[tenantID]; ok {
		return
	}
	workerCtx, cancel := context.WithCancel(ctx)
	s.workers[tenantID] = cancel
	go s.run(workerCtx, tenantID)
}

func (s *HealthScheduler) run(ctx context.Context, tenantID int64) {
	s.monitor.Run(ctx, tenantID)
}
