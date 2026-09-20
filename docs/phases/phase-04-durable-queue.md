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

## Tasks

- [x] ADR-008 delivery semantics, ADR-009 claim strategy, ADR-010 retry policy
- [x] Migration 000002: partial index for the stale-lease reaper (up/down/up
      verified against a real database)
- [x] `internal/queue`: `Enqueue`, `Claim`, `Heartbeat`, `Complete`, `Fail`,
      `Reap`, `Listen`, `Backoff` — fenced, at-least-once
- [x] `internal/worker`: bounded pool, per-job heartbeat, reaper loop,
      `LISTEN`/`NOTIFY` wake-up, panic containment, graceful drain
- [x] `cmd/shipyard-worker` + non-root Dockerfile + CI image build
- [x] `internal/config`: `LoadWorker` with fail-fast parsing
- [x] Makefile: `compose-up`, `compose-down`, `migrate-up`, `migrate-down`
      (the targets ADR-007 promised)
- [x] Tests: 11 queue + 10 worker + config; mutation-checked (see below)
- [x] Live run of the real binary against the compose database
- [ ] Crash-recovery tests pass 100 consecutive runs
- [ ] Git drill: `scripts/drills/bisect-drill.sh` — bisect, then revert
- [ ] Push, PR, CI, merge
- [ ] Phase closeout report

## Test integrity: mutation checks

Green tests only matter if they can fail. Each guarantee was verified by
deliberately breaking the code and confirming a specific test goes red:

| Break | Caught by |
| --- | --- |
| Remove the fence from `Complete` | `TestComplete_IsFenced`, `TestReap_CrashedWorkerJobIsReclaimed` |
| Remove `SKIP LOCKED` | `TestClaim_SkipsRowsLockedByAnotherTransaction` |
| Heartbeat never extends the lease | `TestWorker_HeartbeatKeepsLongJobAlive` |
| Shutdown cancels in-flight handlers | `TestWorker_GracefulDrain` |
| Do not recover handler panics | `TestWorker_PanicIsContainedAndRetried` |
| Disable the `NOTIFY` listener | `TestWorker_NotifyWakesIdleWorker` |

Two of these initially slipped through: removing `SKIP LOCKED` left the
concurrency test green (row locks alone already prevent double-claims, so
that test proved exactly-once but not non-blocking), and a drain test
whose handler ignored its context could not notice shutdown cancelling
it. Both tests were rewritten to target the real property.

## Phase closeout report

Not yet written: fill in after the drill, the 100-run check, and merge.
