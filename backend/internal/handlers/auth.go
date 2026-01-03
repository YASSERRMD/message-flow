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
	query := `
		INSERT INTO users (email, password_hash, tenant_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, email, password_hash, tenant_id, created_at, updated_at`

	if err := a.Store.WithTenantConn(ctx, req.TenantID, func(conn *pgxpool.Conn) error {
		return conn.QueryRow(ctx, query, req.Email, string(passwordHash), req.TenantID, time.Now().UTC(), time.Now().UTC()).Scan(
			&user.ID, &user.Email, &user.PasswordHash, &user.TenantID, &user.CreatedAt, &user.UpdatedAt,
		)
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to register user")
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

	a.logActivity(ctx, user.TenantID, auth.User{ID: user.ID, TenantID: user.TenantID, Email: user.Email}, "auth.register", map[string]any{
		"user_id": user.ID,
	})

	writeJSON(w, http.StatusCreated, map[string]any{
		"token": jwtToken,
		"csrf":  csrfToken,
		"user":  user,
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

	a.logActivity(ctx, user.TenantID, auth.User{ID: user.ID, TenantID: user.TenantID, Email: user.Email}, "auth.login", map[string]any{
		"user_id": user.ID,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"token": jwtToken,
		"csrf":  csrfToken,
		"user":  user,
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

	writeJSON(w, http.StatusOK, map[string]any{
		"user": record,
	})
}
