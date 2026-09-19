# Control Plane

## Responsibility

The control plane is the trusted part of Shipyard. It:

- accepts a project/pipeline submission (API, CLI, or GitHub webhook),
- validates and stores metadata in PostgreSQL,
- enqueues jobs and tracks their state,
- evaluates security policy against a completed run's findings,
- records every privileged action to an append-only audit log,
- serves query endpoints (status, logs, reports) back to the client.

## Hard rule

**The control plane never executes code from a submitted repository.**
Not for detection, not for a "quick check", not ever. Every operation that
touches repository content — cloning it, running a command in it,
scanning it — happens in the execution plane, inside a sandbox. This is
the single most important boundary in the whole system; see
[threat-model.md](threat-model.md) for why.

## Sub-components

| Component | Responsibility |
|---|---|
| API | HTTP surface: submissions, status queries, auth, webhooks |
| Scheduler | Turns a pipeline spec into ready-to-run jobs, respecting the DAG (Phase 05) |
| Policy engine | Evaluates security findings against configured thresholds -> PASS/WARN/BLOCK (Phase 07) |
| Audit log | Records who did what, when — append-only, never edited |

## What the control plane trusts

- Its own database.
- Authenticated API callers, within their granted role (Phase 10 RBAC).
- Nothing about the *content* of a submitted repository — that is
  execution-plane territory, and even there, treated as hostile by
  default.

## Failure posture

If the control plane is unavailable, no new work can be submitted or
scheduled, but jobs already claimed by a worker continue running — the
control plane is not on the critical path of an in-progress job's
execution, only its scheduling and result recording. This constraint
shapes the queue design in Phase 04.
