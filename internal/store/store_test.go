package store

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ISHANT57/shipyard/internal/testdb"
)

// newTestStore starts a real, throwaway, correctly migrated PostgreSQL
// database (via internal/testdb) and returns a Store connected to it.
// Using a real database instead of a mock is deliberate: it is exactly
// the unique constraint and ON CONFLICT behavior under test, and no mock
// reproduces Postgres's actual conflict semantics faithfully.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	dsn := testdb.NewPostgres(t)

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	t.Cleanup(pool.Close)

	return &Store{pool: pool}
}

func TestCreateAndGetProject(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	created, err := s.CreateProject(ctx, "shipyard", "https://github.com/ISHANT57/shipyard")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if created.ID == uuid.Nil {
		t.Error("CreateProject() returned zero ID")
	}

	fetched, err := s.GetProject(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetProject() error = %v", err)
	}
	if fetched != created {
		t.Errorf("GetProject() = %+v, want %+v", fetched, created)
	}
}

func TestGetProject_NotFound(t *testing.T) {
	s := newTestStore(t)

	_, err := s.GetProject(context.Background(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("GetProject() error = %v, want ErrNotFound", err)
	}
}

func TestCreatePipeline_IdempotentSubmission(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	project, err := s.CreateProject(ctx, "shipyard", "https://github.com/ISHANT57/shipyard")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	first, created1, err := s.CreatePipeline(ctx, project.ID, "same-key")
	if err != nil {
		t.Fatalf("first CreatePipeline() error = %v", err)
	}
	if !created1 {
		t.Error("first CreatePipeline(): created = false, want true")
	}

	second, created2, err := s.CreatePipeline(ctx, project.ID, "same-key")
	if err != nil {
		t.Fatalf("second CreatePipeline() error = %v", err)
	}
	if created2 {
		t.Error("second CreatePipeline(): created = true, want false (should return existing row)")
	}
	if second.ID != first.ID {
		t.Errorf("second CreatePipeline() returned a different pipeline: got ID %s, want %s", second.ID, first.ID)
	}
}

func TestCreatePipeline_DistinctKeysCreateDistinctPipelines(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	project, err := s.CreateProject(ctx, "shipyard", "https://github.com/ISHANT57/shipyard")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	a, _, err := s.CreatePipeline(ctx, project.ID, "key-a")
	if err != nil {
		t.Fatalf("CreatePipeline(key-a) error = %v", err)
	}
	b, _, err := s.CreatePipeline(ctx, project.ID, "key-b")
	if err != nil {
		t.Fatalf("CreatePipeline(key-b) error = %v", err)
	}

	if a.ID == b.ID {
		t.Error("distinct idempotency keys produced the same pipeline ID")
	}
}

// TestCreatePipeline_ConcurrentDuplicateSubmission is the concurrency
// case the Phase 03 acceptance criteria calls for directly: N concurrent
// submissions with the same idempotency key must all resolve to exactly
// one pipeline row, with no error from the losing side of the race.
func TestCreatePipeline_ConcurrentDuplicateSubmission(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	project, err := s.CreateProject(ctx, "shipyard", "https://github.com/ISHANT57/shipyard")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	const concurrency = 10
	type result struct {
		id      uuid.UUID
		created bool
		err     error
	}
	results := make(chan result, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			p, created, err := s.CreatePipeline(ctx, project.ID, "race-key")
			results <- result{id: p.ID, created: created, err: err}
		}()
	}

	seen := map[uuid.UUID]bool{}
	createdCount := 0
	for i := 0; i < concurrency; i++ {
		r := <-results
		if r.err != nil {
			t.Fatalf("concurrent CreatePipeline() error = %v", r.err)
		}
		seen[r.id] = true
		if r.created {
			createdCount++
		}
	}

	if len(seen) != 1 {
		t.Errorf("got %d distinct pipeline IDs across %d concurrent submissions, want 1", len(seen), concurrency)
	}
	if createdCount != 1 {
		t.Errorf("got %d submissions reporting created=true, want exactly 1", createdCount)
	}
}
