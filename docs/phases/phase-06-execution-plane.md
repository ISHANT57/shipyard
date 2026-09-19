# Phase 06 — Execution Plane

## Purpose

Run untrusted repository commands safely, inside throwaway containers.
This is the hardest security problem in the whole platform.

## Concepts to learn

Namespaces; cgroups v2; Linux capabilities; seccomp; user namespaces and
rootless containers; why holding the Docker socket is equivalent to root;
image pinning by digest; workspace lifecycle and cleanup.

## Build

`internal/sandbox` via the Docker Engine API (Go SDK): non-root user,
`--cap-drop ALL`, `no-new-privileges`, read-only root filesystem with a
tmpfs workspace, CPU/memory/PID limits, `--network none` for lint/test
stages (network only for the dependency-install stage, explicitly
documented), wall-clock timeout with hard kill, guaranteed cleanup (defer
plus a label-based reaper for orphans), size-capped log streaming. Source
fetched via `git clone --depth 1` into the workspace by the worker.

## Fixtures

`timeout/`, `resource-limit/` (memory hog, fork bomb capped by
`--pids-limit`), a network-access-attempt fixture.

## ADRs

ADR-013: sandbox profile and network policy. ADR-014: how the worker talks
to Docker, with the socket-as-root risk documented plus mitigations.

## Tests

Each fixture produces the correct terminal state and reason; zero
containers left behind after 50 consecutive runs.

## Docs

`docs/security/sandbox.md`, stating plainly: Docker is defense-in-depth,
not an absolute isolation boundary.
