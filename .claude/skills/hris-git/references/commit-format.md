# Commit format

Use Conventional Commits 1.0.0 for commit messages and squash-merge PR titles.

## Contents

[Anatomy](#anatomy) · [Types](#types) · [GSNpeeps scopes](#gsnpeeps-scopes) ·
[Subject rules](#subject-rules) · [Body](#body) · [Footers](#footers) ·
[Breaking changes](#breaking-changes) · [Atomic commits](#atomic-commits) ·
[Examples](#examples) · [Special cases](#special-cases) ·
[Validation checklist](#validation-checklist)

## Anatomy

```text
<type>(<scope>): <imperative summary>

<optional body explaining why and consequences>

<optional footers>
```

With a breaking marker:

```text
<type>(<scope>)!: <imperative summary>
```

Scope is strongly preferred for project changes. Omit it only when no honest single scope
exists, such as some repository-wide CI changes.

## Types

| Type | Use |
|---|---|
| `feat` | New user-visible or API capability |
| `fix` | Defect correction |
| `refactor` | Internal change without intended external behavior change |
| `test` | Test/fixture-only change |
| `docs` | Documentation, specification, comments |
| `perf` | Measured or justified performance improvement |
| `build` | Dependencies, compilation, container/build system |
| `ci` | CI workflow and automation |
| `chore` | Maintenance not better represented above |
| `style` | Formatting-only change with no logic change; use sparingly |
| `revert` | Revert of a prior commit |

Choose the type from the behavior of the commit, not the file extension. An OpenAPI change
that introduces a new API capability may be `feat(api)`, while explanatory OpenAPI cleanup
may be `docs(openapi)`.

## GSNpeeps scopes

Use the narrowest stable product or platform scope.

### Product

| Scope | Area |
|---|---|
| `auth` | Login, logout, JWT session, lockout |
| `employee` | Employee database and details |
| `organization` | Departments, positions, supervisor relation |
| `profile` | Personal profile |
| `dashboard` | HR/Top Management metrics |
| `attendance` | WFO/WFH/WFA check-in/out and live feed |
| `report` | Attendance reports and exports |
| `leave` | Leave/absence requests and master leave types |
| `overtime` | Overtime requests and recap |
| `approval` | Decision, delegation, escalation, timeline |
| `notification` | In-app notifications and unread count |
| `access` | Role and permission administration |
| `audit` | Audit-log behavior and views |

### Technical

| Scope | Area |
|---|---|
| `api` | Cross-module API contract/transport |
| `openapi` | OpenAPI-only documentation/generation |
| `db` or `migration` | Schema, constraint, index, seeds |
| `redis` | Session, rate limit, cache, distributed locks |
| `storage` | Nextcloud/WebDAV and file lifecycle |
| `worker` | Scheduled jobs |
| `middleware` | Backend middleware |
| `ui` | Shared UI primitives |
| `form` | Shared form adapters/patterns |
| `layout` | Frontend shell/navigation layout |
| `frontend` | Cross-feature frontend foundation |
| `backend` | Cross-feature backend foundation |
| `docker` | Docker Compose/images |
| `nginx` | Nginx routing/static delivery |
| `config` | Application/environment configuration |
| `deps` | Dependency-only updates |
| `tooling` | Local developer tooling |
| `ci` | CI configuration |
| `claude` | AI project guidance |
| `release` | Release preparation/versioning |

Do not invent separate singular/plural scopes for backend/frontend variants. Prefer one
product scope, such as `employee`, when a vertical slice changes both sides.

## Subject rules

- Keep the entire subject at or below 72 characters when practical.
- Write the summary in English.
- Use imperative present tense: `add`, `fix`, `prevent`, `remove`, `document`.
- Start lowercase except proper technical names such as `JWT`, `Redis`, or `OpenAPI`.
- Do not end with a period.
- Describe the outcome precisely.
- Avoid filenames, implementation trivia, issue number alone, and vague words.

Good:

```text
feat(employee): add document upload flow
fix(approval): prevent duplicate final decisions
test(attendance): cover WFO radius boundary
docs(openapi): document employee export errors
```

Avoid:

```text
update employee
fix bug
WIP
feat: changes
feat(employee): added form.
```

## Body

Add a body when review/recovery needs context:

- Why the change is necessary.
- Important behavior and tradeoffs.
- Migration/data/config impact.
- Security or authorization impact.
- Compatibility or rollout constraints.
- Tests that demonstrate a non-obvious invariant.

Separate it from the subject with one blank line. Wrap for terminal readability where
practical. Bahasa Indonesia is allowed in the body for business context; keep identifiers
and API terms exact.

Do not paste a full file list or claim checks passed. Verification belongs in the PR/report
unless a short note is materially useful.

Example:

```text
fix(approval): prevent duplicate final decisions

Gunakan conditional update pada tahap approval aktif agar dua approver
tidak dapat menyelesaikan request yang sama. Request yang kalah menerima
ALREADY_DECIDED dan memuat ulang status terbaru.
```

## Footers

Use:

```text
Refs #123
Closes #123
Co-authored-by: Name <email@example.com>
BREAKING CHANGE: <impact and migration path>
```

- Use real issue identifiers only.
- Do not invent co-author attribution.
- Ensure the requested author identity/email is appropriate before recording it.
- Use `Closes` only when merge truly completes the issue.

## Breaking changes

Mark an externally incompatible contract, schema, configuration, CLI, or deployment change:

```text
feat(api)!: rename attendance pagination fields

BREAKING CHANGE: clients must replace `per_page` with `limit` and
consume `total_data` and `total_page`.
```

Before committing a breaking change, verify that it is explicitly approved and that
OpenAPI, migrations, frontend, mocks, deployment, and rollout notes are synchronized.

Do not use `!` for an internal refactor with no consumer impact.

## Atomic commits

A commit should:

- Represent one coherent outcome.
- Build/test at the appropriate boundary when feasible.
- Include the tests and contract/migration required by that outcome.
- Exclude unrelated formatting and user changes.
- Be reviewable and revertible without leaving intentionally broken state.

Split when changes can be reviewed/reverted independently. Keep together when separation
would create a misleading or broken contract, such as a migration plus the code that
requires it.

Use explicit staging or patch staging. Do not create arbitrary "backend" and "frontend"
commits if one inseparable vertical slice is clearer.

## Examples

```text
feat(auth): add Redis-backed session validation
feat(employee): add soft-deactivation endpoint
feat(attendance): enforce WFO office radius
feat(approval): add supervisor delegation
fix(notification): preserve dismissed event idempotency
fix(storage): remove orphan after metadata failure
test(worker): cover repeated contract reminder run
docs(claude): detail frontend architecture guidance
build(docker): add isolated worker service
ci(openapi): validate contract changes
```

Example with migration context:

```text
feat(notification): add idempotent event keys

Tambahkan unique constraint recipient_user_id dan event_key agar
retry worker tidak membuat notifikasi ganda.

Refs #42
```

## Special cases

### Dependency-only

Use `build(deps)` when the dependency changes the production/build graph; use
`chore(deps)` for routine maintenance consistent with repository convention.

```text
build(deps): add approved PostgreSQL driver
chore(deps): update development lint tooling
```

### Revert

```text
revert: feat(attendance): add live camera capture

This reverts commit abc1234.

Reason: browser fallback does not release the media stream reliably.
```

Prefer `git revert` for published commits. Preserve the generated attribution line unless
repository convention requires otherwise.

### Squash PR title

Because the default merge strategy is squash, write the PR title as the final Conventional
Commit subject. Small branch commits may be reorganized before merge, but never rewrite
shared history without coordination.

## Validation checklist

- [ ] Type reflects behavior.
- [ ] Scope is stable and GSNpeeps-specific.
- [ ] Subject is imperative English, precise, and has no trailing period.
- [ ] Commit contains one coherent outcome.
- [ ] Body explains non-obvious why/impact.
- [ ] Breaking changes are approved and marked.
- [ ] Issue/co-author footers are real.
- [ ] Staged diff contains no unrelated or sensitive data.
- [ ] Message does not claim unrun verification.
