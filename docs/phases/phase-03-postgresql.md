# Phase 03 — PostgreSQL

## Purpose

Design the schema and learn transactional correctness before it is
overloaded with queue semantics in Phase 04.

## Concepts to learn

Normalization; constraints as correctness, not decoration; indexes
(B-tree, partial); MVCC; isolation levels; row locks; connection pooling
with `pgx`; reversible migrations; enums vs check constraints.

## Build

`deployments/compose/docker-compose.yml` (Postgres); `migrations/`
(tool chosen via ADR-007); tables: `projects`, `pipelines`, `stages`,
`jobs`, `job_attempts`, `audit_log`; `internal/store` repository layer;
project CRUD + pipeline submission endpoint (`201` + id) with an
idempotency key (unique constraint) to prevent duplicate submissions.

## ADR

ADR-007: migration tool (`goose` vs `golang-migrate`).

## Tests

Integration tests via Testcontainers-Go; constraint-violation cases;
concurrent duplicate submission resolves to exactly one row.

## Release

First tagged release: `v0.1.0` via `gh release create v0.1.0
--generate-notes`. Introduces annotated tags (`git tag -a`),
`git describe`, and SemVer for a pre-1.0 project.

## Acceptance criteria

Migrations are reversible (up and down both tested). Duplicate-submission
test passes. `v0.1.0` is published on GitHub.

## Tasks

- [x] ADR-007: migration tool (golang-migrate)
- [x] `docker-compose.yml` with PostgreSQL (moved off port 5432 to avoid
      colliding with an unrelated local Postgres instance)
- [x] Migration 000001: `projects`, `pipelines`, `stages`, `jobs`,
      `job_attempts`, `audit_log` — verified up/down/up against a real
      database, not just written
- [x] `internal/store`: repository layer over `pgx`, `ErrNotFound`
      sentinel, idempotent `CreatePipeline`
- [x] `internal/testdb`: shared Testcontainers-Go helper (real Postgres,
      real migrations) reused by both `internal/store` and
      `cmd/shipyard-api` tests
- [x] API: `POST /projects`, `POST /pipelines` (201 on create, 200 on
      idempotent replay); `/readyz` now actually checks the database
- [x] Integration tests: constraint violations, idempotent replay,
      10-way concurrent duplicate submission (race-detector clean)
- [x] Live smoke test against the real compose database (not just tests)
- [ ] Push, PR, CI, merge
- [ ] Release v0.1.0
- [ ] Phase closeout report

## Phase closeout report

Not yet written — fill in only once Phase 03 is fully done (after merge
and release).
