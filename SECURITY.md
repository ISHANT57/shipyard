# Security Policy

Shipyard is a learning project and a proof of concept. It is **not** audited
and carries no security guarantee beyond what is explicitly documented here.

## Current status

Pre-0.1. No released version exists yet. No version is "supported" until
`v0.1.0` ships.

## Threat model (summary)

Full detail lives in `docs/architecture/threat-model.md` once Phase 01
writes it. Until then, the working assumption is:

- Shipyard may execute **untrusted repository code**. The execution plane
  treats every submitted repository as hostile by default.
- Container-based isolation (Docker) is **defense-in-depth**, not an
  absolute security boundary. A determined attacker with container-escape
  capability is out of scope for what Shipyard alone can stop.
- The control plane never executes repository code directly — only the
  execution plane, inside a sandbox, does.

## Known limitations

Updated as they're discovered. As of Phase 00: none yet — no code exists.

## Reporting a concern

This is a personal project; open a
[private security advisory](https://github.com/ISHANT57/shipyard/security/advisories)
on the repository rather than a public issue.
