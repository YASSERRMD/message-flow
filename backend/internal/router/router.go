package router

import (
	"net/http"
	"strings"

	"message-flow/backend/internal/handlers"
)

type Router struct {
	api *handlers.API
}

func New(api *handlers.API) *Router {
	return &Router{api: api}
}

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if handleCORS(w, r) {
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")
	if path == "" {
		path = "/"
	}

	switch {
	case path == "/api/v1/dashboard":
		if r.Method == http.MethodGet {
			rt.api.GetDashboard(w, r)
			return
		}
	case path == "/api/v1/dashboard/stream":
		if r.Method == http.MethodGet {
			rt.api.StreamDashboard(w, r)
			return
		}
	case path == "/api/v1/conversations":
		if r.Method == http.MethodGet {
			rt.api.ListConversations(w, r)
			return
		}
	case strings.HasPrefix(path, "/api/v1/conversations/"):
		segments := strings.Split(strings.TrimPrefix(path, "/api/v1/conversations/"), "/")
		if len(segments) == 2 && segments[1] == "messages" {
			if r.Method == http.MethodGet {
				if id, ok := handlers.ParseID(segments[0]); ok {
					rt.api.GetConversationMessages(w, r, id)
					return
				}
			}
		}
	case path == "/api/v1/messages/reply":
		if r.Method == http.MethodPost {
			rt.api.ReplyMessage(w, r)
			return
		}
	case path == "/api/v1/messages/forward":
		if r.Method == http.MethodPost {
			rt.api.ForwardMessage(w, r)
			return
		}
	case path == "/api/v1/important-messages":
		if r.Method == http.MethodGet {
			rt.api.ListImportantMessages(w, r)
			return
		}
	case path == "/api/v1/action-items":
		switch r.Method {
		case http.MethodGet:
			rt.api.ListActionItems(w, r)
			return
		case http.MethodPost:
			rt.api.CreateActionItem(w, r)
			return
		}
	case strings.HasPrefix(path, "/api/v1/action-items/"):
		idPart := strings.TrimPrefix(path, "/api/v1/action-items/")
		if id, ok := handlers.ParseID(idPart); ok {
			switch r.Method {
			case http.MethodPatch:
				rt.api.UpdateActionItem(w, r, id)
				return
			case http.MethodDelete:
				rt.api.DeleteActionItem(w, r, id)
				return
			}
		}
	case path == "/api/v1/daily-summary":
		if r.Method == http.MethodGet {
			rt.api.GetDailySummary(w, r)
			return
		}
	case path == "/api/v1/auth/login":
		if r.Method == http.MethodPost {
			rt.api.Login(w, r)
			return
		}
	case path == "/api/v1/auth/register":
		if r.Method == http.MethodPost {
			rt.api.Register(w, r)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("{" + "\"error\":\"not found\"}"))
}

func handleCORS(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Tenant-ID")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return true
	}
	return false
}
