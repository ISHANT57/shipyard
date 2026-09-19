# Phase 09 — Observability

## Purpose

A single trace should answer: "why did this pipeline take so long, and
where exactly did it fail?"

## Concepts to learn

The three signals (traces, metrics, logs); OTel SDK/API/Collector
distinction; W3C trace-context propagation *through the job queue* (trace
context stored on the job row, not just in-process); span links; RED/USE
metrics; histogram bucket design; cardinality; exemplars; correlating logs
to traces via trace ID.

## Build

`internal/telemetry`; spans across API -> enqueue -> claim -> stage ->
container run -> artifact upload; metrics: `queue_depth`,
`queue_wait_seconds`, `job_duration_seconds`, `stage_duration_seconds`,
`pipeline_success_rate`, `retry_count`, `worker_active_jobs`,
`sandbox_duration`, `security_scan_duration`, `artifact_upload_duration`;
`slog` output carrying `trace_id`. Compose stack: OTel Collector,
Prometheus, Tempo, Loki, Grafana with provisioned dashboards and alert
rules (queue wait, DLQ growth, failure rate).

## SLOs

`docs/operations/slos.md` — e.g. 99% of pipelines reach a terminal state;
a p95 queue-wait target, verified in Phase 12 against real measurements.

## ADR

ADR-016: telemetry stack choice.

## Docs

`docs/operations/free-local-stack.md` (start/stop/reset/logs/troubleshoot),
first runbooks.
