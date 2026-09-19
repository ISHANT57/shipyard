# Phase 11 — GitHub Engineering

## Purpose

Put Shipyard inside a real developer workflow, not just a standalone tool.

## Concepts to learn

Webhooks; HMAC signature verification; replay and duplicate-delivery
handling (idempotency by delivery ID); GitHub App vs personal access token
vs Actions-only integration; the Checks API; least-privilege app
permissions; a free local tunnel (e.g. smee.io) for webhook development.

## Build

A GitHub App (`checks:write`, `contents:read`, `pull_requests:write`);
`/webhooks/github` verifying signatures; push/PR events trigger a
pipeline; Check Run status and summary; a PR comment linking the report.
Alternative path considered: a reusable GitHub Action calling the Shipyard
CLI.

## ADR

ADR-018: integration model (GitHub App vs Action-only).

## Tests

An invalid signature is rejected; a duplicated webhook delivery produces
exactly one pipeline run.

## Release

`v0.4.0`, with real release notes.
