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

## Tasks

- [x] `go.mod`
- [x] `internal/config` (env-based, fail-fast validation, tested)
- [x] `cmd/shipyard-api`: `/healthz`, `/readyz`, request-ID + logging middleware
- [x] Graceful shutdown on SIGTERM — proven by test and by a live smoke run
- [x] `Makefile` (fmt, fmt-check, vet, lint, test, build, clean)
- [x] `.golangci.yml`
- [x] `ci.yml`: gofmt check, vet, lint, `test -race -cover`, build, Docker build
- [x] Multi-stage, non-root Dockerfile (distroless base)
- [x] ADR-006: HTTP router
- [x] Push, PR, CI, merge
- [x] Phase closeout report

## Phase closeout report

**What I built.** The first application code: `internal/config` (env-based,
fail-fast validated, 91.7% test coverage) and `cmd/shipyard-api` — a
`/healthz`/`/readyz` HTTP server with request-ID and logging middleware,
and graceful shutdown on SIGTERM proven both by an automated test and by
a live smoke run (start, curl both endpoints, send SIGTERM, confirm clean
drain and exit 0). Added the project's real engineering scaffolding:
`Makefile`, `.golangci.yml`, a `ci.yml` workflow (gofmt check, vet, lint,
`test -race -cover`, build, Docker build), and a multi-stage, non-root
Dockerfile on a distroless base — verified directly (`docker inspect`
shows `User: 65532:65532`, and the image has no shell at all). ADR-006
records the HTTP router decision (standard library, not `chi`).

**What I learned.** Go's module system and `internal/` visibility in
practice, not just in the abstract. Why `newMux` had to return
`http.Handler`, not `*http.ServeMux`, once middleware wraps it — a real
compile error, not a hypothetical one. Context keys as unexported struct
types, to avoid collisions with other packages. `signal.NotifyContext`
turning OS signals into context cancellation, and `http.Server.Shutdown`
actually waiting for an in-flight handler — proven with a real listener
and a deliberately slow handler, timed to confirm it waited at least as
long as the handler took. Table-driven tests with `t.Run` subtests.
`errcheck` as a linter category: an unchecked `resp.Body.Close()` error
is a real, if minor, category of bug it exists to catch. Multi-stage
Docker builds: why `CGO_ENABLED=0` is what makes a distroless final image
possible at all (no libc dependency to satisfy).

**Decisions made.** ADR-006: the standard library's `net/http.ServeMux`
(Go 1.22+ method-pattern matching) over `chi`, because this project's
routes stay flat and the standard library already covers everything
needed — not a claim that `chi` is worse in general.

**What failed / what I fixed.** Two real compile/config errors, both
fixed by understanding the actual cause rather than guessing:

1. `newMux` was declared to return `*http.ServeMux` but middleware
   wrapping produces an `http.Handler` interface value — a genuine type
   error, fixed by correcting the return type and explaining why the
   interface is the honest signature here.
2. `.golangci.yml` was written in golangci-lint v1's schema (no
   `version` field). Verifying locally with the actual installed
   golangci-lint (v2.13.2) failed immediately with a clear schema error,
   which was fixed by rewriting the config to v2's schema
   (`version: "2"`, `linters.default: none`, `linters.exclusions.paths`).
   CI then failed anyway, for the mirror-image reason: `golangci-lint-action@v6`
   defaults to installing golangci-lint **v1**, which rejected the
   now-correct v2 config. A web search confirmed `golangci-lint-action@v7`
   is the version built for golangci-lint v2; bumping the action version
   fixed it. This is the same class of bug that hit the docs linter twice
   in Phase 00/01 — a tool's "latest" resolving differently in two places
   — but caught mostly before pushing this time, because the lesson from
   those earlier incidents was applied proactively (install and run the
   real linter locally before trusting the config).
Also caught, before either was ever pushed: an `errcheck` finding on
`resp.Body.Close()` in the shutdown test, found by running the real
linter locally first.

**Tests passed.** `go test ./... -race -cover`: all tests green, no race
detected. Coverage: 91.7% on `internal/config`, 45.8% on
`cmd/shipyard-api` (lower here because `main()`/`run()`'s top-level
wiring isn't exercised by a unit test — acceptable at this stage, revisited
if it becomes a real gap). `gofmt -l .` clean. `go vet ./...` clean.
`golangci-lint run ./...` clean (0 issues), verified with the exact
config CI now uses. Docker image builds, runs as UID 65532, serves
`/healthz` correctly when actually run.

**Git concepts used.** Nothing new beyond earlier phases — this phase's
new territory was Go and CI tooling, not Git itself. One small housekeeping
fix along the way: an earlier branch name typo (`issue-l1` instead of
`issue-11`, a letter "l" instead of the digit) was caught and fixed with
`git branch -m` plus a remote branch rename, before any real work had
landed on it.

**GitHub concepts used.** A PR that needed two follow-up fix commits
before its CI (a new, code-focused `ci` workflow, running alongside the
existing `docs` workflow) went green — both required checks now run on
every PR against `main`.

**What remains.** Nothing deferred from this phase's stated scope.

**Technical debt.** `cmd/shipyard-api/main.go`'s top-level `run()` wiring
has lower test coverage (45.8%) than the packages it calls into — noted
here rather than ignored; likely addressed naturally once Phase 03 adds
real dependencies (a database) that `run()` needs to wire up and test
against.

**Next phase.** [Phase 03 — PostgreSQL](phase-03-postgresql.md): schema,
migrations, `internal/store`, and the first release, `v0.1.0`.
