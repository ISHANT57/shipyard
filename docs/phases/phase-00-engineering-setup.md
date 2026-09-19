# Phase 00 — Engineering Setup

## Purpose

Build a professionally structured empty repository and prove the real
GitHub workflow end-to-end before any application code exists.

## Why this phase exists

Every later phase reuses this workflow (issue -> branch -> PR -> CI ->
review -> merge -> release). Learning it once, deliberately, with no
application code to distract from it, means every later phase is only
learning *new* things — Go, Postgres, Docker — not also relearning process.

## Prerequisites

- Git and GitHub CLI (`gh`) installed and authenticated.
- A GitHub account.

## Concepts I must learn

- Git's object model: blob, tree, commit, ref.
- The difference between Git (local history) and GitHub (hosted
  collaboration + API).
- GitHub Flow: issue, branch, PR, CI, review, merge.
- What CLAUDE.md, AGENTS.md, ADRs, phase docs, issue templates, and branch
  protection are each *for*.

## Design questions

- Where does AI-assistant behavior belong (CLAUDE.md) versus general
  repository conventions any contributor or tool should follow (AGENTS.md)?
- What belongs in an ADR versus a phase doc versus a README?

## Decision points

None requiring an ADR in this phase — Phase 01 makes the first real
technical decisions (language, architecture). Phase 00 is process only.

## Tasks

- [x] `git init`, bootstrap commit (README, .gitignore, .editorconfig, LICENSE)
- [x] `gh repo create`, public, connected as `origin`
- [x] Labels, milestones created
- [x] Issue #1 opened, branch created and linked
- [x] Product docs (vision, problem, goals, non-goals)
- [x] CONTRIBUTING.md, SECURITY.md, CHANGELOG.md
- [x] Phase tracker + all phase files (this file + 01-14 outlines)
- [x] ADR README + template
- [x] Learning docs (git, github, troubleshooting) seeded
- [x] GitHub templates: PR template, issue forms, CODEOWNERS, dependabot
- [x] First CI workflow (docs lint)
- [x] CLAUDE.md, AGENTS.md
- [x] Push, PR, review, merge
- [x] Branch protection / ruleset on `main`
- [x] Phase closeout report

## Expected files

`README.md`, `.gitignore`, `.editorconfig`, `LICENSE`, `CLAUDE.md`,
`AGENTS.md`, `CONTRIBUTING.md`, `SECURITY.md`, `CHANGELOG.md`,
`docs/product/*`, `docs/phases/*`, `docs/adr/{README,TEMPLATE}.md`,
`docs/learning/*`, `.github/*`.

## Tests required

None — no executable code in this phase. "Tested" here means: CI (docs
lint) passes, and a direct push to `main` is rejected once the ruleset is
active.

## Git workflow

`init`, `add`, `diff --staged`, `commit`, `log`, `show`, `cat-file`,
`remote`, `push -u`, `switch -c` (via `gh issue develop`), `branch -vv`,
`fetch --prune`.

## Acceptance criteria

- Issue #1 closed by a merged PR.
- CI green on that PR.
- Direct push to `main` rejected (demonstrated once).
- `docs/phases/README.md` correctly points to Phase 01 as next.

## Definition of done

All tasks checked, phase closeout report written below, tracker updated.

## Common mistakes

- Committing directly to `main` out of habit (only the bootstrap commit is
  allowed to do this, and only because no branch/PR target existed yet).
- Creating empty directories and expecting Git to track them — it won't,
  until a file exists inside.
- Writing an ADR for a non-decision just to "fill out the folder".

## What I should be able to explain after completion

- What a commit actually is (tree + parent + metadata), not just "a save".
- Why `main` needed one direct commit and no others ever will.
- The full GitHub Flow, in my own words, without looking it up.
- Why CLAUDE.md and AGENTS.md are two different files, not duplicates.

---

## Phase closeout report

