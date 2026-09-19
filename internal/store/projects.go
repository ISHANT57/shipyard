package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Project mirrors the projects table.
type Project struct {
	ID        uuid.UUID
	Name      string
	RepoURL   string
	CreatedAt time.Time
}

// CreateProject inserts a new project and returns it with its
// database-generated ID and timestamp.
func (s *Store) CreateProject(ctx context.Context, name, repoURL string) (Project, error) {
	var p Project
	err := s.pool.QueryRow(ctx,
		`INSERT INTO projects (name, repo_url) VALUES ($1, $2)
		 RETURNING id, name, repo_url, created_at`,
		name, repoURL,
	).Scan(&p.ID, &p.Name, &p.RepoURL, &p.CreatedAt)
	if err != nil {
		return Project{}, fmt.Errorf("store: create project: %w", err)
	}
	return p, nil
}

// GetProject looks up a project by ID, returning ErrNotFound if none
// exists.
func (s *Store) GetProject(ctx context.Context, id uuid.UUID) (Project, error) {
	var p Project
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, repo_url, created_at FROM projects WHERE id = $1`,
		id,
	).Scan(&p.ID, &p.Name, &p.RepoURL, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Project{}, ErrNotFound
	}
	if err != nil {
		return Project{}, fmt.Errorf("store: get project: %w", err)
	}
	return p, nil
}
