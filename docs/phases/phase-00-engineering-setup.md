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
- [ ] Product docs (vision, problem, goals, non-goals)
- [ ] CONTRIBUTING.md, SECURITY.md, CHANGELOG.md
- [ ] Phase tracker + all phase files (this file + 01-14 outlines)
- [ ] ADR README + template
- [ ] Learning docs (git, github, troubleshooting) seeded
- [ ] GitHub templates: PR template, issue forms, CODEOWNERS, dependabot
- [ ] First CI workflow (docs lint)
- [ ] CLAUDE.md, AGENTS.md
- [ ] Push, PR, review, merge
- [ ] Branch protection / ruleset on `main`
- [ ] Phase closeout report

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

*(filled in when Phase 00 is fully done — do not fill prematurely)*
