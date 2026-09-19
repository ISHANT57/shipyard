# Phase 12 — Reliability

## Purpose

Measure real behavior under failure and load. No invented numbers, ever.

## Failure experiments

Each one documented as: Failure / Detection / Recovery / Final state /
User-visible result / Telemetry. Cases: worker `SIGKILL` mid-job, Postgres
restart, MinIO outage, scanner crash, job timeout, duplicate submission,
duplicate webhook, queue overload (backpressure — bounded enqueue or a
`429`). Scripts in `scripts/chaos/*.sh` using `docker kill/stop/pause`.

## Load testing

k6 runs at 10, 25, 50, and 100 concurrent submissions. Recorded: throughput,
p50/p95/p99 latency, error rate, queue delay, worker saturation. Results,
raw output, and the hardware they were measured on go into
`docs/testing/benchmarks.md`.

## Git to learn

A second `git bisect`, this time on a performance regression, using
`git bisect run` with an automated check script.
