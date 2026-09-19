# ADR-006: HTTP router

Status: Accepted
Date: 2026-09-19

## Context

`cmd/shipyard-api` needs to route incoming requests to handlers. Go 1.22
changed the standard library's `http.ServeMux` to support method-specific
patterns (`"GET /healthz"`) and wildcards, closing most of the gap that
used to justify reaching for a third-party router.

## Problem

Does the API use the standard library's `net/http.ServeMux`, or a
third-party router such as `chi`?

## Decision Drivers

Fewer dependencies (one less thing to audit, update, and explain);
learning value — understanding what the standard library now covers
directly, rather than reaching for a library out of habit; the API's
actual routing needs, which are modest (flat routes, no deep nested
sub-routers) at least through Phase 10.

## Options Considered

### Option A — `net/http.ServeMux` (standard library)

Since Go 1.22: method-specific patterns (`"GET /healthz"`), wildcard path
segments (`"/pipelines/{id}"`), and longest-pattern-wins matching. No
import beyond the standard library.

### Option B — `chi`

A popular, lightweight third-party router: middleware chaining helpers,
sub-router mounting, and a large ecosystem of compatible middleware.
Still meaningfully more ergonomic than the standard library for deeply
nested route groups, which this project does not currently have.

## Decision

`net/http.ServeMux`, already in use in `cmd/shipyard-api/server.go`.

## Why this decision?

Go 1.22's `ServeMux` already covers everything Phase 02 needs — method
matching and simple paths — with zero extra dependencies. This project's
routes stay flat through at least Phase 10 (health checks, pipeline
submission/status, artifact retrieval); nothing in the current design
needs `chi`'s sub-router mounting or route-group middleware. Middleware
composition (seen in `withRequestID`/`withLogging`) works perfectly well
as plain function wrapping with the standard library alone.

## Trade-offs

- Wildcard path parameters are read via `r.PathValue("id")`, a plainer
  API than some routers offer — acceptable at this project's route count.
- If routing needs grow substantially (e.g. many nested resource groups
  in Phase 10's CLI/API surface), revisiting this decision is cheap: the
  handler functions themselves don't need to change, only how they are
  registered.

## Consequences

- No new dependency in `go.mod` for routing.
- Route registration stays in `newMux` (`cmd/shipyard-api/server.go`),
  one function, easy to scan end to end.

## Rejected Alternatives

- **`chi`**: rejected only because the standard library now covers this
  project's actual needs; not a claim that `chi` is a worse choice in
  general for a project with more complex routing requirements.

## Follow-up

Revisit if Phase 10's CLI-facing API surface grows nested route groups
that `ServeMux` handles awkwardly.
