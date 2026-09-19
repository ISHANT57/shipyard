// Package testdb provides a real, throwaway PostgreSQL database, with
// the project's actual migrations applied, for integration tests.
//
// It is a regular (non-_test.go) file so that more than one package's
// tests can import it — internal/store and cmd/shipyard-api both need
// a correctly migrated test database, and this is the one place that
// setup is written, instead of being copy-pasted into each package's
// test file.
package testdb

import (
	"context"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // registers the "postgres" driver scheme via init()
	_ "github.com/golang-migrate/migrate/v4/source/file"       // registers the "file" source scheme via init()
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// migrationsPath is relative to the repository root and works from any
// package exactly two directories deep (internal/store, cmd/shipyard-api
// — the only callers today). If a caller ever lives at a different
// depth, this will need to become a parameter instead of a constant.
const migrationsPath = "file://../../migrations"

// NewPostgres starts a throwaway PostgreSQL container, applies every
// migration in migrations/ against it, and returns a connection string.
// The container is terminated automatically via t.Cleanup.
func NewPostgres(t *testing.T) string {
	t.Helper()
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:16",
		tcpostgres.WithDatabase("shipyard_test"),
		tcpostgres.WithUsername("shipyard"),
		tcpostgres.WithPassword("shipyard"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("testdb: failed to start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("testdb: failed to terminate postgres container: %v", err)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("testdb: failed to get connection string: %v", err)
	}

	applyMigrations(t, dsn)
	return dsn
}

// applyMigrations runs the real migrations/ directory against dsn using
// the same golang-migrate library ADR-007 commits to for production
// use — tests exercise the actual migration files, not a hand-copied
// schema that could silently drift from them.
func applyMigrations(t *testing.T, dsn string) {
	t.Helper()

	// wait.ForListeningPort (inside BasicWaitStrategies) only confirms
	// the port accepts TCP connections, not that Postgres has finished
	// initializing roles/databases; a short retry loop absorbs that gap
	// instead of failing on an occasional slow start.
	var m *migrate.Migrate
	var err error
	for attempt := 0; attempt < 10; attempt++ {
		m, err = migrate.New(migrationsPath, dsn)
		if err == nil {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("testdb: failed to initialize migrate: %v", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil {
		t.Fatalf("testdb: failed to apply migrations: %v", err)
	}
}
