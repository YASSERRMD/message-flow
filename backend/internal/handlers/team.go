package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"message-flow/backend/internal/models"
)

type addTeamUserRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type updateTeamUserRoleRequest struct {
	Role string `json:"role"`
}

type invitationRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (a *API) AddTeamUser(w http.ResponseWriter, r *http.Request) {
	var req addTeamUserRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Email == "" || req.Role == "" {
		writeError(w, http.StatusBadRequest, "email and role are required")
		return
	}

	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	var existing models.User
	err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		query := `SELECT id, email, tenant_id, created_at, updated_at FROM users WHERE email=$1 AND tenant_id=$2`
		return conn.QueryRow(ctx, query, req.Email, tenantID).Scan(&existing.ID, &existing.Email, &existing.TenantID, &existing.CreatedAt, &existing.UpdatedAt)
	})
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusInternalServerError, "failed to lookup user")
			return
		}
		invitation, err := a.createInvitation(ctx, tenantID, req.Email, req.Role, authUserIDPtr(r))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create invitation")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"status":      "invited",
			"invitation":  invitation,
			"description": "user not found, invitation created",
		})
		return
	}

	if err := a.setUserRole(ctx, tenantID, existing.ID, req.Role); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add user")
		return
	}

	a.logAudit(ctx, r, tenantID, authUserIDPtr(r), "team.user_added", stringPtr("user"), &existing.ID, nil, map[string]any{
		"user_id": existing.ID,
		"role":    req.Role,
	})

	writeJSON(w, http.StatusCreated, map[string]any{
		"status": "added",
		"user":   existing,
		"role":   req.Role,
	})
}

func (a *API) ListTeamUsers(w http.ResponseWriter, r *http.Request) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	users := []map[string]any{}
	invitations := []models.TeamInvitation{}

	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `
			SELECT u.id, u.email, u.created_at, u.updated_at, ue.role
			FROM users u
			LEFT JOIN users_extended ue ON ue.user_id = u.id
			WHERE u.tenant_id=$1
			ORDER BY u.created_at DESC`, tenantID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var id int64
			var email string
			var createdAt time.Time
			var updatedAt time.Time
			var role *string
			if err := rows.Scan(&id, &email, &createdAt, &updatedAt, &role); err != nil {
				rows.Close()
				return err
			}
			users = append(users, map[string]any{
				"id":         id,
				"email":      email,
				"role":       role,
				"created_at": createdAt,
				"updated_at": updatedAt,
			})
		}
		rows.Close()

		inviteRows, err := conn.Query(ctx, `
			SELECT id, tenant_id, email, role, status, token, invited_by, created_at, expires_at
			FROM team_invitations
			WHERE tenant_id=$1 AND status='pending'
			ORDER BY created_at DESC`, tenantID)
		if err != nil {
			return err
		}
		defer inviteRows.Close()
		for inviteRows.Next() {
			var inv models.TeamInvitation
			if err := inviteRows.Scan(&inv.ID, &inv.TenantID, &inv.Email, &inv.Role, &inv.Status, &inv.Token, &inv.InvitedBy, &inv.CreatedAt, &inv.ExpiresAt); err != nil {
				return err
			}
			invitations = append(invitations, inv)
		}
		return inviteRows.Err()
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load team users")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"users":       users,
		"invitations": invitations,
	})
}

func (a *API) UpdateTeamUserRole(w http.ResponseWriter, r *http.Request, userID int64) {
	var req updateTeamUserRoleRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Role == "" {
		writeError(w, http.StatusBadRequest, "role is required")
		return
	}

	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := a.setUserRole(ctx, tenantID, userID, req.Role); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update role")
		return
	}

	a.logAudit(ctx, r, tenantID, authUserIDPtr(r), "team.role_updated", stringPtr("user"), &userID, nil, map[string]any{
		"user_id": userID,
		"role":    req.Role,
	})

	writeJSON(w, http.StatusOK, map[string]any{"status": "updated"})
}

