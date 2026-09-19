# ADR-007: Migration tool

Status: Accepted
Date: 2026-09-19

## Context

Shipyard's schema will change repeatedly across Phase 03 (initial tables)
and every later phase that adds state. Migrations need to be versioned,
reversible, and runnable both locally and from CI/an init container,
without hand-tracking which SQL files have already run against a given
database.

## Problem

Which tool manages PostgreSQL schema migrations: `golang-migrate`,
`goose`, or a hand-rolled runner?

## Decision Drivers

Plain SQL migrations (no tool-specific DSL to learn on top of SQL
itself); a Go library form usable directly from `cmd/shipyard-api` for an
optional "migrate on startup" path, not just a CLI; minimal operational
surface; active maintenance.

## Options Considered

### Option A — `golang-migrate/migrate`

Very widely used, supports many database backends beyond Postgres (not
needed here, but signals broad testing), plain up/down `.sql` files
named `NNNN_description.up.sql` / `.down.sql`. Usable as a CLI or as a Go
library.

### Option B — `pressly/goose`

Postgres-focused in practice, also plain SQL (or Go functions, unused
here), single file per migration with `-- +goose Up` / `-- +goose Down`
annotations rather than two separate files. Also usable as CLI or library.

### Option C — Hand-rolled migration runner

A `migrations` table plus a small Go program that applies un-run `.sql`
files in order. Full control, but reimplements version tracking,
concurrent-run locking, and dirty-state detection that both existing
tools already handle correctly.

## Decision

`golang-migrate/migrate`.

## Why this decision?

Two separate up/down files per migration (Option A) keeps each file
readable as plain SQL with no embedded tool-specific comment syntax,
which matters for a project explicitly about learning the underlying
mechanics, not a tool's conventions. Its Go library form
(`github.com/golang-migrate/migrate/v4`) lets `cmd/shipyard-api` run
migrations programmatically if that's ever wanted (e.g. an init check on
startup), without requiring the CLI. A hand-rolled runner (Option C) was
rejected the same way ADR-003 rejected a queue library that would remove
a learning goal — here, correctly, it's the reverse: migration *tracking*
is bookkeeping, not a Phase 03 learning objective the way queue mechanics
are a named Phase 04 objective, so reusing a correct, well-tested tool is
the right call.

## Trade-offs

- Two files per migration (`.up.sql` + `.down.sql`) versus goose's one
  file with annotations — marginally more files, but each one is pure,
  tool-agnostic SQL.
- Yet another dependency in `go.mod` — accepted, since writing a correct
  concurrent-safe migration runner is real, non-trivial work this project
  gets no learning credit for reimplementing.

## Consequences

- Migrations live in `migrations/`, named
  `NNNNNN_description.up.sql` / `.down.sql`.
- `Makefile` gains `migrate-up` / `migrate-down` targets wrapping the
  `migrate` CLI against `docker-compose`'s Postgres service.
- CI (Phase 03 onward) runs migrations against a fresh Postgres container
  before running integration tests, proving they apply cleanly from zero
  every time — not just on a database that's evolved gradually with the
  code.

## Rejected Alternatives

- **goose**: rejected only for the single-file-with-annotations format
  being a slightly less plain-SQL fit than golang-migrate's two-file
  approach — a stylistic preference, not a correctness gap; goose is
  equally capable.
- **Hand-rolled runner**: rejected because correct migration tracking
  (concurrent-run safety, dirty-state detection after a failed migration)
  is exactly the kind of already-solved problem this project should
  reuse rather than relearn, unlike the job queue, which is a named
  learning objective.

## Follow-up

None anticipated.
