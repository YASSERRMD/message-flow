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

	if strings.HasPrefix(path, "/api/v1/") && path != "/api/v1/auth/login" && path != "/api/v1/auth/register" {
		required := handlers.RequiredRole(path, r.Method)
		if required != "" && !rt.api.Authorize(r, required) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("{\"error\":\"forbidden\"}"))
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
			realtime.ServeWS(w, r, rt.hub, user.TenantID, user.ID)
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
	case path == "/api/v1/messages/analyze":
		if r.Method == http.MethodPost {
			rt.api.AnalyzeMessage(w, r)
			return
		}
	case path == "/api/v1/messages/batch-analyze":
		if r.Method == http.MethodPost {
			rt.api.BatchAnalyze(w, r)
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
		segments := strings.Split(idPart, "/")
		if len(segments) == 2 && segments[1] == "comments" {
			if id, ok := handlers.ParseID(segments[0]); ok {
				switch r.Method {
				case http.MethodGet:
					rt.api.ListActionItemComments(w, r, id)
					return
				case http.MethodPost:
					rt.api.CreateActionItemComment(w, r, id)
					return
				}
			}
		} else if id, ok := handlers.ParseID(idPart); ok {
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
	case path == "/api/v1/conversations/summarize":
		if r.Method == http.MethodPost {
			rt.api.SummarizeConversation(w, r)
			return
		}
	case path == "/api/v1/llm/providers":
		switch r.Method {
		case http.MethodPost:
			rt.api.CreateProvider(w, r)
			return
		case http.MethodGet:
			rt.api.ListProviders(w, r)
			return
		}
	case path == "/api/v1/llm/providers/comparison":
		if r.Method == http.MethodGet {
			rt.api.GetProviderComparison(w, r)
			return
		}
	case strings.HasPrefix(path, "/api/v1/llm/providers/"):
		segments := strings.Split(strings.TrimPrefix(path, "/api/v1/llm/providers/"), "/")
		if len(segments) >= 1 {
			if id, ok := handlers.ParseID(segments[0]); ok {
				if len(segments) == 2 && segments[1] == "test" {
					if r.Method == http.MethodPost {
						rt.api.TestProvider(w, r, id)
						return
					}
				} else if len(segments) == 2 && segments[1] == "history" {
					if r.Method == http.MethodGet {
						rt.api.GetProviderHistory(w, r, id)
						return
					}
				} else {
					switch r.Method {
					case http.MethodGet:
						rt.api.GetProvider(w, r, id)
						return
					case http.MethodPatch:
						rt.api.UpdateProvider(w, r, id)
						return
					case http.MethodDelete:
						rt.api.DeleteProvider(w, r, id)
						return
					}
				}
			}
		}
	case path == "/api/v1/llm/usage":
		if r.Method == http.MethodGet {
			rt.api.GetUsageStats(w, r)
			return
		}
	case path == "/api/v1/llm/costs":
		if r.Method == http.MethodGet {
			rt.api.GetCosts(w, r)
			return
		}
	case path == "/api/v1/llm/analytics/cost-breakdown":
		if r.Method == http.MethodGet {
			rt.api.GetCostBreakdown(w, r)
			return
		}
	case path == "/api/v1/llm/analytics/usage-by-feature":
		if r.Method == http.MethodGet {
			rt.api.GetUsageByFeature(w, r)
			return
		}
	case path == "/api/v1/llm/health":
		if r.Method == http.MethodGet {
			rt.api.GetHealthStatus(w, r)
			return
		}
	case path == "/api/v1/llm/features":
		if r.Method == http.MethodGet {
			rt.api.GetFeatures(w, r)
			return
		}
	case strings.HasPrefix(path, "/api/v1/llm/features/"):
		segments := strings.Split(strings.TrimPrefix(path, "/api/v1/llm/features/"), "/")
		if len(segments) >= 1 {
			feature := segments[0]
			if len(segments) == 2 && segments[1] == "assign-provider" {
				if r.Method == http.MethodPost {
					rt.api.AssignProviderToFeature(w, r, feature)
					return
				}
			}
			if len(segments) == 2 && segments[1] == "providers" {
				if r.Method == http.MethodGet {
					rt.api.GetFeatureProviders(w, r, feature)
					return
				}
			}
			if len(segments) == 3 && segments[1] == "providers" {
				if r.Method == http.MethodDelete {
					if id, ok := handlers.ParseID(segments[2]); ok {
						rt.api.DeleteFeatureProvider(w, r, feature, id)
						return
					}
				}
			}
		}
	case path == "/api/v1/llm/bulk-test":
		if r.Method == http.MethodPost {
			rt.api.BulkTestProviders(w, r)
			return
		}
	case path == "/api/v1/llm/recommendations":
		if r.Method == http.MethodGet {
			rt.api.GetRecommendations(w, r)
			return
		}
	case path == "/api/v1/team/users":
		switch r.Method {
		case http.MethodPost:
			rt.api.AddTeamUser(w, r)
			return
		case http.MethodGet:
			rt.api.ListTeamUsers(w, r)
			return
		}
	case strings.HasPrefix(path, "/api/v1/team/users/"):
		segments := strings.Split(strings.TrimPrefix(path, "/api/v1/team/users/"), "/")
		if len(segments) == 2 && segments[1] == "role" {
			if r.Method == http.MethodPatch {
				if id, ok := handlers.ParseID(segments[0]); ok {
					rt.api.UpdateTeamUserRole(w, r, id)
					return
				}
			}
		} else if len(segments) == 1 {
			if r.Method == http.MethodDelete {
				if id, ok := handlers.ParseID(segments[0]); ok {
					rt.api.RemoveTeamUser(w, r, id)
					return
				}
			}
		}
	case path == "/api/v1/team/invitations":
		if r.Method == http.MethodPost {
			rt.api.SendInvitation(w, r)
			return
		}
	case path == "/api/v1/team/activity":
		if r.Method == http.MethodGet {
			rt.api.GetTeamActivity(w, r)
			return
		}
	case path == "/api/v1/workflows":
		switch r.Method {
		case http.MethodPost:
			rt.api.CreateWorkflow(w, r)
			return
		case http.MethodGet:
			rt.api.ListWorkflows(w, r)
			return
		}
	case strings.HasPrefix(path, "/api/v1/workflows/"):
		segments := strings.Split(strings.TrimPrefix(path, "/api/v1/workflows/"), "/")
		if len(segments) >= 1 {
			if id, ok := handlers.ParseID(segments[0]); ok {
				if len(segments) == 2 && segments[1] == "test" {
					if r.Method == http.MethodPost {
						rt.api.TestWorkflow(w, r, id)
						return
					}
				} else if len(segments) == 2 && segments[1] == "executions" {
					if r.Method == http.MethodGet {
						rt.api.GetWorkflowExecutions(w, r, id)
						return
					}
				} else {
					switch r.Method {
					case http.MethodPatch:
						rt.api.UpdateWorkflow(w, r, id)
						return
					case http.MethodDelete:
						rt.api.DeleteWorkflow(w, r, id)
						return
					}
				}
			}
		}
	case strings.HasPrefix(path, "/api/v1/integrations/"):
		segments := strings.Split(strings.TrimPrefix(path, "/api/v1/integrations/"), "/")
		if len(segments) == 1 {
			if r.Method == http.MethodPost {
				rt.api.ConnectIntegration(w, r, segments[0])
				return
			}
		}
		if len(segments) == 2 && segments[1] == "config" {
			if r.Method == http.MethodGet {
				if id, ok := handlers.ParseID(segments[0]); ok {
					rt.api.GetIntegrationConfig(w, r, id)
					return
				}
			}
		}
		if len(segments) == 1 {
			if r.Method == http.MethodDelete {
				if id, ok := handlers.ParseID(segments[0]); ok {
					rt.api.DisconnectIntegration(w, r, id)
					return
				}
			}
		}
	case path == "/api/v1/integrations":
		if r.Method == http.MethodGet {
			rt.api.ListIntegrations(w, r)
			return
		}
	case path == "/api/v1/webhooks/incoming":
		if r.Method == http.MethodPost {
			rt.api.ReceiveWebhook(w, r)
			return
		}
	case path == "/api/v1/audit-logs":
		if r.Method == http.MethodPost {
			rt.api.GetAuditLogs(w, r)
			return
		}
	case path == "/api/v1/notifications":
		if r.Method == http.MethodPost {
			rt.api.ListNotifications(w, r)
			return
		}
	case strings.HasPrefix(path, "/api/v1/notifications/"):
		if r.Method == http.MethodPatch {
			if id, ok := handlers.ParseID(strings.TrimPrefix(path, "/api/v1/notifications/")); ok {
				rt.api.MarkNotificationRead(w, r, id)
				return
			}
		}
	case path == "/api/v1/labels":
		if r.Method == http.MethodPost {
			rt.api.CreateLabel(w, r)
			return
		}
	case strings.HasPrefix(path, "/api/v1/messages/") && strings.HasSuffix(path, "/labels"):
		if r.Method == http.MethodPost {
			segments := strings.Split(strings.TrimPrefix(path, "/api/v1/messages/"), "/")
			if len(segments) == 2 && segments[1] == "labels" {
				if id, ok := handlers.ParseID(segments[0]); ok {
					rt.api.AddLabelToMessage(w, r, id)
					return
				}
			}
		}
	case strings.HasPrefix(path, "/api/v1/comments/"):
		if r.Method == http.MethodDelete {
			if id, ok := handlers.ParseID(strings.TrimPrefix(path, "/api/v1/comments/")); ok {
				rt.api.DeleteComment(w, r, id)
				return
			}
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
	case path == "/api/v1/auth/whatsapp/qr":
		if r.Method == http.MethodGet {
			rt.api.StartWhatsAppAuth(w, r)
			return
		}
	case path == "/api/v1/auth/whatsapp/status":
		if r.Method == http.MethodGet {
			rt.api.WhatsAppAuthStatus(w, r)
			return
		}
	case path == "/api/v1/auth/whatsapp/sync-contacts":
		if r.Method == http.MethodPost {
			rt.api.SyncContacts(w, r)
			return
		}
	case path == "/api/v1/auth/logout":
		if r.Method == http.MethodPost {
			rt.api.LogoutWhatsApp(w, r)
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
	case "/api/v1/auth/login", "/api/v1/auth/register", "/api/v1/auth/whatsapp/qr", "/api/v1/auth/whatsapp/status", "/api/v1/webhooks/incoming":
		return false
	default:
		return strings.HasPrefix(path, "/api/v1/")
	}
}
