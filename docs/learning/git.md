# Git — what I've learned, in the order I learned it

Updated as each phase teaches new commands. See
[docs/phases/README.md](../phases/README.md) for the full curriculum map.

## Phase 00

- **`git init`** creates `.git/`: a `HEAD` file (a pointer, initially to
  a branch name that doesn't exist yet), `objects/` (empty until the first
  commit), `refs/heads/` (empty until the first commit).
- **The object model**: a **blob** is one file's content, addressed by the
  SHA-1/SHA-256 hash of that content. A **tree** is a directory snapshot —
  a list of (mode, hash, filename) entries, some pointing to blobs, some to
  other trees. A **commit** points to one tree plus a parent commit (none,
  for the first commit), an author, and a message. A **branch** is nothing
  more than a small file holding one commit hash — `git branch -vv` and
  `cat .git/refs/heads/main` both show this directly.
- **`git add`** copies file content into a blob object and records it in
  the index (staging area). It does not create a commit.
- **`git status`** shows three states: untracked, staged
  ("Changes to be committed"), and committed.
- **`git diff --staged`** shows exactly what the next commit will contain —
  check this before every commit, not after.
- **`git commit`** builds the tree from the index and writes the commit
  object; it then updates the current branch's ref file to point at the
  new commit. `git log`, `git show`, and `git cat-file -p <hash>` all
  inspect these objects directly.
- **`git remote`** and **`git push -u`**: a remote is just a named URL.
  `-u` (`--set-upstream`) records which remote branch the local branch
  tracks, so plain `git push`/`git pull` know where to go afterward
  (`git branch -vv` shows the tracking branch in brackets).
- **`gh issue develop --checkout`**: creates and checks out a branch while
  also linking it to the GitHub issue, so the issue's page shows the
  branch and, later, the PR.

## Why `main` got exactly one direct commit

A pull request needs an existing branch to target. Before any commit
exists, there's nothing to branch from. The bootstrap commit is the one
deliberate exception to "never commit directly to `main`" — every commit
after it goes through a branch and a PR.
