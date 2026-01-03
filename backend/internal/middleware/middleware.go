package middleware

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"message-flow/backend/internal/auth"
)

const (
	csrfHeader = "X-CSRF-Token"
)

func HandleCORS(w http.ResponseWriter, r *http.Request, allowedOrigin string) bool {
	if allowedOrigin != "" {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
	}
	w.Header().Set("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-CSRF-Token")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return true
	}
	return false
}

func SecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self' http://localhost:8080 ws://localhost:8080 http://localhost:5173; img-src 'self' data:; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com")
}

func Authenticate(r *http.Request, service *auth.Service) (auth.User, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		if token := r.URL.Query().Get("token"); token != "" {
			return service.ParseToken(token)
		}
		return auth.User{}, errors.New("missing authorization")
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return auth.User{}, errors.New("invalid authorization")
	}
	return service.ParseToken(strings.TrimSpace(parts[1]))
}

func ValidateCSRF(r *http.Request, user auth.User) error {
	switch r.Method {
	case http.MethodPost, http.MethodPatch, http.MethodDelete:
		value := r.Header.Get(csrfHeader)
		if value == "" || value != user.CSRF {
			return errors.New("invalid csrf token")
		}
	}
	return nil
}

type RateLimiter struct {
	limit  int
	window time.Duration
	mu     sync.Mutex
	items  map[string]*rateEntry
}

type rateEntry struct {
	count int
	reset time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, window: window, items: map[string]*rateEntry{}}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, ok := rl.items[key]
	if !ok || now.After(entry.reset) {
		rl.items[key] = &rateEntry{count: 1, reset: now.Add(rl.window)}
		return true
	}
	if entry.count >= rl.limit {
		return false
	}
	entry.count++
	return true
}

func ClientKey(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
