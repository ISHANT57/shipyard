// Package store is the repository layer over PostgreSQL. It is the only
// package in Shipyard that knows SQL exists — everything else (the API
// handlers, eventually the worker) calls typed methods here and gets Go
// structs back, never a raw *pgxpool.Pool or a SQL string.
package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned by any lookup method when no matching row
// exists. Callers check for this sentinel with errors.Is instead of
// depending on pgx's own not-found error type, which keeps pgx an
// implementation detail of this package alone.
var ErrNotFound = errors.New("store: not found")

// Store wraps a PostgreSQL connection pool.
type Store struct {
	pool *pgxpool.Pool
}

// New connects to PostgreSQL and verifies the connection with a ping
// before returning — the same fail-fast principle as internal/config:
// a bad connection string is reported at startup, not at the first
// request that happens to touch the database.
func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("store: creating pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("store: ping: %w", err)
	}

	return &Store{pool: pool}, nil
}

// Close releases all pooled connections. Safe to call once, at shutdown.
func (s *Store) Close() {
	s.pool.Close()
}

// Ping is used by the API's /readyz handler: unlike /healthz, readiness
// is allowed to depend on the database being reachable.
func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}
