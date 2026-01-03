package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"message-flow/backend/internal/auth"
)

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)
	key := "client"
	if !limiter.Allow(key) {
		t.Fatalf("expected allow on first")
	}
	if !limiter.Allow(key) {
		t.Fatalf("expected allow on second")
	}
	if limiter.Allow(key) {
		t.Fatalf("expected block on third")
	}
}

func TestValidateCSRF(t *testing.T) {
	user := auth.User{CSRF: "token"}
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	if err := ValidateCSRF(req, user); err == nil {
		t.Fatalf("expected csrf error")
	}
	req.Header.Set("X-CSRF-Token", "token")
	if err := ValidateCSRF(req, user); err != nil {
		t.Fatalf("unexpected csrf error: %v", err)
	}
}
