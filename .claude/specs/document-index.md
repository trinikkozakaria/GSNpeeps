# GSNpeeps — Source Document Index and Traceability

Dokumen ini menjelaskan sumber kebenaran, prioritas konflik, tanggung jawab setiap PDF,
coverage ringkasan `.claude/specs`, dan keputusan/gap yang belum boleh ditebak.

## Contents

[Product identity](#product-identity) · [Source inventory](#source-inventory) ·
[Authority order](#authority-order) · [Responsibility by document](#responsibility-by-document) ·
[Traceability by feature](#traceability-by-feature) ·
[Repository specification map](#repository-specification-map) ·
[Conflict resolution procedure](#conflict-resolution-procedure) ·
[Known decisions](#known-decisions) · [Known gaps and ambiguities](#known-gaps-and-ambiguities) ·
[Version and change control](#version-and-change-control) ·
[Reading checklist](#reading-checklist)

## Product identity

- Product/repository name: **GSNpeeps**.
- Product type: internal web-based employee database and HR dashboard resembling an HRIS.
- Phase scope: Employee Database, HR Dashboard, Attendance, Leave/Overtime Approval,
  Notification, User Management, RBAC, and Audit.
- `janjikupadamu.id` is a candidate deployment domain from source documents, not the product
  name.
- Titles containing “HRIS Employee Database” are descriptive source titles.

Use **GSNpeeps** in new UI, documentation, logs identifying the service, and release material.

## Source inventory

All source files were present at the paths below when this index was updated.

| Priority | Document | Version | Primary authority | Source path |
|---:|---|---:|---|---|
| 1 | PRD Employee Database Dashboard HRIS | 1.2 | Product scope, goals, business rules, phase boundaries | `C:\Users\Magang\Downloads\PRD_Employee_Database_Dashboard_HRIS_V1.2.pdf` |
| 2 | User Story Employee Database Dashboard | 1.2 | User outcomes, roles, journeys, acceptance behavior | `C:\Users\Magang\Downloads\User_Story_Employee_Database_Dashboard_V1.2.pdf` |
| 3 | API Contract HRIS Employee Database | 1.1 | Paths, methods, payloads, response envelopes, errors, status codes | `C:\Users\Magang\Downloads\API_Contract_HRIS_Employee_Database_V1.1.pdf` |
| 4 | Database Schema HRIS Employee Database | 1.1 | Tables, columns, types, nullability, constraints, foreign keys, indexes | `C:\Users\Magang\Downloads\Database_Schema_HRIS_Employee_Database_V1.1.pdf` |
| 5 | Sequence Diagram HRIS Employee Database | 1.1 | Actor/service interaction order, transaction and integration sequence | `C:\Users\Magang\Downloads\Sequence_Diagram_HRIS_Employee_Database_V1.1.pdf` |
| 6 | System Design HRIS Employee Database | 1.0 | Runtime architecture, services, network, storage, deployment | `C:\Users\Magang\Downloads\System_Design_HRIS_Employee_Database_V1_0.pdf` |

Do not copy these local absolute paths into runtime configuration. If the sources are moved
into the repository, preserve filenames/versions in an approved `docs/source/` location and
update this index.

## Authority order

When two sources disagree, use:

1. Latest explicit user instruction.
2. PRD v1.2.
3. User Story v1.2.
4. API Contract v1.1.
5. Database Schema v1.1.
6. Sequence Diagram v1.1.
7. System Design v1.0.
8. `.claude/specs/` summaries.
9. Existing implementation/tests.

This priority does not permit silently breaking a lower-level contract. When a higher source
changes behavior:

- Record the discrepancy.
- Update the affected API/schema/workflow source or obtain an approved version decision.
- Update consumers and tests together.
- Preserve backward/deployment/migration implications explicitly.

## Responsibility by document

### PRD v1.2

Read before:

- Adding/removing a feature.
- Choosing phase scope.
- Defining dashboard metrics.
- Changing attendance, approval, notification, or retention rules.
- Deciding what is Coming Soon.

Do not derive SQL types or HTTP payload names from the PRD.

### User Story v1.2

Read before:

- Designing role-specific UI and journeys.
- Writing acceptance tests.
- Defining loading/error/success behavior from the user's perspective.
- Prioritizing navigation and workflows.

Do not treat a simplified story as permission to bypass API/schema constraints.

### API Contract v1.1

Read before:

- Adding a route/handler/client function.
- Defining DTO/schema/mock.
- Choosing PUT/PATCH, query parameter, upload field, response status, or error.
- Building frontend forms and server-state keys.

The repository summary lists modules and paths, but exact payload fields must be read from
the PDF/OpenAPI generated from it.

### Database Schema v1.1

Read before:

- Writing migrations, models, repositories, constraints, and indexes.
- Deciding nullable/optional behavior.
- Implementing soft delete, status checks, history, or uniqueness.

The summary identifies 26 tables and key constraints; exact columns/types remain governed by
the PDF.

### Sequence Diagram v1.1

Read before:

- Implementing login/logout, upload, check-in/out, approval, notification, and worker flow.
- Choosing transaction boundaries and compensation order.
- Writing integration/E2E sequences.

Use it to validate ordering, not to invent operations absent from the API Contract.

### System Design v1.0

Read before:

- Changing service topology, public ports, Nginx, Docker, storage, Redis, or worker deployment.
- Introducing another service or exposing infrastructure.
- Changing backup/retention or file access architecture.

Do not inherit library/framework choices that System Design does not specify.

## Traceability by feature

| Feature | Product/story | API | Data | Sequence | System |
|---|---|---|---|---|---|
| Login/logout/lockout | PRD + User Story | Auth operations/errors | users/roles | Login/logout sequence | API + Redis |
| Employee database | PRD + User Story | Karyawan/master operations | organization + employee tables | CRUD/upload/export sequence | API + PostgreSQL + Nextcloud |
| Own profile/metrics | PRD + User Story | Profil operations | employee/salary/attendance | Profile read sequence | API |
| HR dashboard/org chart | PRD + User Story | Dashboard operation | employee/attendance/leave/salary | Dashboard read | API + PostgreSQL |
| Attendance | PRD + User Story | Absensi/report operations | attendances | Camera/upload/check-in/out | API + Nextcloud + worker |
| Leave/absence | PRD + User Story | Ketidakhadiran/master izin | leave tables | Submit/approve/delegate | API + worker |
| Overtime | PRD + User Story | Lembur operations | overtime tables | Submit/approve | API + worker |
| Notifications | PRD + User Story | Notifikasi operations | notifications | Event/read/dismiss | API + worker |
| Access/audit | PRD + User Story | Akses operations | roles/permissions/audit | Permission/audit sequence | API + PostgreSQL |
| Deployment | Product NFR | Health/API base | persistence needs | runtime flows | System Design |

## Repository specification map

| File | Purpose | Load when |
|---|---|---|
| `product-requirements.md` | Product goals, actors, functional/NFR scope, boundaries | Planning feature or validating product scope |
| `access-matrix.md` | Role, row scope, route/action constraints | Implementing auth, queries, UI guards, negative tests |
| `api-data-summary.md` | 42-operation inventory, envelopes, 26-table catalog, constraints | Designing contract, DTO, migration, integration |
| `workflows.md` | State machines and end-to-end interaction rules | Implementing attendance, approval, notification, workers |
| `document-index.md` | Authority, traceability, gaps, version control | Beginning any task or resolving conflict |

These files are navigation/working specifications. They do not replace exact PDF fields.

## Conflict resolution procedure

When a conflict appears:

1. Record the exact topic and both source statements.
2. Confirm versions and authority order.
3. Determine affected surfaces: product, API, DB, workflow, UI, migration, deployment.
4. Apply the higher-authority requirement only when its meaning is clear.
5. If applying it changes an external/data contract, request or record an explicit decision.
6. Update canonical contract/spec first.
7. Update implementation, mocks, tests, tasks, and guidance.
8. Add regression/contract coverage.

Suggested record:

```text
Conflict:
Sources:
Higher authority:
Decision:
Affected contracts:
Migration/compatibility:
Approver/date:
```

Do not “resolve” a conflict only inside code comments.

## Known decisions

- JWT lifetime is eight hours.
- Login and logout are approved; no refresh endpoint is listed.
- Redis key `session:<user_id>` participates in active-session validation.
- Five consecutive login failures lock the account; API uses `429 ACCOUNT_LOCKED`.
- Four roles: `karyawan`, `atasan`, `hr`, `top_management`.
- Top Management is read-only except final decision for HR-owned requests; only one account.
- Employee deletion is soft: inactive status plus `deleted_at`.
- File traffic goes through backend; physical files live in Nextcloud; maximum 5 MB.
- Server/network time is authoritative for attendance; local time is watermark context.
- WFO radius is 100 meters; WFH/WFA have no office-radius restriction.
- Approval supervisor SLA is 2x24 hours and escalates only supervisor -> HR.
- Notification event is idempotent and dismissal is persistent per event.
- Attendance photos are retained for three months; attendance rows remain.
- PostgreSQL 16, Redis 7, Go API/worker, React/Tailwind, Nginx, and Docker are approved.
- Only Nginx exposes public ports.

## Known gaps and ambiguities

Do not decide these silently:

1. Reset-password mechanism is required by lockout policy, but its public endpoint is not
   defined in API Contract v1.1.
2. Frontend session restoration/token persistence must work without inventing refresh or
   current-user endpoints.
3. Exact Top Management access to employee document listing/download must follow the API
   Contract; summaries are insufficient.
4. Contract H-30 fallback for an HR employee without supervisor must choose another HR or
   Top Management explicitly and avoid self-notification.
5. Exact health liveness/readiness response semantics must remain contract/deployment aligned.
6. Exact file URL/download access mechanism must not expose permanent public Nextcloud links
   or credentials.
7. Language, bundler, router, package manager, frontend state/form/test libraries, and most
   backend libraries are not established by the source documents.
8. Exact metric formulas/period definitions and every employee payload field must be read
   directly from the corresponding PDF before implementation.
9. Product requirements mention check-in and checkout, while the operation inventory lists
   only `POST /absensi/checkin`; confirm whether checkout is represented by its request
   action/state before adding another route.
10. The summarized database table names total 25 while the source summary states 26; reconcile
    the omitted table directly from Database Schema v1.1 before migrations.

## Version and change control

- Preserve version number in filename and index.
- Never overwrite a source PDF with revised content under the same version.
- Add a new version and record supersession.
- Record review date and change rationale when summaries are updated.
- Keep OpenAPI/migrations/implementation synchronized with approved source changes.
- Do not commit source PDFs if repository policy or confidentiality prohibits it.
- Never include real employee data when producing examples from sources.

When a new document version arrives:

1. Add it to inventory.
2. Compare changed requirements/contracts.
3. Update authority references.
4. Record breaking/data/migration impact.
5. Update all affected specs/tasks/prompts/skills.
6. Add/adjust tests.

## Reading checklist

Before implementation:

- [ ] Read `CLAUDE.md`.
- [ ] Identify the feature's rows in the traceability table.
- [ ] Read the highest-authority product/story sections.
- [ ] Read exact API operation/payload/error definitions.
- [ ] Read exact schema fields/constraints/indexes.
- [ ] Read relevant sequence.
- [ ] Read system topology for external dependencies.
- [ ] Check known gaps.
- [ ] Record assumptions/decisions rather than guessing.
