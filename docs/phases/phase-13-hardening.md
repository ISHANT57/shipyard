# Phase 13 — Hardening

## Purpose

Critical review of everything built so far, before calling it v1.

## Work

Security review (revisit the threat model, check for secrets leaking into
logs, authorization gaps, SSRF via a submitted repository URL — mitigated
by a scheme/host allowlist); concurrency review (race detector run, lock
ordering check); performance profiling with `pprof`; config validation
review; a migration backup/restore runbook (`pg_dump`); retention-job
review; technical-debt issues triaged under the `technical-debt` label;
full documentation pass; optional `kind` deployment manifests, purely to
connect Kubernetes concepts (pods, probes, resource limits) to what
Shipyard already does with `/healthz`/`/readyz` and cgroup limits.

## Acceptance criteria

All P0/P1 findings are either fixed or explicitly accepted as a known
limitation in `SECURITY.md`.
