# ADR-004: Docker as execution isolation

Status: Accepted
Date: 2026-09-19

## Context

Shipyard's execution plane must run arbitrary, untrusted repository code
(lint, test, build commands) without that code compromising the worker
host, other jobs, or the rest of the system. This is the platform's
central security problem — see
[docs/architecture/threat-model.md](../architecture/threat-model.md).

## Problem

What isolation mechanism runs untrusted pipeline stages: standard Docker
containers, a sandboxed container runtime (gVisor), a microVM
(Firecracker), or a syscall-filtering wrapper (nsjail)?

## Decision Drivers

Isolation strength against a hostile workload; operational complexity for
a solo, free/local-only project (no cloud VM infrastructure); how well the
mechanism's actual failure mode can be understood and explained, not just
configured; ease of local development and debugging.

## Options Considered

### Option A — Standard Docker (namespaces + cgroups + capability drop + seccomp default profile)

The most common isolation layer; well documented; runs anywhere Docker
runs, no additional host setup. Its isolation depends on the shared host
kernel — a kernel-level exploit inside the container can, in principle,
escape it.

### Option B — gVisor (`runsc`)

A user-space kernel that intercepts syscalls before they reach the real
host kernel, meaningfully narrowing the attack surface a container escape
would need. Adds a runtime dependency and a small performance overhead;
some syscalls or features pipeline stages might need are not supported.

### Option C — Firecracker (microVMs)

Strong, VM-level isolation (used by AWS Lambda). Considerably more
operational complexity to set up and run locally, and is designed around
a host/orchestration model this project's local-first, free-only scope
does not need.

### Option D — nsjail / bwrap (syscall filtering, no container runtime)

Lighter weight than a full container, but reimplements — with more manual
effort and less community-tested defaults — much of what Docker's
namespace/cgroup/capability model already provides.

## Decision

Standard Docker, with an explicit hardened profile (documented in
[docs/architecture/execution-plane.md](../architecture/execution-plane.md)
and built in Phase 06): non-root, all capabilities dropped,
`no-new-privileges`, read-only root filesystem, resource limits, network
disabled by default. gVisor is left as a documented, revisitable upgrade
path (see Follow-up), not built into v1.0.

## Why this decision?

Docker is already a hard dependency of this project (image building,
compose stack) and is the isolation mechanism the original project
concept and every phase plan assumes — introducing gVisor or Firecracker
now would add real operational and learning surface that competes with,
rather than serves, the project's Phase 06 goal: understanding namespaces,
cgroups, capabilities, and their real limits directly. A hardened standard
Docker profile is a correct, honest baseline as long as its limitation is
stated plainly and repeatedly, which this ADR, the execution-plane doc,
and SECURITY.md all do.

## Trade-offs

- **This is the central, explicitly accepted risk of the whole project:**
  a capable container-escape exploit is not fully mitigated by Docker
  namespaces/cgroups alone. Standard Docker isolation is defense-in-depth,
  not an absolute boundary.
- Choosing not to add gVisor now means Phase 06's fixtures (timeout,
  resource-limit, network-attempt) test the *configured* boundary, not
  kernel-level escape resistance — that gap is deliberate and documented,
  not accidental.

## Consequences

- Every sandboxed stage runs with the full hardened profile described in
  [execution-plane.md](../architecture/execution-plane.md) — no stage is
  exempt without a documented reason (the dependency-install stage's
  network access is the one standing exception).
- [SECURITY.md](../../SECURITY.md) states this limitation as a known,
  permanent constraint of the v1.0 design, not a bug to be silently fixed
  later.
- Phase 13 (Hardening) explicitly revisits whether the Phase 06 fixture
  results justify adding gVisor before v1.0, using evidence rather than
  assumption.

## Rejected Alternatives

- **gVisor**: not rejected outright — deferred. It is the natural next
  step if Phase 06/13 testing shows standard Docker's boundary is
  insufficient for this project's threat model, and is named explicitly
  as that path rather than left unconsidered.
- **Firecracker**: rejected for this project's scope — its operational
  model (host-level VM orchestration) doesn't fit a local-first,
  single-operator system and would consume learning time better spent on
  the queue, DAG, and observability phases.
- **nsjail/bwrap**: rejected because it would mean reimplementing
  Docker's own isolation primitives with less tooling and community
  scrutiny behind the result, for no isolation benefit over a hardened
  Docker profile at this project's threat level.

## Follow-up

Phase 06 builds and tests the hardened profile against real fixtures.
Phase 13 (Hardening) revisits gVisor adoption using that evidence.
