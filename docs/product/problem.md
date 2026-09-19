# Problem

When a repository is handed off — a new hire's first PR, a contractor's
deliverable, an internal team's service — the receiving side has no fast,
consistent way to answer:

- Does it build?
- Do the tests pass?
- Does it lint and type-check cleanly?
- Does it contain a leaked secret or a known-vulnerable dependency?
- Does the Docker image build and run?
- What would break in production, and how would we know?

Today this is either done by hand (slow, inconsistent, skipped under
deadline pressure) or delegated entirely to a hosted CI product that gives
pass/fail with no unified evidence trail, no reasoning about *why* something
failed, and no visibility into the pipeline's own reliability.

## Why this matters for learning

Every one of those checks is a small, well-scoped problem with real
production depth behind it: exactly how CI systems isolate untrusted code,
how job queues survive crashes, how supply-chain tools actually work, how
distributed tracing explains a slow pipeline. Building Shipyard means
building each of those pieces correctly, once, and being able to explain the
trade-offs — not gluing together hosted services.

## Non-problem

Shipyard is not trying to replace GitHub Actions, GitLab CI, or a real IDP
like Backstage. It borrows their ideas at a scale one person can build,
run, and fully understand in 8–12 weeks.
