# Shipyard

A local-first developer automation platform.

Submit a repository and Shipyard detects the project type, runs a pipeline of
lint, test, security and build stages inside isolated containers, stores the
reports as artifacts, and tells you whether the project passed validation and why.

## Status

**Phase 00 — Engineering setup.** No application code exists yet.
The engineering process, documentation and GitHub workflow come first.

## Planned architecture

- **Control plane** (Go): API, pipeline scheduling, job state, policy, audit.
  It never executes repository code.
- **Execution plane** (Go + Docker): workers that claim jobs and run each stage
  in a throwaway, resource-limited container.
- **PostgreSQL** for state and the durable job queue, **MinIO** for artifacts,
  **OpenTelemetry + Grafana** for logs, metrics and traces.

Everything runs locally with free and open-source tools.

## License

[MIT](LICENSE)
