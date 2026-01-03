package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	Pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	return &Store{Pool: pool}, nil
}

func (s *Store) Close() {
	if s.Pool != nil {
		s.Pool.Close()
	}
}

func (s *Store) WithTenantConn(ctx context.Context, tenantID int64, fn func(*pgxpool.Conn) error) error {
	conn, err := s.Pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	// SET command doesn't support parameters ($1), so we format the string
	if _, err := conn.Exec(ctx, fmt.Sprintf("SET app.tenant_id = '%d'", tenantID)); err != nil {
		return err
	}
	defer func() {
		_, _ = conn.Exec(ctx, "RESET app.tenant_id")
	}()
	return fn(conn)
}
