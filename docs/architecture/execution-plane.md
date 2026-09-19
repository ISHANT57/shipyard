# Execution Plane

## Responsibility

The execution plane is where repository code actually runs, and the only
place it does. It:

- claims a job from the queue,
- fetches the target repository into a throwaway workspace
  (`git clone --depth 1`),
- runs the job's stage command inside a resource-limited, non-root
  container,
- streams and captures logs, enforces a timeout, and guarantees cleanup,
- uploads results (logs, reports, artifacts) to MinIO and reports the
  outcome back to the control plane via the job's state in PostgreSQL.

## Trust posture

Every repository submitted to Shipyard is treated as **hostile by
default** — not because most will be, but because the system cannot tell
which ones are without running them, which is exactly the operation being
protected against. See [threat-model.md](threat-model.md).

## Isolation baseline (detailed in Phase 06)

- Non-root user inside the container.
- All Linux capabilities dropped, `no-new-privileges` set.
- Read-only root filesystem; a tmpfs workspace for anything the stage
  needs to write.
- CPU, memory, and PID limits enforced by the container runtime.
- No network access by default; only the dependency-install stage is
  granted network, and that is an explicit, documented exception.
- A hard wall-clock timeout, enforced by the worker, not just the
  container's own settings.
- Guaranteed cleanup: the workspace and container are removed whether the
  stage succeeds, fails, or times out, and an orphan reaper catches
  anything a crash left behind.

## Explicit limitation

**Docker-based isolation is defense-in-depth, not an absolute security
boundary.** A sufficiently capable container-escape exploit is out of
scope for what Shipyard alone can stop. This is stated here, in
[SECURITY.md](../../SECURITY.md), and again in
[ADR-004](../adr/ADR-004-docker-execution-isolation.md), deliberately
repeated rather than assumed.

## Why the worker — not the control plane — holds the Docker socket

Holding the Docker socket is equivalent to holding root on the host: a
container can be started with privileges that escape its own sandbox.
Only the execution plane needs this capability, and only it has it. The
control plane is never given Docker access at all, which limits the
blast radius of a control-plane compromise to "no new jobs run," not
"attacker has root."
