# GitHub — what I've learned, in the order I learned it

## Phase 00

- **Git vs GitHub**: Git is the local, offline history engine. GitHub is a
  hosted service built around it, adding issues, pull requests, code
  review, Actions (CI), releases, and an API (`gh` is a CLI over that API).
- **GitHub Flow**: issue -> branch -> commits -> push -> PR -> CI -> review
  -> merge -> delete branch. `main` is never committed to directly; it
  stays deployable at all times.
- **Labels**: categorize issues/PRs (`feature`, `bug`, `security`,
  `architecture`, `documentation`, `testing`, `performance`, `learning`,
  `technical-debt`), on top of GitHub's own defaults
  (`bug`, `documentation`, `enhancement`, ...).
- **Milestones**: group issues/PRs toward a release
  (`v0.1 Foundation` ... `v1.0 Shipyard`). No native `gh milestone`
  subcommand exists — `gh api repos/OWNER/REPO/milestones` calls the REST
  API directly, which is a useful reminder that `gh` is a thin wrapper
  over the same API anyone can call.
- **Projects**: a board (Backlog/Ready/In Progress/Review/Blocked/Done)
  giving a cross-issue view of status. Created with `gh project create`;
  the default single-select "Status" field's options are edited in the
  web UI, since `gh` doesn't yet expose editing single-select field
  options directly.
- **Issues**: one unit of work. `gh issue develop --checkout` both creates
  the branch and links it to the issue.
- **OAuth scopes**: a token only has the permissions it was granted.
  `gh auth refresh -s project` adds the `project` scope without a full
  re-login — least privilege in practice, not just in theory.

## Coming up

PRs and required checks (Phase 00 closeout), branch protection rulesets
(Phase 00 closeout), releases (Phase 03), GitHub Apps and webhooks
(Phase 11).
