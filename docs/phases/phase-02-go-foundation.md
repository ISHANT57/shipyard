# Phase 02 — Go Foundation

## Purpose

Learn Go through a real, minimal API skeleton rather than tutorials alone.

## Concepts to learn

Modules and packages; visibility; structs and interfaces (accept
interfaces, return structs); pointers; error wrapping (`%w`,
`errors.Is`/`As`); `context`; `net/http` and `ServeMux` routing patterns;
`log/slog`; environment-based config with fail-fast validation; graceful
shutdown (`signal.NotifyContext`, `Server.Shutdown`); table-driven tests;
`httptest`.

## Build

`cmd/shipyard-api`, `internal/config`, `/healthz` (liveness) and `/readyz`
(readiness) endpoints, request-ID middleware, `Makefile`
(`fmt`/`lint`/`test`/`build`), `golangci-lint` config, `ci.yml` (gofmt,
vet, lint, `go test -race`, build), multi-stage Dockerfile running as a
non-root user.

## ADR

ADR-006: HTTP router — standard library vs `chi`.

## Git to learn

`git stash`, `git restore --staged`, `git blame`, resolving a real merge
conflict (staged deliberately, e.g. on the Makefile).

## Acceptance criteria

API starts, shuts down cleanly on SIGTERM mid-request (proven by a test,
not observation), CI green with the race detector enabled, container image
runs as non-root.
