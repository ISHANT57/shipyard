# Phase 07 — Security Pipeline

## Purpose

Add secret scanning, dependency vulnerability scanning, SBOM generation,
and a policy gate.

## Concepts to learn

Secret scanning approaches; CVE/CVSS/EPSS; SBOM formats (CycloneDX vs
SPDX); VEX; image scanning vs filesystem scanning; false-positive
management; supply-chain attack classes (typosquatting, dependency
confusion).

## Build

`internal/security` runs Gitleaks, Trivy (filesystem and image), and Syft
inside pinned-digest sandbox images; a normalized findings model;
`internal/policy` (YAML: `max_critical`, `max_secrets`, `tests_required`,
a time-boxed allowlist) evaluating to PASS/WARN/BLOCK with reasons.

## Fixtures

`secret-detection/` (a fake credential pattern), `vulnerability/` (a
pinned, known-vulnerable dependency).

## Shipyard's own CI

Add Gitleaks, Trivy, and `govulncheck` jobs; Dependabot for Go modules and
npm; GitHub artifact attestations on Shipyard's own image (documented as
provenance, not a security guarantee).

## ADR

ADR-015: policy format and evaluation model.
