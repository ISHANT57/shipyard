# Threat Model

## Scope

This covers Shipyard as designed through v1.0 (Phase 14): a local-first
platform, not a multi-tenant hosted service. It will be revisited at the
end of every phase that changes a trust boundary, and reviewed fully in
Phase 13 (Hardening).

## Assets

| Asset | Why it matters |
| --- | --- |
| PostgreSQL data (projects, pipelines, audit log) | Source of truth; tampering undermines every report Shipyard produces |
| MinIO artifacts (logs, SBOMs, scan reports) | Evidence backing every pass/fail decision |
| API credentials / tokens (Phase 10) | Control plane access |
| The host running the worker | Where sandboxed (and potentially escaping) code executes |
| GitHub App credentials (Phase 11) | Access to real repositories and the ability to post Check results |

## Attackers considered

1. **A hostile or compromised submitted repository.** The primary
   threat. Its code runs inside the execution plane by design — the
   question is what it can reach from there.
2. **A malicious or buggy API caller** (once auth exists, Phase 10):
   attempts privilege escalation, injection, or resource exhaustion
   against the control plane.
3. **A network attacker** intercepting or forging webhook deliveries
   (Phase 11).
4. **Supply-chain compromise** of a tool Shipyard itself depends on
   (a scanner, a base image, a Go module).

*Not* in scope: a fully trusted operator misusing their own
already-granted access, or physical access to the host.

## STRIDE, by component

### Execution plane (sandbox + worker)

| Threat | Mitigation | Residual risk |
| --- | --- | --- |
| Tampering — container escape reaching host or other jobs | Non-root, capabilities dropped, read-only rootfs, resource limits (Phase 06) | Not eliminated — see explicit limitation below |
| DoS — resource exhaustion (fork bomb, memory hog) | CPU/mem/PID limits, wall-clock timeout | Bounded, not zero — a limit set too high still degrades the host |
| Info disclosure — sandbox reads worker's Docker socket or host filesystem | No host mounts, no Docker socket inside the sandbox itself | Worker process holding the socket remains a single point of elevated risk (see execution-plane.md) |
| Spoofing — forged job claiming another worker's lease | Postgres row lock + lease token, not just a job ID | Low — depends on Phase 04 queue correctness |

**Explicit limitation, stated plainly:** container-based isolation is
defense-in-depth. A capable container-escape exploit is not fully
mitigated by Shipyard alone. This is why network access defaults to
none, why the sandbox never holds the Docker socket, and why this line
appears in three separate documents rather than being assumed once.

### Control plane (API, scheduler, policy)

| Threat | Mitigation | Residual risk |
| --- | --- | --- |
| Elevation of privilege — caller acts outside their role | RBAC (Phase 10), enforced server-side on every write | None until Phase 10 ships — currently no auth exists at all |
| Tampering — audit log edited to hide an action | Append-only table, no update/delete path exposed | A direct database compromise still bypasses this |
| Repudiation — no record of who triggered a run or overrode policy | Audit log records actor + action for every privileged operation | Depends on auth existing (Phase 10) |
| DoS — submission flood | Backpressure / rate limiting (Phase 12 load testing informs the real limits) | Not yet measured — see Phase 12 |

### Webhook ingestion (Phase 11, planned)

| Threat | Mitigation | Residual risk |
| --- | --- | --- |
| Spoofing — forged webhook payload | HMAC signature verification | A leaked webhook secret defeats this |
| Replay — duplicated delivery triggers duplicate work | Idempotency by GitHub delivery ID | None significant once implemented |
| SSRF — a submitted repository URL points at an internal service | Scheme/host allowlist on submission | Allowlist must be kept current |

## Blast radius summary

- Execution-plane compromise -> that worker's host, bounded by the
  isolation baseline. Should not reach PostgreSQL, MinIO, or other
  workers directly (no shared credentials, no host mounts).
- Control-plane compromise -> full read/write of Shipyard's own state
  (high impact), but does not itself grant code execution on a worker
  host (workers pull jobs, they don't accept arbitrary commands pushed
  from the control plane beyond "run stage X of pipeline Y").
- Supply-chain compromise of a scanning tool -> false negatives in
  security results (a BLOCK policy might wrongly PASS) — mitigated in
  Phase 07 by pinning tool versions/digests, not eliminated.

## Open questions carried into later phases

- Exact rate-limit numbers for the control plane — deferred to Phase 12,
  where they can be set from measured load-test data instead of a guess.
- Whether the worker host itself needs additional OS-level hardening
  (e.g. gVisor) beyond standard Docker — revisited in Phase 13 if the
  Phase 06 fixtures suggest standard isolation is insufficient.
