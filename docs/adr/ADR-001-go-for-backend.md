# ADR-001: Go for the backend

Status: Accepted
Date: 2026-09-19

## Context

Shipyard's backend needs to run a control-plane API, a durable job queue
with concurrent workers, and a sandbox orchestrator talking to the Docker
Engine API — all under explicit performance and correctness requirements
(no lost/duplicated jobs, bounded resource use). The existing skill set
(React, TypeScript, Node.js, Express) already covers a capable backend
stack. This project's stated purpose is also to learn areas of weakness,
concurrency and systems programming chief among them.

## Problem

Which language and runtime should the control plane and worker be
written in?

## Decision Drivers

In order of weight for this project:

1. **Learning value** — this is an explicit, stated project goal, weighted
   above convenience.
2. **Fit for concurrent, I/O-bound systems programming** — the queue and
   sandbox orchestrator are exactly this kind of problem.
3. **Operational simplicity** — single static binary, easy to containerize
   and deploy without a runtime dependency.
4. **Ecosystem fit** — mature libraries for Postgres, Docker's API, and
   OpenTelemetry.
5. **Familiarity** — lowest weight here, deliberately, since familiarity
   is not a learning goal.

## Options Considered

### Option A — Go

Compiled, statically typed, goroutines and channels as first-class
concurrency primitives, small standard library that already covers HTTP,
context cancellation, and structured concurrency patterns well.

### Option B — Node.js / TypeScript

Already the strongest existing skill. Single-threaded event loop with
`async`/`await`; true parallel work needs worker threads or child
processes, which is a weaker fit for a CPU-adjacent, many-concurrent-jobs
worker pool than goroutines are.

### Option C — Python

Strong ecosystem for the AI-analysis pieces the original concept
considered, but the GIL limits true parallelism for CPU-bound work, and
typing/tooling for a large concurrent service is comparatively weaker
than Go's.

## Decision

Go, for both the control-plane API and the worker.

## Why this decision?

Go's goroutines/channels model maps directly onto the worker pool and
sandbox lifecycle management this project needs, and learning that model
well is one of the project's explicit goals. It compiles to a single
binary, which simplifies the multi-stage Docker builds planned from
Phase 02 onward. Its standard library and ecosystem (`pgx`, the Docker
Engine SDK, the OpenTelemetry Go SDK) cover every subsystem in this
project's scope without exotic dependencies. Node/TS and Python were both
capable alternatives — this is not a claim that either is worse in
general, only that Go serves this project's stated learning and
concurrency goals better than reusing existing strengths would.

## Trade-offs

- Slower initial velocity than writing in an already-known stack —
  accepted, because velocity is not the primary objective here.
- A smaller pool of existing personal experience to draw on when
  debugging — mitigated by the project's teaching-first working method
  (see [CLAUDE.md](../../CLAUDE.md)).
- Go's error handling (explicit `if err != nil` everywhere) is more
  verbose than exceptions — accepted as idiomatic and, for a learning
  project, a feature: it forces every error path to be seen.

## Consequences

- `cmd/`, `internal/` package layout follows Go module conventions
  (see [ADR-002](ADR-002-monorepo-layout.md)).
- CI (from Phase 02) runs `gofmt`, `go vet`, `golangci-lint`, and
  `go test -race`.
- The web UI (Phase 10) remains React/TypeScript — this decision is
  scoped to the backend only; there is no reason to abandon an existing
  strength for the frontend.

## Rejected Alternatives

- **Node.js/TypeScript**: rejected for the backend specifically because
  it does not exercise the concurrency learning goal, not because it is
  incapable of building this system.
- **Python**: rejected primarily for the GIL's fit with a concurrent
  worker pool; its strength in AI/data tooling is not needed here since
  Shipyard's original AI-report ambition was cut from the v1.0 scope
  (see [docs/product/non-goals.md](../product/non-goals.md)).

## Follow-up

ADR-006 (Phase 02) will decide the HTTP router within Go
(standard library vs. `chi`).
