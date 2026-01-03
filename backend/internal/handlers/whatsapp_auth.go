package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"golang.org/x/crypto/bcrypt"

	"message-flow/backend/internal/auth"
	"message-flow/backend/internal/models"
)

type whatsappQRResponse struct {
	SessionID      string `json:"session_id"`
	QRCode         string `json:"qr_code"`
	TimeoutSeconds int    `json:"timeout_seconds"`
	Status         string `json:"status"`
	Error          string `json:"error,omitempty"`
	TenantID       int64  `json:"tenant_id"`
}

func (a *API) StartWhatsAppAuth(w http.ResponseWriter, r *http.Request) {
	if a.WhatsApp == nil {
		writeError(w, http.StatusServiceUnavailable, "whatsapp integration not configured")
		return
	}

	tenantID := a.tenantID(r)

	// Use background context for WhatsApp session - it must persist beyond this HTTP request
	session, err := a.WhatsApp.StartSession(context.Background(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start whatsapp session")
		return
	}

	// Wait up to 8 seconds for QR code to be generated
	deadline := time.Now().Add(8 * time.Second)
	for session.LastQR == "" && time.Now().Before(deadline) {
		time.Sleep(250 * time.Millisecond)
		updated, ok := a.WhatsApp.GetSession(session.ID)
		if !ok {
			break
		}
		session = updated
	}

	response := whatsappQRResponse{
		SessionID: session.ID,
		Status:    session.Status,
		TenantID:  tenantID,
	}

	if session.LastQR != "" {
		png, err := qrcode.Encode(session.LastQR, qrcode.Medium, 280)
		if err == nil {
			response.QRCode = "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
			response.TimeoutSeconds = int(session.LastExpiry.Seconds())
		}
	}

	writeJSON(w, http.StatusOK, response)
}

func (a *API) WhatsAppAuthStatus(w http.ResponseWriter, r *http.Request) {
	if a.WhatsApp == nil {
		writeError(w, http.StatusServiceUnavailable, "whatsapp integration not configured")
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "session_id is required")
		return
	}

	session, ok := a.WhatsApp.GetSession(sessionID)
	if !ok {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	response := whatsappQRResponse{
		SessionID: session.ID,
		Status:    session.Status,
		TenantID:  session.TenantID,
		Error:     session.Error,
	}

	if session.LastQR != "" {
		png, err := qrcode.Encode(session.LastQR, qrcode.Medium, 280)
		if err == nil {
			response.QRCode = "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
			response.TimeoutSeconds = int(session.LastExpiry.Seconds())
		}
	}

	if session.Status == "connected" {
		user, role, token, csrf, err := a.ensureWhatsAppUser(r.Context(), session.TenantID, session.Client)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to finalize whatsapp login")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":    session.Status,
			"session":   response,
			"user":      user,
			"role":      role,
			"token":     token,
			"csrf":      csrf,
			"tenant_id": session.TenantID,
		})
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (a *API) ensureWhatsAppUser(ctx context.Context, tenantID int64, client *whatsmeow.Client) (models.User, string, string, string, error) {
	var user models.User
	if client == nil || client.Store == nil || client.Store.ID == nil {
		return user, "", "", "", errNotFound
	}

	jid := client.Store.ID.String()
	email := "whatsapp+" + jid + "@messageflow.local"

	role := "member"
	err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		var count int
		if err := conn.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE tenant_id=$1`, tenantID).Scan(&count); err != nil {
			return err
		}
		if count == 0 {
			role = "owner"
		}
		query := `
			SELECT id, email, password_hash, tenant_id, created_at, updated_at
			FROM users
			WHERE email=$1 AND tenant_id=$2`
		if err := conn.QueryRow(ctx, query, email, tenantID).Scan(
			&user.ID, &user.Email, &user.PasswordHash, &user.TenantID, &user.CreatedAt, &user.UpdatedAt,
		); err == nil {
			return nil
		}

		pass := make([]byte, 24)
		_, _ = rand.Read(pass)
		hash, err := bcrypt.GenerateFromPassword(pass, bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		insert := `
			INSERT INTO users (email, password_hash, tenant_id, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5)
			RETURNING id, email, password_hash, tenant_id, created_at, updated_at`
		return conn.QueryRow(ctx, insert, email, string(hash), tenantID, time.Now().UTC(), time.Now().UTC()).Scan(
			&user.ID, &user.Email, &user.PasswordHash, &user.TenantID, &user.CreatedAt, &user.UpdatedAt,
		)
	})
	if err != nil {
		return user, "", "", "", err
	}

	if err := a.setUserRole(ctx, tenantID, user.ID, role); err != nil {
		return user, "", "", "", err
	}

	csrfToken, err := auth.GenerateCSRFToken()
	if err != nil {
		return user, "", "", "", err
	}
	jwtToken, err := a.Auth.GenerateToken(user, csrfToken)
	if err != nil {
		return user, "", "", "", err
	}

	a.logActivity(ctx, tenantID, auth.User{ID: user.ID, TenantID: tenantID, Email: user.Email}, "auth.whatsapp", map[string]any{
		"user_id": user.ID,
		"jid":     jid,
	})

	return user, role, jwtToken, csrfToken, nil
}
