# Pull request checklist

Use this checklist before preparing, opening, reviewing, or merging a GSNpeeps PR.

## Contents

[Authorization and repository state](#authorization-and-repository-state) ·
[Scope and history](#scope-and-history) · [Universal checks](#universal-checks) ·
[Backend checks](#backend-checks) · [Frontend checks](#frontend-checks) ·
[Contract and migration checks](#contract-and-migration-checks) ·
[Security and privacy](#security-and-privacy) · [Verification evidence](#verification-evidence) ·
[UI evidence](#ui-evidence) · [PR description template](#pr-description-template) ·
[Review checklist](#review-checklist) · [Feedback workflow](#feedback-workflow) ·
[Merge and post-merge](#merge-and-post-merge) · [Hotfix checklist](#hotfix-checklist)

## Authorization and repository state

- [ ] Opening/updating the PR is authorized.
- [ ] Remote, repository, base branch, and head branch were verified.
- [ ] The base branch matches repository policy.
- [ ] No direct push to protected `main`.
- [ ] Existing PR/check/review state was inspected before mutation.

Preparing a PR title/body locally does not itself authorize publishing it.

## Scope and history

- [ ] Branch name follows `references/branch-strategy.md`.
- [ ] PR title follows `references/commit-format.md`.
- [ ] Diff matches the active task and acceptance criteria.
- [ ] No unrelated formatting, dependency upgrade, refactor, or user change.
- [ ] Generated files are included only when repository policy requires them.
- [ ] Commits contain no `WIP`, placeholder, or misleading subjects.
- [ ] Target strategy (default squash) is known.
- [ ] Dependencies on other branches/PRs are documented.

Inspect:

```bash
git status --short --branch
git log --oneline --decorate <base>..HEAD
git diff --stat <base>...HEAD
git diff <base>...HEAD
git diff --check <base>...HEAD
```

Use the verified base branch instead of assuming `<base>` is `main`.

## Universal checks

- [ ] Code follows `CLAUDE.md` and the relevant backend/frontend skill.
- [ ] Behavior matches PRD, User Story, OpenAPI, schema, and workflow specifications.
- [ ] Format and lint pass for every affected area.
- [ ] Unit/integration/E2E checks pass as applicable.
- [ ] Build succeeds for affected deployable artifacts.
- [ ] Errors, cancellation, timeouts, and cleanup are handled.
- [ ] No new unexplained `TODO`, `FIXME`, disabled check, or commented-out code.
- [ ] No debug print/log/route or temporary artifact remains.
- [ ] Known limitations and unrun checks are explicit.

Do not invent package-manager or linter commands. Use the commands approved in repository
configuration.

## Backend checks

- [ ] Dependency direction follows handler -> service -> repository/integration.
- [ ] Authentication, RBAC, ownership, subordinate relation, and approval stage are enforced.
- [ ] Negative authorization tests prove no side effects.
- [ ] Request/response/errors match OpenAPI.
- [ ] Parameterized queries and explicit columns are used.
- [ ] Transaction boundary covers business state, audit, and durable events as required.
- [ ] Redis keys have TTL/invalidation/failure behavior.
- [ ] Nextcloud operations validate files and handle compensation/orphans.
- [ ] Worker jobs are locked/idempotent/bounded and safe to retry.
- [ ] Context cancellation, timeouts, structured logs, and redaction are preserved.
- [ ] Relevant Go format, vet, lint, unit, integration, race/concurrency, and build checks pass.

Run only repository-defined concrete commands and report their exact output status.

## Frontend checks

- [ ] Routes, DTOs, errors, and pagination mirror OpenAPI.
- [ ] Protected data is not fetched/rendered before auth/capability resolution.
- [ ] Karyawan, Atasan, HR, and Top Management variants are tested.
- [ ] Top Management remains read-only except the documented HR-request decision path.
- [ ] Loading, empty, filtered-empty, validation, forbidden, not-found, conflict, rate-limit,
  offline, and server-error states exist where relevant.
- [ ] Form double-submit, field-error mapping, and unsaved/recoverable input are handled.
- [ ] Camera/geolocation denial and file-upload fallback are covered where relevant.
- [ ] Keyboard, focus, labels, non-color status, mobile layout, and 200% zoom are checked.
- [ ] Protected cache/state/media/object URLs are cleared on logout/unmount.
- [ ] Format, lint, tests, static production build, and relevant E2E pass.

Do not require example-project libraries that GSNpeeps has not approved.

## Contract and migration checks

- [ ] OpenAPI changes precede or accompany implementation changes.
- [ ] Frontend schemas/mocks and backend DTOs use the same contract.
- [ ] No undocumented endpoint or field is introduced.
- [ ] Database changes use a new reviewable migration.
- [ ] Forward migration, rollback-one when supported, and re-apply behavior were tested.
- [ ] Constraints/indexes/foreign keys/soft-delete match Database Schema.
- [ ] Data backfill and deployment ordering are safe.
- [ ] Breaking change is approved, marked, and has a rollout/migration path.
- [ ] Seeds and four-role permissions remain deterministic.

## Security and privacy

- [ ] No `.env`, secret, token, cookie, private key, database dump, or credential.
- [ ] No real employee PII, salary, KTP, BPJS, NPWP, address, phone, photo, or document.
- [ ] Screenshots and fixtures use synthetic data.
- [ ] Logs/responses do not expose password hash, token, storage credentials, SQL, or stack.
- [ ] Upload checks include size, MIME, extension/signature, and authorized access.
- [ ] Audit behavior covers required write/security operations.
- [ ] Dependency/config changes received appropriate security review.

If a secret was committed, deleting or rewriting history is not sufficient: revoke/rotate
the secret immediately and follow the incident process.

## Verification evidence

Report each command as:

```text
<command> -> passed | failed | not run
reason/context
```

Include applicable:

- Format/lint/vet.
- Unit, integration, contract, concurrency, accessibility, and E2E.
- Backend/frontend/worker production build.
- OpenAPI validation.
- Migration up/down-one/re-up.
- Docker Compose configuration.
- Manual smoke tests.

Never mark an unavailable/skipped check as passed. Attach concise failure context when the
PR intentionally remains draft.

## UI evidence

For UI changes, include synthetic screenshots or recordings for the meaningful states:

- Desktop and narrow/mobile.
- Default and relevant loading/empty/error state.
- Role/read-only/action variants.
- Dialog/form validation when visually material.

Do not require light/dark screenshots unless both themes are approved. Check the image for
names, emails, tokens, browser tabs, paths, and notifications before attaching.

## PR description template

```markdown
## Summary

Explain the outcome and why it is needed in 2-4 sentences.

## Scope

- Included:
- Not included:

## Changes

- Backend:
- Frontend:
- Contract/database/infrastructure:

## Authorization and behavior

- Roles/scopes affected:
- Negative authorization behavior:

## Migration and configuration

- Migration:
- New/changed environment variables:
- Deployment order/rollback:

## Verification

| Command/check | Result |
|---|---|
| `<exact command>` | Passed / Failed / Not run |

## Screenshots

Use synthetic data. Add only when UI changed.

## Risks and rollback

- Risk:
- Rollback/compensation:

## Related

Refs #<real-issue>
```

Remove irrelevant empty sections only when they add no review value. Do not invent issue
numbers, test results, reviewers, or deployment evidence.

## Review checklist

Review line-by-line for:

- Scope and acceptance criteria.
- Contract/schema/workflow correctness.
- Dependency direction and ownership.
- Role, row scope, approval stage, and negative authorization.
- Transaction, concurrency, idempotency, and error mapping.
- Sensitive data and log redaction.
- Frontend complete-state and accessibility behavior.
- Tests that fail for the intended bug/regression.
- Migration and rollback safety.
- Operational impact and observability.

Self-review is required even when another reviewer will review.

## Feedback workflow

- Address feedback with focused changes.
- Re-run affected checks.
- Resolve conversations only after the concern is actually addressed.
- Prefer a follow-up commit on shared branches.
- Amend/rebase and `--force-with-lease` only on an exclusively owned branch and with
  authorization.
- Never force-push a shared branch merely to make history prettier.

Do not dismiss a failing check as unrelated without inspecting and documenting evidence.

## Merge and post-merge

Before merge:

- [ ] Required checks pass.
- [ ] Required review/approval exists.
- [ ] PR title is the desired squash commit subject.
- [ ] Migration/deployment/rollback notes are current.
- [ ] Merge is explicitly authorized.

After confirmed merge:

- [ ] Confirm resulting commit/check status.
- [ ] Close linked issues only when appropriate.
- [ ] Update local `main` when authorized and safe.
- [ ] Delete branches only when authorized and verified merged.
- [ ] Create/push a release tag only when separately authorized.

## Hotfix checklist

- [ ] Reproduce the production issue safely.
- [ ] Add a regression test when feasible.
- [ ] Keep the patch minimal.
- [ ] Run the full affected-area suite.
- [ ] Review security/data/migration impact.
- [ ] Document rollback.
- [ ] Obtain expedited but real review.
- [ ] Tag/deploy only with explicit authorization.
- [ ] Monitor using the approved operational process.
