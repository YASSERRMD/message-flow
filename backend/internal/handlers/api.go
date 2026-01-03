package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"message-flow/backend/internal/auth"
	"message-flow/backend/internal/db"
	"message-flow/backend/internal/llm"
	"message-flow/backend/internal/realtime"
	"message-flow/backend/internal/whatsapp"
)

type API struct {
	Store           *db.Store
	Auth            *auth.Service
	Hub             *realtime.Hub
	LLM             *llm.Service
	LLMStore        *llm.Store
	Queue           *llm.Queue
	HealthScheduler *llm.HealthScheduler
	WorkerScheduler *llm.WorkerScheduler
	WhatsApp        *whatsapp.Manager
}

func NewAPI(store *db.Store, authService *auth.Service, hub *realtime.Hub, llmService *llm.Service, llmStore *llm.Store, queue *llm.Queue, scheduler *llm.HealthScheduler, workerScheduler *llm.WorkerScheduler, waManager *whatsapp.Manager) *API {
	return &API{Store: store, Auth: authService, Hub: hub, LLM: llmService, LLMStore: llmStore, Queue: queue, HealthScheduler: scheduler, WorkerScheduler: workerScheduler, WhatsApp: waManager}
}

func (a *API) tenantID(r *http.Request) int64 {
	if user, ok := auth.UserFromContext(r.Context()); ok {
		return user.TenantID
	}
	value := r.Header.Get("X-Tenant-ID")
	if value == "" {
		return 1
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 1
	}
	return id
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func readJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func ParseID(pathPart string) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(pathPart), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func parsePagination(r *http.Request) (int, int) {
	page := 1
	limit := 20
	if value := r.URL.Query().Get("page"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if value := r.URL.Query().Get("limit"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	return page, limit
}
