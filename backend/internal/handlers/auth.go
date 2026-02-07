package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"message-flow/backend/internal/auth"
	"message-flow/backend/internal/models"
)

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	TenantID int64  `json:"tenant_id"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	TenantID int64  `json:"tenant_id"`
}

func (a *API) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Email == "" || req.Password == "" || req.TenantID == 0 {
		writeError(w, http.StatusBadRequest, "email, password, and tenant_id are required")
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var user models.User
	role := "member"
	query := `
		INSERT INTO users (email, password_hash, tenant_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, email, password_hash, tenant_id, created_at, updated_at`

	if err := a.Store.WithTenantConn(ctx, req.TenantID, func(conn *pgxpool.Conn) error {
		var count int
		if err := conn.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE tenant_id=$1`, req.TenantID).Scan(&count); err != nil {
			return err
		}
		if count == 0 {
			role = "owner"
		}
		return conn.QueryRow(ctx, query, req.Email, string(passwordHash), req.TenantID, time.Now().UTC(), time.Now().UTC()).Scan(
			&user.ID, &user.Email, &user.PasswordHash, &user.TenantID, &user.CreatedAt, &user.UpdatedAt,
		)
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to register user")
		return
	}

	// Ensure this user can actually access member/owner endpoints (RBAC reads users_extended).
	if err := a.setUserRole(ctx, req.TenantID, user.ID, role); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to assign role")
		return
	}

	csrfToken, err := auth.GenerateCSRFToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate csrf token")
		return
	}
	jwtToken, err := a.Auth.GenerateToken(user, csrfToken)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	// Log registration
	a.logActivity(ctx, user.TenantID, auth.User{ID: user.ID, TenantID: user.TenantID, Email: user.Email}, "auth.register", map[string]any{
		"user_id": user.ID,
	})

	writeJSON(w, http.StatusCreated, map[string]any{
		"result": "registered",
		"token":  jwtToken,
		"csrf":   csrfToken,
		"user":   user,
		"role":   role,
	})
}

func (a *API) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Email == "" || req.Password == "" || req.TenantID == 0 {
		writeError(w, http.StatusBadRequest, "email, password, and tenant_id are required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var user models.User
	query := `
		SELECT id, email, password_hash, tenant_id, created_at, updated_at
		FROM users
		WHERE email=$1 AND tenant_id=$2`

	if err := a.Store.WithTenantConn(ctx, req.TenantID, func(conn *pgxpool.Conn) error {
		return conn.QueryRow(ctx, query, req.Email, req.TenantID).Scan(
			&user.ID, &user.Email, &user.PasswordHash, &user.TenantID, &user.CreatedAt, &user.UpdatedAt,
		)
	}); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	role, roleErr := a.getUserRole(ctx, user.TenantID, user.ID)
	if roleErr != nil || role == "" || role == "viewer" {
		// Bootstrap role records for users created before team/roles migrations (or after bugs).
		// Default: member; first user in tenant: owner.
		fallbackRole := "member"
		_ = a.Store.WithTenantConn(ctx, user.TenantID, func(conn *pgxpool.Conn) error {
			var count int
			if err := conn.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE tenant_id=$1`, user.TenantID).Scan(&count); err != nil {
				return err
			}
			if count == 1 {
				fallbackRole = "owner"
			}
			return nil
		})
		if err := a.setUserRole(ctx, user.TenantID, user.ID, fallbackRole); err == nil {
			role = fallbackRole
		} else {
			role = "viewer"
		}
	}

	csrfToken, err := auth.GenerateCSRFToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate csrf token")
		return
	}
	jwtToken, err := a.Auth.GenerateToken(user, csrfToken)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	// Log login
	a.logActivity(ctx, user.TenantID, auth.User{ID: user.ID, TenantID: user.TenantID, Email: user.Email}, "auth.login", map[string]any{
		"user_id": user.ID,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"token": jwtToken,
		"csrf":  csrfToken,
		"user":  user,
		"role":  role,
	})
}

func (a *API) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var record models.User
	query := `
		SELECT id, email, password_hash, tenant_id, created_at, updated_at
		FROM users
		WHERE id=$1 AND tenant_id=$2`

	if err := a.Store.WithTenantConn(ctx, user.TenantID, func(conn *pgxpool.Conn) error {
		return conn.QueryRow(ctx, query, user.ID, user.TenantID).Scan(
			&record.ID, &record.Email, &record.PasswordHash, &record.TenantID, &record.CreatedAt, &record.UpdatedAt,
		)
	}); err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	role, _ := a.getUserRole(ctx, user.TenantID, user.ID)
	writeJSON(w, http.StatusOK, map[string]any{
		"user": record,
		"role": role,
	})
}

func (a *API) SyncContacts(w http.ResponseWriter, r *http.Request) {
	tenantID := a.tenantID(r)
	if a.WhatsApp != nil {
		// Trigger background sync
		go a.WhatsApp.SyncContactsForTenant(tenantID)
		writeJSON(w, http.StatusOK, map[string]string{"status": "sync_started"})
		return
	}
	writeError(w, http.StatusServiceUnavailable, "whatsapp not initialized")
}
