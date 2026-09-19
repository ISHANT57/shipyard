# Phase 01 — Architecture

## Purpose

Define requirements, system boundaries, and the threat model before any
code is written. Make the first real, defensible technical decisions.

## Concepts to learn

Functional vs non-functional requirements; control plane vs execution
plane; trust boundaries; STRIDE threat modeling; blast radius; at-least-once
processing semantics (preview, implemented in Phase 04).

## Deliverables

- `docs/architecture/{system-overview,control-plane,execution-plane,
  data-flow,threat-model}.md`, each with a Mermaid diagram.
- `docs/security/` baseline notes.

## Decisions requiring an ADR

- ADR-001: Go for the backend (vs Node/TypeScript, Python).
- ADR-002: monorepo layout.
- ADR-003: PostgreSQL as the job queue (vs Redis, RabbitMQ, a queue library).
- ADR-004: Docker as the execution isolation mechanism (vs gVisor,
  Firecracker, nsjail) — must state the defense-in-depth limitation.
- ADR-005: MinIO for artifact storage (vs filesystem, Postgres bytea).

## Git to learn

`git log -p`, `git diff main...branch`, amending before push,
`git commit --fixup` + `git rebase -i --autosquash` (first interactive
rebase, on an unpushed/own branch only).

## Acceptance criteria

Each ADR reaches "Accepted" status. The threat model lists assets,
attackers, mitigations, and residual risk. I can explain every trade-off
made without re-reading the ADR.

## Definition of done

Diagrams render on GitHub, ADRs merged, phase closeout report written.

## Tasks

- [x] docs/architecture: system-overview, control-plane, execution-plane, data-flow, threat-model
- [x] ADR-001: Go for the backend
- [x] ADR-002: monorepo layout
- [x] ADR-003: PostgreSQL as the job queue
- [x] ADR-004: Docker as execution isolation (defense-in-depth)
- [x] ADR-005: MinIO for artifact storage
- [ ] Push, PR, CI, merge
- [ ] Phase closeout report

## Phase closeout report

Not yet written — fill in only once Phase 01 is fully done (after merge).
