# ADR-002: Monorepo layout

Status: Accepted
Date: 2026-09-19

## Context

Shipyard has multiple deployable units (API, worker, CLI) plus a web UI,
database migrations, deployment manifests, and documentation. A layout
decision made now determines every later `go.mod`, import path, and CI
job boundary.

## Problem

Should Shipyard's code live in one repository with one Go module, or be
split across multiple repositories/modules from the start?

## Decision Drivers

Simplicity for a solo developer; ability to share internal packages
(e.g. `internal/store`) between the API and worker binaries without a
publish/version step; a single CI pipeline and a single source of truth
for cross-cutting docs (ADRs, phase tracker); avoiding premature
distribution that this project's scale does not need yet.

## Options Considered

### Option A — Single repository, single Go module

One `go.mod` at the repo root; multiple `cmd/` entry points share
`internal/` packages directly.

### Option B — Single repository, multiple Go modules

Separate `go.mod` per component (e.g. `api/go.mod`, `worker/go.mod`),
still in one Git repository.

### Option C — Multiple repositories

Separate GitHub repositories per component, versioned and consumed as
external dependencies.

## Decision

Option A: single repository, single Go module, with `cmd/shipyard-api`,
`cmd/shipyard-worker`, and `cmd/shipyard` (CLI) as separate binaries
sharing `internal/` packages directly.

## Why this decision?

The API and worker are two processes of one system, not two independently
versioned products — they are always deployed from the same commit in
this project's design, so multi-repo versioning overhead buys nothing.
`internal/` (Go's own visibility mechanism) already prevents these shared
packages from being imported outside this module, which is the main
benefit multiple modules would otherwise provide. A single CI pipeline
also keeps the "everything green on `main`" guarantee simple to reason
about — one pipeline, one status.

## Trade-offs

- A single `go.mod` means every binary shares the same dependency
  versions; there is no scenario in this project where that's a real
  constraint, since nothing here needs conflicting versions of the same
  library.
- If Shipyard ever needed to let external projects depend on one internal
  package (unlikely, given [non-goals](../product/non-goals.md)), that
  package would need to be promoted out of `internal/` and versioned
  separately at that point — deferred, not designed for now.

## Consequences

- Repository layout follows [AGENTS.md](../../AGENTS.md)'s target
  structure: `cmd/`, `internal/`, `web/`, `migrations/`, `test-fixtures/`,
  `tests/`, `deployments/`, `docs/`.
- One `go.mod` at the repository root, created in Phase 02.
- CI runs one pipeline covering all Go binaries; the web UI (Phase 10)
  gets its own job within that same pipeline, not a separate one.

## Rejected Alternatives

- **Multiple Go modules** (Option B): adds version-pinning overhead
  between modules that share the same release cadence — no benefit at
  this scale.
- **Multiple repositories** (Option C): would require publishing internal
  packages and coordinating cross-repo releases for a system that has
  exactly one deployment target (the operator's own machine). This is
  exactly the kind of complexity [non-goals.md](../product/non-goals.md)
  rules out ("no microservices for their own sake").

## Follow-up

None — revisit only if a genuinely independent, separately-versioned
component emerges, which nothing in the current 15-phase plan implies.
