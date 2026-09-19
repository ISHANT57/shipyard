# AGENTS.md

Repository-level engineering handbook. Where [CLAUDE.md](CLAUDE.md) governs
how Claude behaves and teaches in this repo, this file states the
conventions **any** contributor or tool follows, regardless of who or what
is writing the code.

## Repository layout (target — built incrementally by phase)

```
cmd/                  entry points: shipyard-api, shipyard-worker, shipyard (CLI)
internal/             application code, not importable outside this module
  api/ config/ store/ queue/ pipeline/ detect/ sandbox/
  security/ artifact/ telemetry/ policy/
migrations/           reversible SQL migrations
web/                  React + TypeScript UI
test-fixtures/        sample repos used as pipeline test inputs
tests/                integration/, e2e/, load/
deployments/compose/  local docker-compose stack
docs/                 product, architecture, adr, phases, operations,
                      security, testing, runbooks, learning
.github/              issue/PR templates, workflows, CODEOWNERS, dependabot
```

Nothing under `cmd/` or `internal/` exists yet — created starting Phase 02.

## Architecture boundary

The **control plane** (API) never executes repository code directly. Only
the **execution plane** (worker, via the sandbox) runs untrusted commands,
and only inside a resource-limited, non-root container. This boundary is
enforced in code review, not just documentation.

## Coding conventions

- Go: `gofmt`+`goimports` clean, `golangci-lint` clean, no unchecked
  errors, wrap errors with context (`fmt.Errorf("...: %w", err)`), accept
  interfaces / return structs, pass `context.Context` explicitly (first
  argument), no global mutable state.
- Commits: [Conventional Commits](https://www.conventionalcommits.org/),
  one logical change per commit.
- Branches: `<type>/issue-<number>-<slug>` (see
  [CONTRIBUTING.md](CONTRIBUTING.md)).

## Test expectations

Every phase's acceptance criteria (`docs/phases/phase-NN-*.md`) must be
demonstrated by a passing, repeatable test — not by a one-off manual run.
Concurrency-sensitive code (queue, sandbox lifecycle) runs under
`go test -race`. Integration tests use real dependencies via
Testcontainers, not mocks, wherever feasible.

## Security expectations

- No secret ever committed; `.gitignore` covers `.env*` and key files.
- Every external input (webhook payload, submitted repository URL,
  pipeline spec) is validated before use.
- Sandbox defaults are deny-by-default: no network, minimal capabilities,
  resource limits, unless a stage explicitly documents why it needs more.

## Review checklist

Before merging any PR:
- [ ] Correctness — does it do what the issue asked?
- [ ] Concurrency — any new shared state? Race-checked?
- [ ] Security — new input surface validated? Least privilege held?
- [ ] Tests — do they prove the acceptance criteria, not just "it runs"?
- [ ] API design — consistent with existing endpoints/CLI conventions?
- [ ] Observability — does a new stage/operation emit a span/metric?
- [ ] Performance — any obviously avoidable O(n²) or blocking call on a
      hot path?
- [ ] Docs — ADR/phase file/learning notes updated if relevant?

## Git conventions

See [docs/learning/git.md](docs/learning/git.md) for concepts learned so
far and [CONTRIBUTING.md](CONTRIBUTING.md) for the required workflow.

## Documentation expectations

- A decision with real trade-offs → ADR (`docs/adr/`).
- A completed phase → update `docs/phases/README.md` and the phase file's
  closeout section.
- A new external dependency or tool → one line in
  `docs/operations/free-local-stack.md` once that doc exists (Phase 09).
