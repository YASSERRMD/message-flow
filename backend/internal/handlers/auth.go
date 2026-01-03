package handlers

import (
	"context"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

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

	if err := a.Store.Pool.QueryRow(ctx, query, req.Email, string(passwordHash), req.TenantID, time.Now().UTC(), time.Now().UTC()).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.TenantID, &user.CreatedAt, &user.UpdatedAt,
	); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to register user")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"user": user,
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

	if err := a.Store.Pool.QueryRow(ctx, query, req.Email, req.TenantID).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.TenantID, &user.CreatedAt, &user.UpdatedAt,
	); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token": "mock-token",
		"user":  user,
	})
}
