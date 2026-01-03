package router

import (
	"net/http"
	"strconv"
	"strings"

	"message-flow/backend/internal/auth"
	"message-flow/backend/internal/handlers"
	"message-flow/backend/internal/middleware"
	"message-flow/backend/internal/realtime"
)

type Router struct {
	api     *handlers.API
	auth    *auth.Service
	limiter *middleware.RateLimiter
	origin  string
	hub     *realtime.Hub
}

func New(api *handlers.API, authService *auth.Service, limiter *middleware.RateLimiter, origin string, hub *realtime.Hub) *Router {
	return &Router{api: api, auth: authService, limiter: limiter, origin: origin, hub: hub}
}

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if middleware.HandleCORS(w, r, rt.origin) {
		return
	}
	middleware.SecurityHeaders(w)

	path := strings.TrimSuffix(r.URL.Path, "/")
	if path == "" {
		path = "/"
	}

	if requiresAuth(path) {
		user, err := middleware.Authenticate(r, rt.auth)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("{\"error\":\"unauthorized\"}"))
			return
		}
		if rt.limiter != nil {
			key := "user:" + strconv.FormatInt(user.ID, 10)
			if !rt.limiter.Allow(key) {
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte("{\"error\":\"rate limit exceeded\"}"))
				return
			}
		}
		if err := middleware.ValidateCSRF(r, user); err != nil {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("{\"error\":\"invalid csrf token\"}"))
			return
		}
		r = r.WithContext(auth.WithUser(r.Context(), user))
	} else if rt.limiter != nil {
		key := middleware.ClientKey(r)
		if !rt.limiter.Allow(key) {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte("{\"error\":\"rate limit exceeded\"}"))
			return
		}
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
	case path == "/api/v1/ws":
		if r.Method == http.MethodGet && rt.hub != nil {
			user, err := middleware.Authenticate(r, rt.auth)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("{\"error\":\"unauthorized\"}"))
				return
			}
			realtime.ServeWS(w, r, rt.hub, user.TenantID)
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
	case path == "/api/v1/auth/me":
		if r.Method == http.MethodGet {
			rt.api.Me(w, r)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("{\"error\":\"not found\"}"))
}

func requiresAuth(path string) bool {
	switch path {
	case "/api/v1/auth/login", "/api/v1/auth/register":
		return false
	default:
		return strings.HasPrefix(path, "/api/v1/")
	}
}
