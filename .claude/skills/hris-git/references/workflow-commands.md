# Workflow commands

Use this command reference after reading the action's safety and authorization requirements.
Replace placeholders only after inspecting the repository.

## Contents

[Inspection first](#inspection-first) · [Compare changes](#compare-changes) ·
[Start a task](#start-a-task) · [Stage safely](#stage-safely) ·
[Commit](#commit) · [Push](#push) · [Synchronize](#synchronize) ·
[Stash](#stash) · [Resolve conflicts](#resolve-conflicts) ·
[Undo and recovery](#undo-and-recovery) · [History inspection](#history-inspection) ·
[Cherry-pick and revert](#cherry-pick-and-revert) · [Tags](#tags) ·
[Pull requests](#pull-requests) · [End-to-end sequences](#end-to-end-sequences) ·
[Never run casually](#never-run-casually)

## Inspection first

Start with read-only commands:

```bash
git rev-parse --show-toplevel
git status --short --branch
git remote -v
git branch -vv
git log --oneline --decorate -n 20
```

Interpret short status:

```text
?? untracked
 M modified in working tree
M  staged modification
MM staged and then modified again
A  staged addition
D  staged deletion
UU unresolved conflict
```

Check whether the repository has guidance/configuration:

```bash
git config --get remote.origin.url
git config --get branch.<branch>.remote
git config --get branch.<branch>.merge
```

Do not change global Git configuration as a routine task action.

## Compare changes

Working tree:

```bash
git diff
git diff --stat
git diff --check
```

Staged:

```bash
git diff --staged
git diff --staged --stat
git diff --staged --check
git diff --staged --name-status
```

Branch versus base:

```bash
git diff --stat main...HEAD
git diff main...HEAD
git log --oneline --decorate main..HEAD
```

Use `...` for the PR-style diff from merge base. Use `-- <path>` to narrow inspection:

```bash
git diff main...HEAD -- backend/
git diff --staged -- frontend/
git log -p -- docs/openapi.yaml
```

Quote paths containing spaces.

## Start a task

Only after checking a clean/safe switching state and receiving branch authorization:

```bash
git switch main
git pull --ff-only origin main
git switch -c feat/employee-document-upload
```

If `main` is not the verified base, use the actual approved base. Do not branch from a stale
local base when remote synchronization is required and authorized.

For an existing remote branch:

```bash
git fetch origin
git switch --track origin/<branch>
```

Inspect before creating a similarly named branch.

## Stage safely

Prefer explicit paths:

```bash
git add backend/internal/service/employee/
git add docs/openapi.yaml
git add frontend/src/modules/employees/
```

For mixed files, stage interactively:

```bash
git add -p -- <path>
```

Unstage without changing the working file:

```bash
git restore --staged -- <path>
```

Review immediately:

```bash
git diff --staged --name-status
git diff --staged
git diff --staged --check
```

Avoid `git add .` or `git add -A` when unrelated changes exist. Never stage a file merely
to make `git status` shorter.

Before commit, search staged content for:

- Secrets and environment values.
- Real employee PII/documents/photos.
- Debug output and temporary files.
- Conflict markers.
- Generated/build artifacts.
- Unrelated formatting.

Use repository-approved secret scanning when available.

## Commit

Create a one-line commit:

```bash
git commit -m "feat(employee): add document upload flow"
```

For a body, use the configured editor:

```bash
git commit
```

Or pass separate paragraphs without shell interpolation:

```bash
git commit -m "fix(approval): prevent duplicate final decisions" -m "Use a conditional update so concurrent decisions cannot both succeed."
```

Verify:

```bash
git show --stat --oneline HEAD
git show --format=fuller --no-patch HEAD
git status --short --branch
```

Amend only when the last commit is exclusively owned, amendment is authorized, and no
collaborator depends on the old hash:

```bash
git commit --amend
git commit --amend --no-edit
```

Inspect `git show HEAD` after amendment.

## Push

Verify remote and upstream:

```bash
git remote -v
git branch -vv
```

First authorized push:

```bash
git push -u origin <branch>
```

Later authorized push:

```bash
git push
```

After an authorized rebase/amend of an exclusively owned remote branch:

```bash
git fetch origin
git push --force-with-lease
```

Never substitute `--force`. Never force-push a protected or shared branch.

## Synchronize

Inspect remote changes:

```bash
git fetch origin
git log --oneline --left-right --graph HEAD...origin/main
```

Personal/exclusively owned branch:

```bash
git rebase origin/main
```

Shared branch:

```bash
git merge origin/main
```

After either operation:

1. Inspect status and log.
2. Run affected quality gates again.
3. Inspect the PR-style diff.
4. Push only when authorized.

Use:

```bash
git pull --ff-only origin main
```

only when a fast-forward is the desired verified behavior. Avoid bare `git pull` when local
merge/rebase configuration is unknown.

## Stash

Stash is a mutation and can hide work. Inspect first and use it only when necessary and
authorized:

```bash
git stash push -u -m "wip: employee document upload before branch switch"
git stash list
git stash show --stat stash@{0}
git stash show -p stash@{0}
```

Prefer `apply` before `drop`:

```bash
git stash apply stash@{0}
```

After verifying the restored changes:

```bash
git stash drop stash@{0}
```

Do not use `stash pop` when conflicts or recovery risk are significant. Never clear all
stashes casually.

In PowerShell, quote a stash reference containing braces:

```powershell
git stash show -p 'stash@{0}'
```

## Resolve conflicts

List conflicts:

```bash
git status --short
git diff --name-only --diff-filter=U
```

Inspect conflict stages where useful:

```bash
git show :1:<path>
git show :2:<path>
git show :3:<path>
```

Resolve the file intentionally, remove markers, run focused checks, then:

```bash
git add -- <resolved-path>
git diff --check
git rebase --continue
```

For merge, continue with the repository's expected merge commit flow.

Abort safely when the strategy or intent is wrong:

```bash
git rebase --abort
git merge --abort
git cherry-pick --abort
```

Do not use `ours`/`theirs` across whole files without understanding that their meaning can
differ between merge and rebase.

## Undo and recovery

### Unstage while preserving work

```bash
git restore --staged -- <path>
```

### Discard a working-tree change

`git restore -- <path>` destroys uncommitted content. Use it only for an exact path after the
user explicitly requests that discard and after inspecting the diff.

### Undo a published commit

Prefer a new inverse commit:

```bash
git revert <commit>
```

### Abort an in-progress operation

```bash
git rebase --abort
git merge --abort
git cherry-pick --abort
```

### Find lost references

Read-only recovery inspection:

```bash
git reflog --date=local
git fsck --no-reflogs --unreachable
```

Create a recovery branch at a verified commit rather than resetting destructively:

```bash
git branch recovery/<description> <commit>
```

Do not run hard reset as a standard recovery step.

## History inspection

```bash
git log --oneline --graph --decorate --all
git log --first-parent --oneline main
git log --since="2 weeks ago" --oneline
git log --author="<name>" --oneline
git log -- <path>
git log -p -- <path>
git show <commit>
git blame -L <start>,<end> -- <path>
```

Use history to understand intent, not to expose or repeat private author information
unnecessarily.

## Cherry-pick and revert

Inspect the commit first:

```bash
git show --stat <commit>
git show <commit>
```

Authorized cherry-pick:

```bash
git cherry-pick <commit>
```

Use cherry-pick for a deliberate transfer, such as an approved hotfix. Do not duplicate
commits across branches without understanding later merge behavior.

Revert a published change:

```bash
git revert <commit>
```

For either operation, resolve conflicts intentionally and rerun checks.

## Tags

Inspect tags:

```bash
git tag --list
git show <tag>
git ls-remote --tags origin
```

Authorized annotated tag:

```bash
git tag -a v0.1.0 -m "release: v0.1.0" <verified-commit>
```

Authorized tag push:

```bash
git push origin v0.1.0
```

Do not move, recreate, or delete a local/remote tag without explicit target-specific
authorization.

## Pull requests

If the repository uses GitHub and the approved CLI is authenticated, inspect first:

```bash
gh pr status
gh pr list --head <branch>
gh pr view <number>
gh pr checks <number>
```

Open a PR only when authorized:

```bash
gh pr create --base main --head <branch> --title "feat(employee): add document upload flow" --body-file <prepared-file>
```

Do not invent reviewers, labels, issue numbers, or check results. Do not merge with the CLI
unless merge authorization is explicit.

## End-to-end sequences

### Prepare a commit from an existing worktree

```text
1. Inspect repository, status, diff, and recent history.
2. Identify exact task files and unrelated changes.
3. Run affected quality gates.
4. Stage explicit paths/hunks.
5. Inspect staged name/status, full diff, and whitespace.
6. Scan sensitive/generated/debug content.
7. Compose Conventional Commit message.
8. Commit only when authorized.
9. Verify commit and remaining worktree.
```

### Prepare a PR

```text
1. Verify base/head/remote and authorization.
2. Fetch and synchronize with the approved strategy.
3. Rerun checks after synchronization.
4. Inspect base...HEAD log/diff/check.
5. Complete PR checklist and description.
6. Push when authorized.
7. Open PR when authorized.
8. Report URL/check state without claiming pending checks passed.
```

## Never run casually

- `git reset --hard`
- `git clean -fd` or `git clean -fdx`
- Broad `git restore` or checkout used to discard changes
- `git push --force`
- Force push to protected/shared branches
- Interactive rebase of shared history
- Local/remote branch deletion
- Local/remote tag deletion or retagging
- Direct push to `main`
- Merge/release/deploy without explicit authorization

If a conflict overlaps user work, stop and identify exact files instead of choosing a side
silently.
