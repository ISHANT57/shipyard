# ADR-010: Retry policy

Status: Accepted
Date: 2026-09-20

## Context

Jobs fail for transient reasons (a dependency briefly down) and permanent
ones (a poison job that always fails). The queue needs a retry policy
that recovers from the first without looping forever on the second.

## Problem

How are failed jobs retried, when do they give up, and where do the ones
that gave up go?

## Decision Drivers

Do not hammer a failing dependency; avoid synchronized retry storms;
bound total work per job; keep given-up jobs inspectable; keep the
claim query and its index simple.

## Options Considered

### Option A — Immediate retry

Simple, but hammers a struggling dependency.

### Option B — Fixed delay

Better, but many jobs failing together retry together.

### Option C — Exponential backoff with full jitter

Delay = random(0, min(cap, base * 2^attempt)). Spreads retries out and
grows the gap as failures continue.

## Decision

Option C with these defaults (configuration, revisited from Phase 12
data): base 5s, cap 5m, `max_attempts` 5 (already the column default).

- **Failure with attempts left**: set `status = 'queued'`,
  `run_after = now() + backoff`, clear the lease. The `retrying`
  status in migration 000001's CHECK constraint is not used.
- **Failure at `max_attempts`**: set `status = 'dead'` (the DLQ). Dead
  jobs stay in the `jobs` table, queryable and requeueable by an operator.
- **Expired lease** (worker crash): the reaper treats it as a failed
  attempt, using the same rules.
- Every attempt is recorded in `job_attempts` with its error.

## Why this decision?

Full jitter is the standard fix for retry synchronization. Returning
failed jobs to `queued` reuses the one claim path and the one partial
index instead of adding a second retry state to poll. A DLQ as a status
avoids a second table and a copy step that could itself fail.

## Trade-offs

- A `status = 'dead'` job counts toward table size until an operator
  removes it; retention is a Phase 13 concern.
- Backoff randomness makes exact retry times non-deterministic, so tests
  inject the random source.

## Consequences

- The `retrying` value stays in the CHECK constraint, unused; removing it
  would be a migration for no benefit today.
- A metric for DLQ size (`status = 'dead'`) is a required alert in Phase 09.

## Rejected Alternatives

- **Immediate retry, fixed delay**: fail the "don't hammer / don't
  synchronize" drivers.
- **Separate DLQ table**: adds a two-step move for no gain here.

## Follow-up

Poison-job handling policy for operators (requeue, inspect, purge) is
documented in the Phase 13 runbook.