**What I built.** A professionally structured empty repository: product
docs (vision/problem/goals/non-goals), a 15-phase plan with acceptance
criteria per phase, an ADR process, CONTRIBUTING/SECURITY/CHANGELOG,
GitHub issue forms and a PR template, CODEOWNERS, Dependabot, a docs CI
workflow (markdownlint + lychee), and CLAUDE.md/AGENTS.md governing how
work proceeds from here. Proved the full GitHub Flow end-to-end: issue ->
branch -> commits -> PR -> CI -> merge -> branch deletion, three times
over (the main PR, and two Dependabot dependency-bump PRs). Protected
`main` with a ruleset (PR required, `lint-and-links` check required,
branch must be up to date, no force-push, no deletion).

**What I learned.** Git's object model (blob/tree/commit/ref) inspected
directly with `cat-file`; staging vs committing; `git rebase -i` for
rewording history before a push; remotes and upstream tracking; the
difference between Git and GitHub; issues, labels, milestones, project
boards; `gh` as a thin wrapper over GitHub's REST API (no native
`gh milestone`, so `gh api` was used directly); PR templates and required
status checks; `strict_required_status_checks_policy` and why a PR can
show green CI yet still be blocked from merging until its branch is
resynced with a moved base; Dependabot as a real, automated contributor
going through the same protected pipeline as manual work.

**Decisions made.** No ADR was written — Phase 00 was deliberately process
only, no technical trade-off big enough to warrant one. The one real
judgment call (CLAUDE.md vs AGENTS.md split: AI-assistant behavior vs
tool-agnostic repo conventions) is documented in each file's own opening
paragraph rather than a separate ADR, since it isn't a reversible-cost
engineering decision.

**What failed / what I fixed.** CI failed twice on the first PR: 13
markdownlint errors (missing fence languages, missing blank lines around
lists/fences, an emphasis-only line flagged as a heading) — fixed by
editing the actual docs, not by weakening the lint config. Then a lychee
flag (`--exclude-mail`) didn't exist in the pinned action version — fixed
by removing it (mail-link exclusion is lychee's default behavior anyway).
Later, the markdownlint-cli2-action Dependabot bump (v18 -> v24) exposed
that a locally-run newer markdownlint-cli2 enforces an extra rule
(`MD060`, table column spacing) that CI's older pinned version didn't —
fixed the one affected table regardless, since it was a real formatting
inconsistency, and confirmed CI stayed green after the bump merged.

**Tests passed.** No executable tests in this phase (none required — see
Acceptance criteria). Verification was procedural: CI green on every
merged PR, issue #1 and #5 closed automatically via `Fixes #`, and the
ruleset demonstrably blocked a stale-branch merge (PR #4) until synced.

**Git concepts used.** `init`, `add`, `diff --staged`, `commit`, `log`,
`show`, `cat-file -p`, `remote -v`, `push -u`, `branch -vv`,
`rebase -i` (reword), `switch`, `fetch --prune`, `branch -d`.

**GitHub concepts used.** Issues, labels, milestones, `gh issue develop`,
PR templates, issue forms, CODEOWNERS, Dependabot, required status
checks, branch rulesets (`gh api repos/.../rulesets`), squash merge,
`gh pr checks --watch`, `pulls/{n}/update-branch`.

**What remains.** A GitHub Projects board (Backlog/Ready/In
Progress/Review/Blocked/Done) was planned but not created — deferred, not
blocking; can be added anytime without affecting the workflow already
proven. The manual "push directly to `main` gets rejected" demonstration
was superseded by a stronger real proof (the stale-branch block on PR #4)
and was not additionally run.

**Technical debt.** None accepted yet — no application code exists to
carry debt.

**Next phase.** [Phase 01 — Architecture](phase-01-architecture.md):
requirements, system diagrams, threat model, and the first real ADRs
(language choice, monorepo layout, queue technology, execution isolation,
artifact storage).
