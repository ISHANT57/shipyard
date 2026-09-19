package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/ISHANT57/shipyard/internal/store"
)

// createProjectRequest / pipelineResponse etc. are small, explicit
// wire types, kept separate from store.Project/store.Pipeline. The API
// contract and the database row shape are allowed to diverge (e.g. if a
// column is renamed internally, or a field shouldn't be exposed) —
// collapsing them into one type would accidentally couple the two.

type createProjectRequest struct {
	Name    string `json:"name"`
	RepoURL string `json:"repo_url"`
}

type projectResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	RepoURL   string `json:"repo_url"`
	CreatedAt string `json:"created_at"`
}

func toProjectResponse(p store.Project) projectResponse {
	return projectResponse{
		ID:        p.ID.String(),
		Name:      p.Name,
		RepoURL:   p.RepoURL,
		CreatedAt: p.CreatedAt.Format(timeFormat),
	}
}

type createPipelineRequest struct {
	ProjectID      string `json:"project_id"`
	IdempotencyKey string `json:"idempotency_key"`
}

type pipelineResponse struct {
	ID             string `json:"id"`
	ProjectID      string `json:"project_id"`
	IdempotencyKey string `json:"idempotency_key"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at"`
}

func toPipelineResponse(p store.Pipeline) pipelineResponse {
	return pipelineResponse{
		ID:             p.ID.String(),
		ProjectID:      p.ProjectID.String(),
		IdempotencyKey: p.IdempotencyKey,
		Status:         p.Status,
		CreatedAt:      p.CreatedAt.Format(timeFormat),
	}
}

const timeFormat = "2006-01-02T15:04:05Z07:00" // RFC 3339, the standard wire format for timestamps

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

// handleCreateProject handles POST /projects.
func handleCreateProject(s *store.Store, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createProjectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Name == "" || req.RepoURL == "" {
			writeError(w, http.StatusBadRequest, "name and repo_url are required")
			return
		}

		project, err := s.CreateProject(r.Context(), req.Name, req.RepoURL)
		if err != nil {
			logger.Error("create project failed", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to create project")
			return
		}

		writeJSON(w, http.StatusCreated, toProjectResponse(project))
	}
}

// handleCreatePipeline handles POST /pipelines. It returns 201 when a new
// pipeline was created, and 200 when the idempotency key matched an
// existing one — the client sees the same pipeline either way, but the
// status code tells it honestly whether this call was the one that
// created it.
func handleCreatePipeline(s *store.Store, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createPipelineRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.IdempotencyKey == "" {
			writeError(w, http.StatusBadRequest, "idempotency_key is required")
			return
		}
		projectID, err := uuid.Parse(req.ProjectID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "project_id must be a valid UUID")
			return
		}

		if _, err := s.GetProject(r.Context(), projectID); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "project not found")
				return
			}
			logger.Error("lookup project failed", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to look up project")
			return
		}

		pipeline, created, err := s.CreatePipeline(r.Context(), projectID, req.IdempotencyKey)
		if err != nil {
			logger.Error("create pipeline failed", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to create pipeline")
			return
		}

		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}
		writeJSON(w, status, toPipelineResponse(pipeline))
	}
}
