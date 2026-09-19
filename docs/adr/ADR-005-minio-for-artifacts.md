# ADR-005: MinIO for artifact storage

Status: Accepted
Date: 2026-09-19

## Context

Every pipeline run produces logs, test reports, SBOMs, and security scan
output — data too large or too binary to comfortably store as database
rows, but which must be durable, checksummed, and retrievable by a
specific pipeline run.

## Problem

Where do pipeline artifacts (logs, reports, SBOMs) live: the local
filesystem, PostgreSQL itself (`bytea`), or an object store?

## Decision Drivers

Free/local-only constraint (no paid cloud storage); an API shape that
matches what a real production system would use (so the skills transfer);
support for content-addressed storage, checksums, and presigned URLs;
operational simplicity for a solo, single-host setup.

## Options Considered

### Option A — Local filesystem

Simplest possible option: write files to a directory, store the path in
Postgres. No extra service to run.

### Option B — PostgreSQL `bytea` columns

Keeps everything in one datastore. Postgres is not designed for large
binary blobs at volume — it bloats the database, complicates backups, and
doesn't provide object-storage semantics (presigned URLs, multipart
upload) a real system would want.

### Option C — MinIO (S3-compatible object storage)

Runs locally in the compose stack, speaks the same S3 API real cloud
deployments use, and provides checksum verification and presigned URLs
natively.

## Decision

MinIO.

## Why this decision?

MinIO gives Shipyard a real object-storage API (the same one AWS S3,
GCS-via-compatibility-layers, and most production systems use) while
running entirely locally and for free. This matches the project's stated
goal of learning production-shaped patterns, not shortcuts specific to a
toy setup — code written against MinIO's S3 API would need minimal change
to point at real S3 later, which a bespoke filesystem scheme would not
offer. It also natively supports the checksum-verified, two-phase
upload-then-commit-metadata pattern Phase 08 is built around.

## Trade-offs

- One more container in the local compose stack, versus the zero-cost
  local filesystem option — accepted because the API-shape learning value
  outweighs that small operational cost.
- Slightly more setup than `bytea` columns for a very small project —
  irrelevant at Shipyard's actual artifact volume, but matters for
  learning the pattern correctly.

## Consequences

- `internal/artifact` (Phase 08) talks to MinIO via an S3-compatible SDK,
  storing only a reference (bucket, key, checksum) in PostgreSQL, not the
  binary content itself.
- The local compose stack (`deployments/compose/`) gains a MinIO service
  alongside PostgreSQL from Phase 08 onward.
- Retention and cleanup jobs (Phase 08) must delete from both MinIO and
  Postgres together, or an orphan (object without metadata, or metadata
  without object) results — this two-phase-delete concern is documented
  directly in the Phase 08 test plan.

## Rejected Alternatives

- **Local filesystem**: rejected because it teaches nothing that
  transfers to a real deployment, and lacks built-in checksum/presigned-
  URL semantics Shipyard's design already assumes.
- **PostgreSQL `bytea`**: rejected because storing large binary artifacts
  in the primary transactional database risks bloating it and
  complicating the backup/restore story this project also wants to learn
  cleanly (Phase 13's `pg_dump` runbook).

## Follow-up

None currently — revisit only if a future phase needs artifact storage
semantics MinIO doesn't provide (none currently anticipated).
