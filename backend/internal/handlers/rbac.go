package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	roleOwner   = "owner"
	roleAdmin   = "admin"
	roleManager = "manager"
	roleMember  = "member"
	roleViewer  = "viewer"
)

var roleRank = map[string]int{
	roleOwner:   5,
	roleAdmin:   4,
	roleManager: 3,
	roleMember:  2,
	roleViewer:  1,
}

func (a *API) Authorize(r *http.Request, requiredRole string) bool {
	if requiredRole == "" {
		return true
	}
	tenantID := a.tenantID(r)
	userID := authUserIDPtr(r)
	role := roleViewer
	if userID != nil {
		if value, err := a.getUserRole(r.Context(), tenantID, *userID); err == nil && value != "" {
			role = value
		}
	}

	allowed := roleRank[strings.ToLower(role)] >= roleRank[strings.ToLower(requiredRole)]
	a.logAudit(r.Context(), r, tenantID, userID, "permission.check", stringPtr("api"), nil, nil, map[string]any{
		"required_role": requiredRole,
		"user_role":     role,
		"method":        r.Method,
		"path":          strings.TrimSuffix(r.URL.Path, "/"),
		"allowed":       allowed,
	})
	return allowed
}

func (a *API) getUserRole(ctx context.Context, tenantID int64, userID int64) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var role string
	err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		query := `
			SELECT role
			FROM users_extended
			WHERE tenant_id=$1 AND user_id=$2`
		return conn.QueryRow(ctx, query, tenantID, userID).Scan(&role)
	})
	if err == nil {
		return role, nil
	}

	return roleViewer, err
}

func RequiredRole(path, method string) string {
	switch {
	case path == "/api/v1/auth/login", path == "/api/v1/auth/register":
		return ""
	case path == "/api/v1/auth/whatsapp/qr", path == "/api/v1/auth/whatsapp/status":
		return ""
	case path == "/api/v1/ws":
		return roleViewer
	case path == "/api/v1/dashboard":
		return roleViewer
	case path == "/api/v1/dashboard/stream":
		return roleViewer
	case path == "/api/v1/conversations":
		return roleViewer
	case strings.HasPrefix(path, "/api/v1/conversations/") && strings.HasSuffix(path, "/messages"):
		return roleViewer
	case path == "/api/v1/messages/reply":
		return roleMember
	case path == "/api/v1/messages/forward":
		return roleMember
	case path == "/api/v1/important-messages":
		return roleViewer
	case strings.HasPrefix(path, "/api/v1/messages/") && strings.HasSuffix(path, "/labels"):
		return roleMember
	case path == "/api/v1/action-items":
		if method == http.MethodGet {
			return roleViewer
		}
		return roleManager
	case strings.HasPrefix(path, "/api/v1/action-items/") && strings.HasSuffix(path, "/comments"):
		if method == http.MethodGet {
			return roleViewer
		}
		return roleMember
	case strings.HasPrefix(path, "/api/v1/action-items/"):
		return roleManager
	case path == "/api/v1/daily-summary":
		return roleViewer
	case path == "/api/v1/conversations/summarize":
		return roleManager
	case path == "/api/v1/llm/providers":
		if method == http.MethodGet {
			return roleAdmin
		}
		return roleAdmin
	case path == "/api/v1/llm/providers/comparison":
		return roleManager
	case strings.HasPrefix(path, "/api/v1/llm/providers/"):
		if strings.HasSuffix(path, "/history") {
			return roleAdmin
		}
		if strings.HasSuffix(path, "/test") {
			return roleAdmin
		}
		return roleAdmin
	case path == "/api/v1/llm/usage":
		return roleManager
	case path == "/api/v1/llm/costs":
		return roleManager
	case path == "/api/v1/llm/analytics/cost-breakdown":
		return roleManager
	case path == "/api/v1/llm/analytics/usage-by-feature":
		return roleManager
	case path == "/api/v1/llm/health":
		return roleManager
	case path == "/api/v1/llm/features":
		return roleAdmin
	case strings.HasPrefix(path, "/api/v1/llm/features/"):
		return roleAdmin
	case path == "/api/v1/llm/bulk-test":
		return roleAdmin
	case path == "/api/v1/llm/recommendations":
		return roleManager
	case path == "/api/v1/team/users":
		if method == http.MethodGet {
			return roleAdmin
		}
		return roleAdmin
	case strings.HasPrefix(path, "/api/v1/team/users/"):
		return roleAdmin
	case path == "/api/v1/team/invitations":
		return roleAdmin
	case path == "/api/v1/team/activity":
		return roleManager
	case path == "/api/v1/workflows":
		if method == http.MethodGet {
			return roleManager
		}
		return roleAdmin
	case strings.HasPrefix(path, "/api/v1/workflows/"):
		if strings.HasSuffix(path, "/executions") {
			return roleManager
		}
		if strings.HasSuffix(path, "/test") {
			return roleAdmin
		}
		return roleAdmin
	case strings.HasPrefix(path, "/api/v1/integrations"):
		if method == http.MethodGet {
			return roleAdmin
		}
		return roleAdmin
	case path == "/api/v1/webhooks/incoming":
		return ""
	case path == "/api/v1/audit-logs":
		return roleAdmin
	case path == "/api/v1/notifications":
		return roleMember
	case strings.HasPrefix(path, "/api/v1/notifications/"):
		return roleMember
	case path == "/api/v1/labels":
		return roleManager
	case strings.HasPrefix(path, "/api/v1/comments/"):
		return roleMember
	default:
		if strings.HasPrefix(path, "/api/v1/") {
			return roleViewer
		}
	}
	return ""
}

func stringPtr(value string) *string {
	return &value
}
