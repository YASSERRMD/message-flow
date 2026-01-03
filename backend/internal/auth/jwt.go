package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"message-flow/backend/internal/models"
)

type Service struct {
	secret []byte
	ttl    time.Duration
}

type Claims struct {
	UserID   int64  `json:"uid"`
	TenantID int64  `json:"tid"`
	Email    string `json:"email"`
	CSRF     string `json:"csrf"`
	jwt.RegisteredClaims
}

type User struct {
	ID       int64
	TenantID int64
	Email    string
	CSRF     string
}

func NewService(secret string, ttl time.Duration) (*Service, error) {
	if secret == "" {
		return nil, errors.New("JWT_SECRET is required")
	}
	if ttl == 0 {
		ttl = 24 * time.Hour
	}
	return &Service{secret: []byte(secret), ttl: ttl}, nil
}

func (s *Service) GenerateToken(user models.User, csrf string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   user.ID,
		TenantID: user.TenantID,
		Email:    user.Email,
		CSRF:     csrf,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

func (s *Service) ParseToken(token string) (User, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		return s.secret, nil
	})
	if err != nil {
		return User{}, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return User{}, errors.New("invalid token")
	}
	return User{ID: claims.UserID, TenantID: claims.TenantID, Email: claims.Email, CSRF: claims.CSRF}, nil
}

func GenerateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

type ctxKey struct{}

func WithUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, ctxKey{}, user)
}

func UserFromContext(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(ctxKey{}).(User)
	return user, ok
}
