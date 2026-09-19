# Phase 04 — Durable Queue

The deepest phase in the project. Goal: a Postgres-backed job queue with
at-least-once delivery and idempotent processing — built, not imported.

## Concepts to learn

Transaction boundaries; `SELECT ... FOR UPDATE SKIP LOCKED`; leases
(`locked_until`); heartbeats; visibility timeout; exponential backoff with
full jitter; max attempts and dead-letter queues; poison-job handling;
stale-lease reaping; `LISTEN`/`NOTIFY` vs polling; backpressure;
goroutines, channels, `sync.WaitGroup`, worker pools, `errgroup`; mutex vs
channel for coordination.

## Build

`internal/queue` (enqueue, claim, heartbeat, complete, fail, reap);
`cmd/shipyard-worker` with a bounded worker pool and graceful drain; job
state machine `queued -> running -> succeeded | failed -> retrying -> dead`;
`job_attempts` table; metrics hooks for later OTel wiring.

## ADRs

ADR-008: delivery semantics (at-least-once — and precisely why not
exactly-once). ADR-009: claim strategy (polling plus `NOTIFY` as a hint).
ADR-010: retry policy (backoff curve, max attempts, DLQ threshold).

## Tests

N workers x M jobs -> every job completes exactly once (verified by count,
not assumed); killing a worker mid-job -> lease expires -> job is reclaimed
and completes; a poison job reaches the DLQ after max attempts; race
detector clean; state-machine transition tests.

## Git to learn

First real `git bisect` on a planted or natural regression in the claim
logic. `git revert` of a bad commit that already reached `main`.

## Acceptance criteria

I can explain precisely what happens when two workers race to claim the
same row. The crash-recovery test passes 100 consecutive runs.
