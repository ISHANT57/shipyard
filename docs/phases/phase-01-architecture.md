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
- [x] Push, PR, CI, merge
- [x] Phase closeout report

## Phase closeout report

**What I built.** Five architecture documents
(`docs/architecture/system-overview.md`, `control-plane.md`,
`execution-plane.md`, `data-flow.md`, `threat-model.md`) with two Mermaid
diagrams (a component map and a pipeline-run sequence diagram) that
render natively on GitHub. Five accepted ADRs: Go for the backend,
monorepo layout, PostgreSQL as the job queue, Docker as execution
isolation (with the defense-in-depth limitation stated explicitly), and
MinIO for artifacts. Still zero application code — this phase is pure
reasoning and documentation, as designed.

**What I learned.** The concrete difference between functional and
non-functional requirements, using Shipyard's own goals as the example.
Why a control-plane/execution-plane split exists as a trust boundary, not
just a folder convention. STRIDE applied per-component, producing a real
table of threats, mitigations, and residual risk rather than a vague
"security matters" statement. Blast-radius reasoning: what a compromise
of each component can and cannot reach. Mermaid diagram syntax
(flowchart and sequence diagrams) as a first-class, version-controlled
part of the docs, not an external image.

**Decisions made.** All five ADRs record real trade-offs, not rubber
stamps: Go was chosen over Node/TS and Python explicitly for its
concurrency-primitive fit and the project's stated learning goal, not
because the alternatives are worse in general. PostgreSQL was chosen as
the queue specifically because building the mechanics by hand is a named
Phase 04 learning goal — a queue library was rejected for removing that
learning, not for any technical shortfall. Docker's isolation limitation
(defense-in-depth, not absolute) was written down in three separate
places on purpose, rather than assumed once and forgotten. gVisor was
explicitly deferred, not rejected — Phase 13 revisits it with real
evidence from Phase 06.

**What failed / what I fixed.** CI failed twice on this PR. First: 10
`MD060` (table-column-style) errors across three new files — the tight
`|---|---|` separator style used in every new table didn't satisfy the
newer markdownlint-cli2 pulled in by Phase 00's own Dependabot bump
(exactly the risk flagged in that phase's report). Fixed by padding every
separator to `| --- | --- |`. Second: lychee's link check failed on a
genuine broken link — `ADR-003` linked forward to
`ADR-008-delivery-semantics.md`, an ADR that doesn't exist until Phase
04. Fixed by writing that forward reference as plain text instead of a
live link; a systematic check confirmed no other new file had the same
problem. Separately, I (the assistant) missed committing
`docs/phases/phase-01-architecture.md`'s own task-checklist edit in the
first commit batch — caught before pushing, folded into the lint-fix
commit instead of shipped broken.

**Tests passed.** No executable tests (none required at this stage).
Verification was: markdownlint-cli2 and lychee both green in CI, and a
local run of the same markdownlint-cli2 version used in CI confirmed
clean before each push — closing the exact gap (local linter version
mismatch) that caused friction in Phase 00.

**Git concepts used.** Nothing new beyond Phase 00's list — this phase
reused `switch`, `add`, `diff --staged`, `commit`, `push` without
introducing a new Git concept, since no history-rewriting was needed.

**GitHub concepts used.** `gh issue develop --checkout` again; a PR that
required two follow-up fix commits before merging, demonstrating that
"CI failed" is a normal, expected step in the loop, not a sign something
is broken about the process itself.

**What remains.** Nothing deferred from this phase's stated scope.

**Technical debt.** None — still no application code to carry debt.

**Next phase.** [Phase 02 — Go Foundation](phase-02-go-foundation.md): the
first application code. A minimal `cmd/shipyard-api` skeleton, Go module
setup, `/healthz`/`/readyz`, graceful shutdown, and the project's first
CI workflow that actually builds and tests code.
