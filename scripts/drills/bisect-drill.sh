#!/usr/bin/env bash
# Git bisect + revert drill.
#
# Creates a throwaway branch (drill/bisect) with 7 commits. One of them
# quietly breaks the queue's fencing check (ADR-008); it is disguised as a
# harmless refactor. You then use `git bisect` to find it, and `git revert`
# to undo it, exactly as you would a real regression. The branch is never
# pushed and is deleted at the end.
#
# Usage (from a clean working tree, on any branch containing internal/queue):
#   scripts/drills/bisect-drill.sh
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

if [ -n "$(git status --porcelain)" ]; then
  echo "Working tree is not clean. Commit or stash your changes first." >&2
  exit 1
fi

ORIGIN_BRANCH=$(git branch --show-current)
GOOD=$(git rev-parse --short HEAD)
DRILL=drill/bisect

if git show-ref --verify --quiet "refs/heads/$DRILL"; then
  echo "Branch $DRILL already exists from an earlier drill. Delete it first:" >&2
  echo "  git switch $ORIGIN_BRANCH && git branch -D $DRILL" >&2
  exit 1
fi

git switch -q -c "$DRILL"

harmless() {
  echo "note $1" >> drill-notes.txt
  git add drill-notes.txt
  git commit -q -m "chore(drill): harmless change $1"
}

introduce_bug() {
  python3 - <<'PY'
p = "internal/queue/queue.go"
s = open(p).read()
old = "WHERE id = $1 AND locked_by = $2 AND attempt = $3 AND status = 'running'`,\n\t\t\tjob.ID, workerID, job.Attempt)\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\tif tag.RowsAffected() == 0 {\n\t\t\treturn ErrLeaseLost\n\t\t}\n\t\t_, err = tx.Exec(ctx,\n\t\t\t`UPDATE job_attempts SET status = 'succeeded'"
new = "WHERE id = $1 AND status = 'running'`,\n\t\t\tjob.ID)\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\tif tag.RowsAffected() == 0 {\n\t\t\treturn ErrLeaseLost\n\t\t}\n\t\t_, err = tx.Exec(ctx,\n\t\t\t`UPDATE job_attempts SET status = 'succeeded'"
assert old in s, "drill could not find Complete's SQL; queue.go has changed since this script was written"
open(p, "w").write(s.replace(old, new, 1))
PY
  git commit -q -am "refactor(queue): simplify Complete's ownership check"
}

harmless 1
harmless 2
harmless 3
introduce_bug
harmless 4
harmless 5
harmless 6

cat <<EOF

Drill branch '$DRILL' is ready: 7 commits on top of $GOOD.
One of them broke Complete's fencing. You do not know which.

1. FIND IT with bisect ($GOOD is known-good, the tip is known-bad):

     git log --oneline $GOOD..HEAD
     git bisect start
     git bisect bad HEAD
     git bisect good $GOOD
     git bisect run go test ./internal/queue -run TestComplete_IsFenced
     git bisect log
     git bisect reset

   'bisect run' treats exit code 0 as good and non-zero as bad, so the
   failing test is the oracle. With 7 commits it needs about 3 test runs
   instead of 7: bisect is a binary search over history.

2. UNDO IT with revert (the culprit is now known, and history is shared
   by others, so we add a new commit instead of rewriting history):

     git revert <culprit-hash>
     go test ./internal/queue -run TestComplete_IsFenced

3. CLEAN UP:

     git switch $ORIGIN_BRANCH
     git branch -D $DRILL
EOF
