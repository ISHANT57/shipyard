# Phase 10 — CLI + UI

## Purpose

Usable interfaces to the platform. Functional, not polished.

## Build

`cmd/shipyard` CLI (`submit`, `status`, `logs`, `report`; tool choice via
ADR-017). API authentication: hashed API tokens, RBAC roles
(admin/maintainer/viewer), audit log entries for every privileged action.
React + Vite + TypeScript UI in `web/`: project list, pipeline DAG view,
live stage logs (via SSE — taught against WebSocket as the alternative),
report view, link-through to the matching Grafana trace.

## Tests

CLI golden-output tests; per-role authorization tests on the API;
Playwright smoke test on the UI.
