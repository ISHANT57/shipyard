# ADR-009: Claim strategy

Status: Accepted
Date: 2026-09-20

## Context

Multiple workers must claim jobs concurrently without double-claiming and
without blocking each other, and should start work promptly when a job is
enqueued.

## Problem

How do workers find and claim the next job: polling, `LISTEN`/`NOTIFY`,
or both?

## Decision Drivers

Correctness under concurrency; no lost wakeups; low idle-time load;
latency of pickup; simplicity.

## Options Considered

### Option A — Polling only

Each worker runs the claim query on an interval. Simple and always
correct, but pickup latency equals the interval and idle workers keep
querying.

### Option B — `LISTEN`/`NOTIFY` only

Enqueue sends `NOTIFY`; workers wake immediately. Notifications are not
durable: a worker that is disconnected or busy when it fires never sees
it, so jobs can sit unclaimed. Unsafe as the only mechanism.

### Option C — Polling as source of truth, `NOTIFY` as a latency hint

Workers always poll (with a moderate interval, plus jitter). `NOTIFY`
only wakes a sleeping poll loop early. A missed notification costs at
most one poll interval, never a lost job.

## Decision

Option C. The claim itself is one atomic statement:

```sql
UPDATE jobs
SET status = 'running', locked_by = $1, attempt = attempt + 1,
    locked_until = now() + $2::interval, updated_at = now()
WHERE id = (
    SELECT id FROM jobs
    WHERE status = 'queued' AND run_after <= now()
    ORDER BY run_after
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING id, stage_id, attempt, max_attempts, locked_until;
```

`FOR UPDATE SKIP LOCKED` lets concurrent workers claim different rows
without blocking or double-claiming. The existing partial index
`idx_jobs_claimable` (`WHERE status = 'queued'`) serves this query.

## Why this decision?

Correctness never depends on notifications (Option B's flaw), while
pickup latency stays low in the common case (Option A's flaw). Polling
and the atomic claim are the safety net; `NOTIFY` is an optimization that
can be removed without breaking anything.

## Trade-offs

- More moving parts than polling alone. Mitigated by making the
  `NOTIFY` path strictly optional and tested separately.
- Poll interval bounds worst-case pickup latency; measured in Phase 12,
  not guessed.

## Consequences

- Retries must return to `status = 'queued'` (see
  [ADR-010](ADR-010-retry-policy.md)) so `idx_jobs_claimable` stays
  valid; the `retrying` status is not used.
- Migration 000002 adds a partial index on `running` jobs by
  `locked_until` for the stale-lease reaper.

## Rejected Alternatives

- **Polling only**: kept as the base, not rejected; `NOTIFY` is added on top.
- **`NOTIFY` only**: rejected; not durable.

## Follow-up

Poll interval and lease length are configuration, tuned from Phase 12
measurements.
