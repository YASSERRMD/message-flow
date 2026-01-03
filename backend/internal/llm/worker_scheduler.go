package llm

import (
	"context"
	"sync"

	"message-flow/backend/internal/db"
	"message-flow/backend/internal/realtime"
)

type WorkerScheduler struct {
	mu      sync.Mutex
	queue   *Queue
	service *Service
	store   *db.Store
	hub     *realtime.Hub
	workers map[int64]context.CancelFunc
}

func NewWorkerScheduler(queue *Queue, service *Service, store *db.Store, hub *realtime.Hub) *WorkerScheduler {
	return &WorkerScheduler{
		queue:   queue,
		service: service,
		store:   store,
		hub:     hub,
		workers: map[int64]context.CancelFunc{},
	}
}

func (s *WorkerScheduler) EnsureTenant(ctx context.Context, tenantID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.workers[tenantID]; ok {
		return
	}
	workerCtx, cancel := context.WithCancel(ctx)
	s.workers[tenantID] = cancel
	worker := &Worker{Queue: s.queue, Service: s.service, DB: s.store, Hub: s.hub, BatchSize: 100}
	go worker.Start(workerCtx, tenantID)
}
