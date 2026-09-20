# ADR-008: Delivery semantics

Status: Accepted
Date: 2026-09-20

## Context

[ADR-003](ADR-003-postgres-as-queue.md) committed to PostgreSQL as the
job queue. A worker can crash before running a job, during it, or after
finishing it but before recording the result. The queue cannot tell these
cases apart from the outside, so it must pick a delivery guarantee.

## Problem

What delivery semantics does the job queue promise: at-most-once,
at-least-once, or exactly-once?

## Decision Drivers

No job may be silently lost (a lost security scan reads as a pass);
correctness must be provable by a test, not asserted; the guarantee must
be achievable without distributed transactions across Postgres and a
container runtime.

## Options Considered

### Option A — At-most-once

Claim a job and mark it done immediately. A crash loses the job.

### Option B — At-least-once with idempotent handlers

A claim is a time-limited lease. A crash lets the lease expire and the
job is delivered again. A job may therefore run more than once, so
handlers must tolerate re-execution.

### Option C — Exactly-once

Not achievable in general: a worker crash between "side effect done" and
"result recorded" cannot be made atomic across two systems. Any system
claiming it is at-least-once plus deduplication.

## Decision

At-least-once delivery, idempotent processing, and **fenced completion**:
`Complete` and `Fail` only succeed when
`locked_by = <worker> AND attempt = <attempt>` still matches the row. A
worker that lost its lease (slow, paused, partitioned) cannot overwrite
the outcome of the worker that took the job over.

Exactly-once is claimed for one operation only: **pipeline creation**
for a given idempotency key (unique constraint, Phase 03). It is never
claimed for job execution.

## Why this decision?

Option A can lose security scans silently. Option C is not implementable.
Option B is the only honest choice, and fencing closes its one dangerous
gap, a stale worker reporting a result after being replaced.

## Trade-offs

- Handlers (sandbox stages, scans, artifact uploads) must be safe to run
  twice. Later phases design for this: artifact keys are content-addressed
  (Phase 08) and stage output is overwritten, not appended.
- A slow-but-alive worker may do work whose result is discarded.

## Consequences

- `job_attempts` records every attempt, so a duplicate run is visible,
  not hidden.
- Tests prove the guarantee directly: N workers x M jobs each reach
  `succeeded` exactly once, and a worker killed mid-job leaves a job
  that another worker completes.

## Rejected Alternatives

- **At-most-once**: silent loss is unacceptable for a validation platform.
- **Exactly-once**: rejected as a claim, not just as a design.

## Follow-up

ADR-009 (claim strategy), ADR-010 (retry policy).
