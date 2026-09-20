# Changelog

All notable changes to this project are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and versioning follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- Durable Postgres job queue (`internal/queue`): at-least-once delivery with
  fenced completion, `FOR UPDATE SKIP LOCKED` claims, leases and
  heartbeats, full-jitter backoff, dead-letter status, stale-lease reaper,
  and `LISTEN`/`NOTIFY` wake-ups (ADR-008, ADR-009, ADR-010).
- Worker (`internal/worker`, `cmd/shipyard-worker`): bounded pool,
  per-job heartbeats, panic containment, graceful drain on shutdown.
- Worker configuration (`SHIPYARD_WORKER_*`) with fail-fast validation.
- Migration `000002`: partial index supporting the lease reaper.
- Makefile targets: `compose-up`, `compose-down`, `migrate-up`, `migrate-down`.
- Git bisect/revert drill script (`scripts/drills/bisect-drill.sh`).

## [0.1.0] - 2026-09-19

### Added

- Engineering foundation: product docs, phase tracker, ADR process,
  contributing and security policy, GitHub issue and PR templates,
  documentation CI (markdownlint, link check), branch ruleset.
- Architecture documents with Mermaid diagrams and a STRIDE threat model;
  ADR-001 to ADR-007.
- Control-plane API skeleton (`cmd/shipyard-api`): `/healthz`, `/readyz`,
  graceful shutdown, request-ID and logging middleware, non-root
  distroless image, code CI (gofmt, vet, golangci-lint, race-enabled tests).
- PostgreSQL schema and reversible migrations (`projects`, `pipelines`,
  `stages`, `jobs`, `job_attempts`, `audit_log`), `internal/store`, and
  idempotent pipeline submission (`POST /projects`, `POST /pipelines`).
- Integration test harness using Testcontainers-Go (`internal/testdb`).
