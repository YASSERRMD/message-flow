package auth

import (
	"testing"
	"time"

	"message-flow/backend/internal/models"
)

func TestTokenRoundTrip(t *testing.T) {
	service, err := NewService("test-secret", time.Hour)
	if err != nil {
		t.Fatalf("service init: %v", err)
	}

	csrf, err := GenerateCSRFToken()
	if err != nil {
		t.Fatalf("csrf: %v", err)
	}

	user := models.User{ID: 7, TenantID: 3, Email: "test@example.com"}
	token, err := service.GenerateToken(user, csrf)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	parsed, err := service.ParseToken(token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}

	if parsed.ID != user.ID || parsed.TenantID != user.TenantID || parsed.Email != user.Email {
		t.Fatalf("unexpected claims: %+v", parsed)
	}
	if parsed.CSRF != csrf {
		t.Fatalf("csrf mismatch")
	}
}
