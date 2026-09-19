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
- [x] Push, PR, CI, merge
- [x] Release v0.1.0
- [x] Phase closeout report

## Phase closeout report

**What I built.** `internal/store`, a repository layer over `pgx`
(`ErrNotFound` sentinel, idempotent `CreatePipeline` via
`ON CONFLICT DO NOTHING RETURNING`); `internal/testdb`, a shared
Testcontainers-Go helper reused by both `internal/store` and
`cmd/shipyard-api` tests so migration/container setup exists in exactly
one place; the first real migration (`000001_initial_schema`, six
tables); `docker-compose.yml` for local Postgres; `POST /projects` and
`POST /pipelines` wired to the real database, with `/readyz` now
actually checking connectivity instead of always returning OK. Tagged
and published the project's first release, **v0.1.0**.

**What I learned.** Constraints as correctness in practice, not just in
principle — proved a `UNIQUE` constraint really blocks a duplicate
`idempotency_key` by attempting the insert twice via raw `psql` before
any Go code existed to do it for me. `SELECT`/`INSERT ... ON CONFLICT`
patterns and why `ON CONFLICT DO NOTHING RETURNING` reporting "no row"
is a normal outcome to branch on, not an error. `TEXT + CHECK` versus
native `ENUM` for status columns, chosen for reversibility. Reversible
migrations, proven by an actual up/down/up round-trip against a running
database, not just by writing both files and assuming. Why `internal/`
packages can still be tested with real infrastructure (Testcontainers)
instead of mocks, and why that matters specifically for conflict
semantics a mock can't faithfully reproduce. The real difference between
a **lightweight** and an **annotated** Git tag — and that
`gh release create` makes a lightweight one by default, which does not
satisfy what `git describe` or a real release process expects.

**Decisions made.** ADR-007 (already accepted before this phase's main
work): `golang-migrate` over `goose` or a hand-rolled runner.

**What failed / what I fixed.** Five real problems, in the order hit:

1. `docker compose up` collided with an unrelated project's Postgres
   already bound to host port 5432 (`goqii-communicaton-all-postgres-1`)
   — the first `migrate up` attempt actually silently authenticated
   against *that* database and failed with a confusing password error.
   Fixed by moving Shipyard's compose mapping to port 5433 and
   documenting why, rather than assuming port collisions fail loudly.
2. `go get`ting `pgx/v5` auto-bumped `go.mod`'s directive from `1.24.0`
   to `1.25.0` (later `1.25.11`, pulled in further by `testcontainers-go`)
   — a real, correct dependency requirement, not a mistake, but one that
   had a knock-on effect (see #3).
3. The Docker build then failed outright: the builder stage was still
   pinned to `golang:1.24-bookworm`, and the official golang images set
   `GOTOOLCHAIN=local`, so it refused to auto-fetch a newer toolchain
   the way local `go build` had. Fixed by bumping the builder stage to
   `golang:1.26-bookworm`, matching what local tooling had already
   resolved to.
4. A live container test using `host.docker.internal` failed to resolve
   — that hostname is a Docker Desktop (Mac/Windows) convenience not
   present on plain Linux Docker by default. Not a code bug: the app
   failed fast with a clear, correctly-formatted error and a non-zero
   exit code, exactly as `internal/config`'s fail-fast design intends.
   Fixed the test itself with `--add-host=host.docker.internal:host-gateway`.
5. `gh release create v0.1.0` produced a lightweight tag, not the
   annotated tag this phase's own Git curriculum called for. Fixed by
   deleting and recreating it with `git tag -a`, then force-pushing the
   corrected tag — confirmed via `git cat-file -t` (now reports `tag`,
   not `commit`) and `git describe --tags`.

**Tests passed.** Integration tests via Testcontainers-Go: project
CRUD, not-found handling, idempotent resubmission, distinct keys produce
distinct pipelines, and — the acceptance-criteria case specifically — 10
concurrent goroutines submitting the same idempotency key resolve to
exactly one pipeline row with exactly one reporting `created=true`,
clean under `-race`. End-to-end HTTP tests against the real mux and a
real (test) database for the new endpoints. A live smoke test of the
compiled binary and, separately, the built container image, both against
the actual compose Postgres — not only the test suite.

**Git concepts used.** First real, hands-on use of annotated tags
(`git tag -a`), `git describe --tags`, and force-pushing a corrected tag
(`git push --force` on a tag ref specifically, understood as a narrow,
deliberate exception to "don't force-push", not a general habit).

**GitHub concepts used.** `gh release create --generate-notes` for the
project's first release; confirmed a PR with substantial, high-risk
changes (real database code, concurrency, a toolchain bump) can still go
green on the first CI attempt when every part of it was already verified
against real infrastructure locally first, rather than discovered by CI.

**What remains.** Nothing deferred from this phase's stated scope.

**Technical debt.** None newly accepted. `cmd/shipyard-api` test
coverage improved (45.8% -> 52.3%) as a side effect of testing the new
endpoints end-to-end through the real mux, partially addressing the gap
noted in the Phase 02 closeout — not fully closed, since `main()`/`run()`
wiring itself is still not directly exercised by a test.

**Next phase.** [Phase 04 — Durable Queue](phase-04-durable-queue.md) —
the deepest phase in the project: `SELECT ... FOR UPDATE SKIP LOCKED`,
leases, heartbeats, backoff, dead-letter handling, and a worker pool,
built by hand on the `jobs` table this phase already created.
