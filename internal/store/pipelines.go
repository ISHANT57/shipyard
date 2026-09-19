package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Pipeline mirrors the pipelines table.
type Pipeline struct {
	ID             uuid.UUID
	ProjectID      uuid.UUID
	IdempotencyKey string
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CreatePipeline submits a pipeline for a project, keyed by an
// idempotency key supplied by the caller.
//
// If a pipeline with this idempotency key already exists, that existing
// pipeline is returned instead of an error, and created reports false —
// this is what makes pipeline submission safe to retry (see
// docs/architecture/data-flow.md): a client that resubmits after a
// timeout, unsure whether the first attempt landed, gets back the same
// pipeline either way instead of creating a second one or seeing a
// confusing conflict error.
func (s *Store) CreatePipeline(ctx context.Context, projectID uuid.UUID, idempotencyKey string) (pipeline Pipeline, created bool, err error) {
	// INSERT ... ON CONFLICT DO NOTHING RETURNING: if the key is new, the
	// row is inserted and returned in one round trip. If it already
	// exists, ON CONFLICT DO NOTHING means no row is returned and no
	// error is raised either — pgx.ErrNoRows on the Scan tells us "the
	// key was already taken", not "something went wrong".
	err = s.pool.QueryRow(ctx,
		`INSERT INTO pipelines (project_id, idempotency_key)
		 VALUES ($1, $2)
		 ON CONFLICT (idempotency_key) DO NOTHING
		 RETURNING id, project_id, idempotency_key, status, created_at, updated_at`,
		projectID, idempotencyKey,
	).Scan(&pipeline.ID, &pipeline.ProjectID, &pipeline.IdempotencyKey, &pipeline.Status, &pipeline.CreatedAt, &pipeline.UpdatedAt)

	if err == nil {
		return pipeline, true, nil
	}

	// The insert hit the conflict; fetch and return the existing row.
	existing, getErr := s.getPipelineByIdempotencyKey(ctx, idempotencyKey)
	if getErr != nil {
		return Pipeline{}, false, fmt.Errorf("store: create pipeline: insert failed (%v) and lookup after conflict failed: %w", err, getErr)
	}
	return existing, false, nil
}

func (s *Store) getPipelineByIdempotencyKey(ctx context.Context, key string) (Pipeline, error) {
	var p Pipeline
	err := s.pool.QueryRow(ctx,
		`SELECT id, project_id, idempotency_key, status, created_at, updated_at
		 FROM pipelines WHERE idempotency_key = $1`,
		key,
	).Scan(&p.ID, &p.ProjectID, &p.IdempotencyKey, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return Pipeline{}, fmt.Errorf("store: get pipeline by idempotency key: %w", err)
	}
	return p, nil
}
