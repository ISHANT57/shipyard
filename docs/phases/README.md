# Phase Tracker

This file is the single source of truth for "what's next". Read it before
asking; update it before ending any session of work.

```text
CURRENT PHASE:     01 — Architecture
CURRENT TASK:      Not yet started
CURRENT BRANCH:    main
CURRENT ISSUE:     None open yet for Phase 01
CURRENT OBJECTIVE: Requirements, system diagrams, threat model, and the
                    first real ADRs (Go, monorepo layout, Postgres as
                    queue, Docker as execution isolation, MinIO artifacts).
BLOCKERS:          None
NEXT ACTION:       Open the first Phase 01 issue and start with the
                    system-overview diagram before any ADR — see
                    phase-01-architecture.md.
```

## Milestone map

| Milestone | Phases | Release |
| --------- | ------ | ------- |
| v0.1 Foundation | 00–03 | v0.1.0 |
| v0.2 Pipeline Engine | 04–05 | v0.2.0 |
| v0.3 Secure Execution | 06–08 | v0.3.0 |
| v0.4 Observability | 09–11 | v0.4.0 |
| v1.0 Shipyard | 12–14 | v1.0.0 |

## Phase index

- [Phase 00 — Engineering Setup](phase-00-engineering-setup.md)
- [Phase 01 — Architecture](phase-01-architecture.md)
- [Phase 02 — Go Foundation](phase-02-go-foundation.md)
- [Phase 03 — PostgreSQL](phase-03-postgresql.md)
- [Phase 04 — Durable Queue](phase-04-durable-queue.md)
- [Phase 05 — Pipeline DAG](phase-05-pipeline-dag.md)
- [Phase 06 — Execution Plane](phase-06-execution-plane.md)
- [Phase 07 — Security Pipeline](phase-07-security-pipeline.md)
- [Phase 08 — Artifacts](phase-08-artifacts.md)
- [Phase 09 — Observability](phase-09-observability.md)
- [Phase 10 — CLI + UI](phase-10-cli-ui.md)
- [Phase 11 — GitHub Engineering](phase-11-github-engineering.md)
- [Phase 12 — Reliability](phase-12-reliability.md)
- [Phase 13 — Hardening](phase-13-hardening.md)
- [Phase 14 — v1.0 Release](phase-14-release.md)

## Standard loop for every task, in every phase

```text
Concept taught
  -> design note (+ ADR if a real decision)
  -> gh issue create
  -> git switch -c <type>/issue-<n>-<slug>
  -> small implementation step
  -> tests
  -> review pass (adversarial, not a rubber stamp)
  -> fix
  -> commit (Conventional Commits)
  -> push
  -> gh pr create (Fixes #<n>)
  -> CI green
  -> merge
  -> git pull, delete branch
  -> update this file + docs/learning/*
```

Every phase ends with a report: what was built, what was learned, what
decisions were made (and why), what failed, what was fixed, which tests
pass, which Git concepts were used, which GitHub concepts were used, what
remains, what technical debt was accepted, and what the next phase is.
Work stops at that report until the next session begins.
