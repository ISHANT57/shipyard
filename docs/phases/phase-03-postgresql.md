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
