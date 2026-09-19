# Contributing

Shipyard is currently a solo learning project, but it follows the same
workflow a real team would use — that consistency is the point.

## Local setup

Prerequisites and setup steps land here as each phase introduces them
(Go in Phase 02, PostgreSQL/Docker Compose in Phase 03, and so on). Nothing
to install yet in Phase 00.

## Workflow

1. Every change starts as a GitHub issue, labeled and (once past Phase 00)
   assigned a milestone.
2. Branch from `main`:
   ```
   <type>/issue-<number>-<short-slug>
   ```
   `type` is one of `feat`, `fix`, `docs`, `chore`, `security`, `test`,
   `refactor`, `ci`. Example: `feat/issue-42-postgres-job-claim`.
3. Commit using [Conventional Commits](https://www.conventionalcommits.org/):
   ```
   <type>(<scope>): <short summary>
   ```
   Examples: `feat(queue): add durable job claiming`,
   `fix(worker): recover expired leases`,
   `docs(adr): document postgres queue decision`.
   Keep each commit to one logical change — not one commit per keystroke,
   not one commit for an entire feature.
4. Push the branch and open a pull request using the PR template. Reference
   the issue with `Fixes #<number>` when the PR closes it.
5. CI must pass. A review (self-review, since this is solo — see
   `docs/phases/README.md` for how that works) must be resolved before merge.
6. Merge, then delete the branch (`git branch -d`, `git fetch --prune`).

`main` is protected: no direct pushes, PRs required, CI must pass.

## Tests

Every phase's "Done when" criteria (see `docs/phases/`) must be demonstrated
by a test or a recorded command, not by inspection alone.

## Security

See [SECURITY.md](SECURITY.md) for how to report a concern and what is
and isn't covered by Shipyard's security model at this stage.

## Documentation

- A decision with real trade-offs gets an ADR (`docs/adr/`), not just a
  comment in code.
- Each phase updates `docs/phases/README.md` and, where relevant,
  `docs/learning/`.
