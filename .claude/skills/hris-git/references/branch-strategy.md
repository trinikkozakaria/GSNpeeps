# Branch strategy

Use this reference before creating, renaming, synchronizing, rebasing, merging, or deleting
a branch.

## Contents

[Model](#model) · [Branch names](#branch-names) · [Scope rules](#scope-rules) ·
[Starting a branch](#starting-a-branch) · [Branch lifecycle](#branch-lifecycle) ·
[Synchronizing with main](#synchronizing-with-main) ·
[Conflict policy](#conflict-policy) · [Merge strategy](#merge-strategy) ·
[Hotfixes](#hotfixes) · [Release branches and tags](#release-branches-and-tags) ·
[Stacked work](#stacked-work) · [Cleanup](#cleanup) ·
[Protected operations](#protected-operations)

## Model

Use lightweight trunk-based development:

```text
main (protected)
  |-- feat/employee-document-upload
  |-- fix/approval-double-decision
  |-- docs/update-api-contract
  `-- chore/frontend-foundation
```

- Treat `main` as the integration/release-ready branch.
- Change `main` through a reviewed PR.
- Keep feature branches short-lived and narrowly scoped.
- Do not introduce `develop` or `staging` branches unless repository governance explicitly
  adopts them.
- Verify the actual default/protected branch from repository/remote configuration before a
  remote action; guidance does not override server policy.

## Branch names

Use:

```text
<type>/<specific-kebab-description>
```

| Prefix | Use |
|---|---|
| `feat/` | New user-visible capability |
| `fix/` | Defect correction |
| `refactor/` | Internal restructuring without intended behavior change |
| `test/` | Test-only addition or correction |
| `docs/` | Documentation/specification only |
| `chore/` | Maintenance and configuration |
| `build/` | Build/container/dependency workflow |
| `ci/` | Continuous-integration configuration |
| `hotfix/` | Urgent production correction |
| `release/` | Explicit approved release preparation |

Rules:

- Use lowercase kebab-case.
- Use two to five meaningful words after the prefix.
- Describe the product/task outcome, not the implementer.
- Avoid issue-only, personal, generic, or environment-specific names.
- Follow an active prompt's authorized branch name when it is valid and not superseded.

Examples:

```text
feat/employee-document-upload
feat/attendance-live-feed
feat/leave-approval-delegation
fix/approval-double-decision
fix/notification-unread-count
refactor/employee-repository-boundary
test/attendance-radius-errors
docs/align-api-contract
chore/frontend-foundation
build/nginx-static-delivery
ci/add-contract-validation
hotfix/login-lockout-regression
release/v0.1.0
```

Avoid:

```text
feature_employee
feat/update
my-branch
magang-fix
fix/bug
```

## Scope rules

Keep one coherent vertical slice or maintenance concern per branch.

Include together when required for one outcome:

- Migration and repository/service/handler changes.
- OpenAPI and corresponding backend/frontend/mock changes.
- Feature implementation and its tests.
- Configuration and `.env.example` changes required by the feature.

Separate:

- Unrelated bug fixes.
- Broad formatting.
- Dependency upgrades unrelated to the task.
- Opportunistic refactors that make review materially larger.
- Generated artifacts not required by repository policy.

## Starting a branch

Before mutation, inspect:

```bash
git status --short --branch
git remote -v
git branch -vv
git log --oneline --decorate -n 10
```

Confirm:

- The directory is a Git repository.
- The base branch and remote are correct.
- The worktree contains no unrelated changes that branch switching would endanger.
- Branch creation is authorized.

Typical authorized sequence:

```bash
git switch main
git pull --ff-only origin main
git switch -c feat/employee-document-upload
```

Prefer `switch` for branch operations when supported. Do not pull or switch if it could
overwrite/entangle uncommitted user work; preserve it and report the condition.

## Branch lifecycle

```text
base from current main
  -> implement focused task
  -> create atomic commits
  -> synchronize safely
  -> rerun affected checks
  -> push when authorized
  -> open PR when authorized
  -> review/checks
  -> squash merge by approved actor
  -> cleanup when authorized
```

Do not create, push, open, merge, tag, or delete merely because a planning prompt lists
those future steps. Each action must be within current authorization.

## Synchronizing with main

Fetch first:

```bash
git fetch origin
```

### Personal unpublished/exclusively owned branch

Rebase may keep history linear:

```bash
git rebase origin/main
```

After an already-pushed authorized rebase:

```bash
git push --force-with-lease
```

Only do this when rewrite and push are authorized and the branch is not shared.

### Shared branch

Prefer merge to preserve collaborators' commits:

```bash
git merge origin/main
```

Never rebase a shared branch or a branch used as another person's PR base without
coordination.

Do not use `git pull` with an unknown configured merge/rebase behavior. Prefer explicit
`fetch` plus selected integration, or `pull --ff-only` when only fast-forward is acceptable.

## Conflict policy

For each conflict:

1. List conflict files with `git status`.
2. Read both sides and the surrounding contract/tests.
3. Preserve the intended behavior from both changes when compatible.
4. Run focused tests after resolving.
5. Stage only resolved files.
6. Continue the rebase/merge.

Do not automatically choose `ours` or `theirs` for code, migrations, API contracts, lockfiles,
or user-authored documents. If intent is ambiguous or overlaps unrelated user work, abort
or pause and request direction.

Safe aborts:

```bash
git rebase --abort
git merge --abort
git cherry-pick --abort
```

## Merge strategy

Use Squash and Merge as the target default for ordinary feature branches:

- The PR becomes one coherent commit on `main`.
- PR title must follow Conventional Commits.
- The PR body preserves verification and decision context.

Use rebase merge only when repository policy approves it and every commit is atomic and
review-quality. Use a merge commit only when preserving branch topology is intentional.

Do not merge a PR without explicit authorization and required checks/review.

## Hotfixes

Create an urgent branch from the latest production `main`:

```text
hotfix/<specific-regression>
```

Require:

- Reproduction evidence.
- Regression test when feasible.
- Minimal fix and no unrelated refactor.
- Full affected-area checks.
- Explicit rollback and monitoring plan.
- Patch tag only after the hotfix is merged and release/tagging is authorized.

Urgency does not permit force-pushing protected branches or skipping security review.

## Release branches and tags

Use `release/vX.Y.Z` only if an approved release process needs coordinated version,
documentation, migration, and deployment preparation. Otherwise release from `main`.

Use Semantic Versioning once the repository adopts a version:

```text
v0.1.0
v0.1.1
v0.2.0
v1.0.0
```

Prefer annotated tags:

```bash
git tag -a v0.1.0 -m "release: v0.1.0"
```

Verify the target commit before tagging. Tag creation, tag push, retagging, and tag deletion
are separate remote-impacting actions and require explicit authorization. Never silently
move an existing release tag.

## Stacked work

Avoid branching from an unmerged feature branch. If a dependency makes stacked work
necessary:

- Document the parent branch/PR.
- Keep dependent changes separable.
- Target the dependent PR appropriately.
- Retarget/rebase only with coordination after the parent merges.

Do not allow accidental parent commits to enter an unrelated PR.

## Cleanup

After confirmed merge, inspect whether the local branch contains unique work:

```bash
git branch --merged main
git log main..<branch>
```

Delete local or remote branches only when authorized. Prefer safe local deletion (`-d`) and
do not escalate to `-D` just because Git refuses; investigate why the branch appears
unmerged.

Pruning stale remote-tracking references is not the same as deleting remote branches:

```bash
git fetch --prune
```

## Protected operations

Never perform these without explicit, target-specific authorization:

- Direct push to `main`.
- Any force push to protected/shared branches.
- History rewrite of a shared branch.
- Merge.
- Local/remote branch deletion.
- Tag creation, movement, push, or deletion.
- Release or deployment.