func (a *API) RemoveTeamUser(w http.ResponseWriter, r *http.Request, userID int64) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		_, err := conn.Exec(ctx, `DELETE FROM users_extended WHERE tenant_id=$1 AND user_id=$2`, tenantID, userID)
		if err != nil {
			return err
		}
		_, err = conn.Exec(ctx, `DELETE FROM team_members WHERE team_id=$1 AND user_id=$2`, tenantID, userID)
		return err
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove user")
		return
	}

	a.logAudit(ctx, r, tenantID, authUserIDPtr(r), "team.user_removed", stringPtr("user"), &userID, nil, nil)
	writeJSON(w, http.StatusOK, map[string]any{"status": "removed"})
}

func (a *API) SendInvitation(w http.ResponseWriter, r *http.Request) {
	var req invitationRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Email == "" || req.Role == "" {
		writeError(w, http.StatusBadRequest, "email and role are required")
		return
	}

	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	invitation, err := a.createInvitation(ctx, tenantID, req.Email, req.Role, authUserIDPtr(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create invitation")
		return
	}
	a.logAudit(ctx, r, tenantID, authUserIDPtr(r), "team.invitation_sent", stringPtr("invitation"), &invitation.ID, nil, map[string]any{
		"email": invitation.Email,
		"role":  invitation.Role,
	})
	writeJSON(w, http.StatusCreated, invitation)
}

func (a *API) GetTeamActivity(w http.ResponseWriter, r *http.Request) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	activity := []map[string]any{}
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `
			SELECT user_id, action, details_json, created_at
			FROM user_activity_logs
			WHERE tenant_id=$1
			ORDER BY created_at DESC
			LIMIT 200`, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var userID int64
			var action string
			var details *string
			var createdAt time.Time
			if err := rows.Scan(&userID, &action, &details, &createdAt); err != nil {
				return err
			}
			activity = append(activity, map[string]any{
				"user_id":    userID,
				"action":     action,
				"details":    details,
				"created_at": createdAt,
			})
		}
		return rows.Err()
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load activity")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": activity})
}

func (a *API) createInvitation(ctx context.Context, tenantID int64, email, role string, invitedBy *int64) (models.TeamInvitation, error) {
	var invitation models.TeamInvitation
	token, err := generateToken()
	if err != nil {
		return invitation, err
	}

	expires := time.Now().UTC().Add(7 * 24 * time.Hour)
	err = a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		query := `
			INSERT INTO team_invitations (tenant_id, email, role, status, token, invited_by, created_at, expires_at)
			VALUES ($1,$2,$3,'pending',$4,$5,$6,$7)
			ON CONFLICT (tenant_id, email, status)
			DO UPDATE SET token=EXCLUDED.token, role=EXCLUDED.role, created_at=EXCLUDED.created_at, expires_at=EXCLUDED.expires_at
			RETURNING id, tenant_id, email, role, status, token, invited_by, created_at, expires_at`
		return conn.QueryRow(ctx, query, tenantID, email, role, token, invitedBy, time.Now().UTC(), expires).Scan(
			&invitation.ID, &invitation.TenantID, &invitation.Email, &invitation.Role, &invitation.Status, &invitation.Token, &invitation.InvitedBy, &invitation.CreatedAt, &invitation.ExpiresAt,
		)
	})
	return invitation, err
}

func (a *API) setUserRole(ctx context.Context, tenantID, userID int64, role string) error {
	return a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		_, err := conn.Exec(ctx, `
			INSERT INTO users_extended (user_id, tenant_id, role, team_id, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6)
			ON CONFLICT (user_id)
			DO UPDATE SET role=EXCLUDED.role, updated_at=EXCLUDED.updated_at`, userID, tenantID, role, tenantID, time.Now().UTC(), time.Now().UTC())
		if err != nil {
			return err
		}
		_, err = conn.Exec(ctx, `
			INSERT INTO team_members (team_id, user_id, role, joined_at)
			VALUES ($1,$2,$3,$4)
			ON CONFLICT (team_id, user_id)
			DO UPDATE SET role=EXCLUDED.role`, tenantID, userID, role, time.Now().UTC())
		if err != nil {
			return err
		}
		_, err = conn.Exec(ctx, `
			INSERT INTO user_roles (user_id, tenant_id, role, created_at)
			VALUES ($1,$2,$3,$4)`, userID, tenantID, role, time.Now().UTC())
		return err
	})
}

func generateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
