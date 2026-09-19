# Phase 05 — Pipeline DAG

## Purpose

Turn a flat job queue into a declarative pipeline: stages with
dependencies, some parallel, some sequential.

## Concepts to learn

Directed acyclic graphs; topological sort (Kahn's algorithm); cycle
detection; fan-out/fan-in; per-stage state machine; skip-on-upstream-failure
semantics; cancellation propagated via `context`.

## Build

`internal/pipeline` (spec parsing/validation, a scheduler that enqueues
ready stages once their dependencies succeed, aggregate pipeline status).
`internal/detect` interface with Node.js and Python detectors (language,
package manager, framework, test command, Dockerfile presence). Default
DAG: `detect -> {lint, test, security} -> build -> artifact -> report`.
Fixture repos: `test-fixtures/{node-success,node-failure,python-success,
python-failure}`.

## ADRs

ADR-011: pipeline spec format (a YAML file checked into the target repo vs
auto-generated purely from detection). ADR-012: where scheduling lives —
control plane vs worker.

## Tests

Cycles are rejected; a diamond-shaped DAG executes in correct order;
failure in one branch correctly skips or fails dependents; mid-pipeline
cancellation stops in-flight work cleanly.

## Git to learn

`git cherry-pick` (backporting a fix to another branch), `git worktree`
(reviewing one branch while working on another, without stashing).

## Release

`v0.2.0`.
