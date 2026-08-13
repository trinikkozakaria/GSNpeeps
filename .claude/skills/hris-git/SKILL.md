---
name: hris-git
description: Prepare and review safe, focused Git changes for GSNpeeps. Use when creating branches, inspecting status or diffs, composing Conventional Commits, preparing pull requests, resolving ordinary conflicts, tagging approved releases, or reporting verification evidence for backend, frontend, migration, infrastructure, and documentation tasks.
---

# GSNpeeps Git Workflow

Use a lightweight trunk-based workflow that keeps changes focused, reviewable, reversible,
and supported by verification evidence.

## Quick rules

- Target branch: `main`, protected and changed through reviewed PRs.
- Working branch: short-lived `<type>/<specific-kebab-description>`.
- Commit/PR title: `<type>(<scope>): <imperative English summary>`.
- Default merge strategy: squash merge unless repository policy says otherwise.
- Stage explicit paths and inspect the staged diff.
- Preserve unrelated user work.
- Treat commit, push, PR, merge, tag, release, history rewrite, and deletion as distinct
  actions requiring the authorization applicable to each.

## Workflow

1. Read `CLAUDE.md`, the active prompt, and its branch/commit/remote authorization.
2. Inspect repository root, current branch, tracking branch, status, remotes, and recent log.
3. Read `references/branch-strategy.md` before creating, syncing, rebasing, merging, or
   deleting a branch.
4. Identify the exact task files and preserve unrelated changes.
5. Run the quality gates required by the affected backend, frontend, contract, migration,
   infrastructure, or documentation area.
6. Scan the intended patch for secrets, PII, generated output, debug code, and temporary
   artifacts.
7. Stage explicit files or hunks and inspect `git diff --staged`.
8. Read `references/commit-format.md`, then create the smallest coherent commit when
   authorized.
9. Read `references/pr-checklist.md` before preparing or opening a PR.
10. Use `references/workflow-commands.md` for safe command sequences and recovery.

## Reference routing

- Read `references/branch-strategy.md` for branch types, naming, lifecycle, synchronization,
  merge strategy, hotfixes, releases, and cleanup.
- Read `references/commit-format.md` for anatomy, types, GSNpeeps scopes, body/footer,
  breaking changes, atomicity, and examples.
- Read `references/pr-checklist.md` for pre-PR checks, review focus, description template,
  migrations, contracts, security, UI evidence, hotfixes, and post-merge actions.
- Read `references/workflow-commands.md` for inspection, staging, commit, sync, conflict,
  stash, revert, recovery, tag, and remote-operation commands.

## Safety

- Do not infer authorization for commit, push, PR, merge, tag, release, deployment, or
  production mutation from a request to edit or review code.
- Never force-push `main` or another protected/shared branch.
- Use `--force-with-lease` on a personal branch only when history rewrite is authorized and
  the remote state was freshly inspected.
- Never use hard reset, destructive clean, broad restore/checkout, or branch/tag deletion
  merely to make the worktree clean.
- Never commit `.env`, credentials, tokens, dumps, production employee data, Nextcloud
  secrets, real documents/photos, or sensitive screenshots.
- Do not claim a check passed when it was skipped, unavailable, or failed.
- Stop on conflicts that overlap unrelated user changes; report the exact files.

## Completion report

Report current branch, commit hash if created, files included, quality commands and results,
remote actions actually performed, and anything intentionally left uncommitted/unpushed.
