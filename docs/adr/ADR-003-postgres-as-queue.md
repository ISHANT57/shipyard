# ADR-003: PostgreSQL as the job queue

Status: Accepted
Date: 2026-09-19

## Context

Shipyard needs a durable job queue for pipeline stages: jobs must survive
a worker crash, retry with backoff, and never be silently lost or
processed in a way the operator can't observe. PostgreSQL is already the
system of record for projects, pipelines, and audit data. A separate
message broker is one more moving part to run, secure, and understand.

## Problem

What technology backs the durable job queue: PostgreSQL itself, Redis,
a dedicated broker (RabbitMQ), or a purpose-built queue library?

## Decision Drivers

Operational simplicity (fewer moving parts to run locally, for free);
transactional consistency between "job state" and "the business data that
job affects" (both live in the same database, so both can commit or roll
back together); learning value — building the queue mechanics directly
(`SELECT ... FOR UPDATE SKIP LOCKED`, leases, backoff) is a named Phase 04
learning goal, which a managed broker would bypass entirely; the
project's free/local-only constraint.

## Options Considered

### Option A — PostgreSQL (`SELECT ... FOR UPDATE SKIP LOCKED`)

Jobs are rows in a table. Workers claim a row transactionally; the row
lock plus `SKIP LOCKED` lets concurrent workers claim different rows
without blocking each other or double-claiming the same one.

### Option B — Redis (lists / streams)

Fast, simple primitives (`BRPOPLPUSH`, or Streams with consumer groups).
Requires running and operating a second stateful service, and durability
depends on Redis persistence configuration (AOF/RDB) rather than the
same ACID guarantees Postgres already provides for the rest of the data.

### Option C — RabbitMQ (or another dedicated broker)

Purpose-built for messaging: routing, exchanges, delivery guarantees.
Explicitly excluded by
[docs/product/non-goals.md](../product/non-goals.md) — a broker is real
operational surface (clustering, disk alarms, protocol tuning) this
project has no problem that requires solving.

### Option D — A Go queue library on top of Postgres (e.g. River)

Gets the "Postgres as queue" benefit without building the mechanics —
but building the mechanics directly is the named Phase 04 learning goal;
using a library would remove exactly the concept this phase exists to
teach.

## Decision

PostgreSQL, with the queue mechanics (claim, lease, heartbeat, retry,
DLQ) built directly in `internal/queue`, no external broker and no queue
library.

## Why this decision?

`SELECT ... FOR UPDATE SKIP LOCKED` gives correct concurrent claiming
without a second stateful service. Because job state lives in the same
database as pipeline and project state, a job's completion and the
business-data update it causes can commit in the same transaction —
removing an entire class of dual-write consistency bugs that a
Postgres-plus-Redis split would introduce. It also directly serves the
project's Phase 04 learning goal: transaction boundaries, row locking,
lease-based crash recovery, and backoff are exactly what this ADR commits
to building, not importing.

## Trade-offs

- Throughput ceiling is lower than a purpose-built broker at very high
  message rates — not a concern at this project's scale (Phase 12's load
  tests go up to 100 concurrent submissions, well within what a
  correctly-indexed Postgres table handles).
- Polling (even with a `LISTEN`/`NOTIFY` hint, decided in ADR-009,
  Phase 04) adds latency compared to a push-based broker — acceptable
  given the queue-wait SLO is a target to measure, not a hard real-time
  requirement (see [docs/architecture/system-overview.md](../architecture/system-overview.md)).
- Building queue mechanics by hand carries real correctness risk (double
  claims, lost leases) that a battle-tested broker would have already
  solved — mitigated by the concurrency test plan in Phase 04
  (N workers x M jobs, crash-mid-job, `-race`).

## Consequences

- Delivery semantics are **at-least-once**, with idempotent processing at
  the consumer — never a claim of exactly-once job execution (formalized
  in ADR-008, Phase 04, once that phase begins).
- The job row carries not just "what to run" but also the caller's trace
  context, so a trace survives the queue hop (see
  [docs/architecture/data-flow.md](../architecture/data-flow.md)).
- No new infrastructure dependency beyond PostgreSQL, which the project
  already requires.

## Rejected Alternatives

- **Redis**: rejected because it adds a second stateful service with
  weaker durability guarantees than Postgres already provides, for a
  workload that doesn't need Redis's raw throughput.
- **RabbitMQ**: rejected outright per
  [non-goals.md](../product/non-goals.md) — no problem in this project's
  scope requires a dedicated broker's feature set.
- **A queue library (e.g. River)**: rejected specifically because it
  would remove the hands-on learning this phase is built around, not
  because it is technically unsuitable.

## Follow-up

ADR-008 (claim/retry semantics), ADR-009 (claim strategy: polling vs.
`LISTEN`/`NOTIFY`), and ADR-010 (retry/backoff policy) — all in Phase 04
— refine the mechanics this ADR commits to.
