# Goals

## Product goals

1. Given a repository, automatically detect its language, package manager,
   and framework.
2. Run lint, type-check, and test stages, in parallel where dependencies
   allow, inside isolated containers.
3. Run secret scanning, dependency vulnerability scanning, and SBOM
   generation, gated by a configurable policy (block on critical CVE or
   leaked secret).
4. Build and validate a Docker image for the project.
5. Store every log, report, and artifact durably, addressable by pipeline run.
6. Emit traces, metrics, and structured logs for every stage, so a single
   trace answers "why did this take so long / why did this fail".
7. Survive a worker crash or dependency outage without losing or
   duplicating work in a way the operator cannot detect.
8. Produce a final report a human can read in under a minute.

## Learning goals (equally weighted against product goals)

- Go: interfaces, concurrency, context, graceful shutdown.
- Git internals and a real GitHub Flow: issues, branches, PRs, reviews, CI,
  releases, branch protection.
- PostgreSQL as both source of truth and durable job queue.
- Container-based isolation and its real limits.
- Supply-chain security tooling (SBOM, CVE scanning, secret scanning).
- OpenTelemetry: traces, metrics, logs, and context propagation across a
  queue boundary.
- SRE practice: SLOs, load testing, deliberate failure injection.

## Success criteria (measured, not claimed)

- A pipeline run for a real sample repo completes end-to-end locally.
- A worker killed mid-job does not lose or silently duplicate that job
  (proven by a repeatable test, not by inspection).
- A single Grafana trace shows every stage of one pipeline run with real
  timings.
- Load test results (p50/p95/p99, throughput, error rate) are recorded from
  an actual run, not estimated.
