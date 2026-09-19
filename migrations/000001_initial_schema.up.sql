-- Initial schema: projects, pipelines, stages, jobs, job_attempts, audit_log.
--
-- Status columns use TEXT + CHECK rather than native Postgres ENUM types.
-- A Postgres ENUM can't have a value removed or reordered without
-- rebuilding the type; a CHECK constraint is just a migration to alter.
-- Given this schema changes across multiple future phases (Phase 04
-- queue states, Phase 05 DAG states), that flexibility is worth more
-- than an ENUM's slightly stronger type guarantee.

CREATE EXTENSION IF NOT EXISTS pgcrypto; -- provides gen_random_uuid()

CREATE TABLE projects (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    repo_url    TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE pipelines (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id       UUID NOT NULL REFERENCES projects (id),
    -- Enforces exactly-once *pipeline creation* per key (see ADR-003 and
    -- docs/architecture/data-flow.md) — a retried or duplicated
    -- submission with the same key cannot create a second row.
    idempotency_key  TEXT NOT NULL UNIQUE,
    status           TEXT NOT NULL DEFAULT 'pending'
                         CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'cancelled')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_pipelines_project_id ON pipelines (project_id);

CREATE TABLE stages (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_id  UUID NOT NULL REFERENCES pipelines (id),
    name         TEXT NOT NULL,
    -- Dependency edges between stages are modeled in Phase 05 alongside
    -- the DAG scheduler itself; this table holds one row per stage only.
    status       TEXT NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending', 'ready', 'running', 'succeeded', 'failed', 'skipped')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_stages_pipeline_id ON stages (pipeline_id);

CREATE TABLE jobs (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stage_id       UUID NOT NULL REFERENCES stages (id),
    status         TEXT NOT NULL DEFAULT 'queued'
                       CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'retrying', 'dead')),
    attempt        INTEGER NOT NULL DEFAULT 0,
    max_attempts   INTEGER NOT NULL DEFAULT 5,
    -- Leasing columns for Phase 04's queue (SELECT ... FOR UPDATE SKIP
    -- LOCKED claims a job and sets locked_by/locked_until; a stale lease
    -- — locked_until in the past — is what lets a reaper reclaim a job
    -- whose worker crashed).
    locked_by      TEXT,
    locked_until   TIMESTAMPTZ,
    run_after      TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Carries the caller's trace context across the queue boundary so a
    -- trace survives from submission to execution (see
    -- docs/architecture/data-flow.md); populated starting Phase 09.
    trace_context  TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_jobs_stage_id ON jobs (stage_id);

-- Partial index: only rows a worker would ever claim are indexed, not
-- the (eventually much larger) set of already-finished jobs.
CREATE INDEX idx_jobs_claimable ON jobs (run_after) WHERE status = 'queued';

CREATE TABLE job_attempts (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id        UUID NOT NULL REFERENCES jobs (id),
    attempt_number INTEGER NOT NULL,
    status        TEXT NOT NULL
                      CHECK (status IN ('running', 'succeeded', 'failed')),
    error         TEXT,
    started_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at   TIMESTAMPTZ
);

CREATE INDEX idx_job_attempts_job_id ON job_attempts (job_id);

CREATE TABLE audit_log (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor          TEXT NOT NULL,
    action         TEXT NOT NULL,
    resource_type  TEXT NOT NULL,
    resource_id    UUID,
    metadata       JSONB,
    -- Append-only by convention: no UPDATE/DELETE path is exposed by
    -- internal/store. Postgres itself doesn't enforce this (see
    -- docs/architecture/threat-model.md — a direct DB compromise still
    -- bypasses it), but no application code path ever mutates a row here.
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_log_resource ON audit_log (resource_type, resource_id);
