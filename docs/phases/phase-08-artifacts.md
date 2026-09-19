# Phase 08 — Artifacts

## Purpose

Durable, addressable storage for every report, log, and SBOM a pipeline
run produces.

## Concepts to learn

Object storage; content addressing; SHA-256 checksums; presigned URLs;
retention policy; orphan cleanup; two-phase write (upload object, then
commit its metadata row).

## Build

MinIO added to the compose stack; `internal/artifact` (checksum-verified
upload, Postgres metadata row, retention job, presigned-URL download).
Stores logs, test reports, SBOMs, scan reports, and pipeline metadata.

## Tests

MinIO unavailable mid-run -> the stage retries rather than silently losing
the artifact; checksum mismatch is detected; retention deletes both the
object and its metadata row together.

## Release

`v0.3.0`.
