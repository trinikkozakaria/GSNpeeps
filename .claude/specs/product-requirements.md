# GSNpeeps — Product Requirements Specification

## Contents

[Overview](#overview) · [Problem statement](#problem-statement) ·
[Goals](#goals) · [Non-goals](#non-goals) · [Actors and organization](#actors-and-organization) ·
[Functional scope](#functional-scope) · [Business rules](#business-rules) ·
[Data and security requirements](#data-and-security-requirements) ·
[Non-functional requirements](#non-functional-requirements) ·
[Information architecture](#information-architecture) ·
[Implementation phases](#implementation-phases) · [Success criteria](#success-criteria) ·
[Open decisions](#open-decisions) · [Traceability](#traceability)

## Overview

GSNpeeps is an internal web application resembling an HRIS. It centralizes employee data,
HR metrics, photo attendance, leave/overtime approval, notifications, role/permission
administration, and audit evidence.

The approved phase focuses on:

- Employee Database.
- Personal Profile and Metrics.
- HR/Top Management Dashboard.
- WFO/WFH/WFA Attendance.
- Leave/Absence and Overtime Approval.
- In-app Notifications.
- User Management, RBAC, and Audit Log.

## Problem statement

Without one controlled system, employee information and HR operations risk:

- Duplicate/inconsistent employee records.
- Limited visibility into headcount, turnover, attendance, leave, and payroll cost.
- Unclear attendance evidence and work-mode/location handling.
- Manual, untraceable, or duplicated approval decisions.
- Role/organizational data leakage.
- Missed contract-expiry follow-up.
- Notifications that duplicate or disappear.
- Weak auditability for sensitive writes and permission changes.

GSNpeeps must become the authoritative operational interface while PostgreSQL remains the
durable business-data source.

## Goals

### Primary goals

- Provide a single source of truth for employee and organization data.
- Give HR and Top Management reliable monitoring metrics.
- Support accountable photo attendance for WFO, WFH, and WFA.
- Automate deterministic leave/overtime approval routing.
- Enforce role, ownership, direct-report, and approval-stage access.
- Produce idempotent, recipient-scoped in-app notifications.
- Preserve immutable audit evidence for sensitive operations.

### Secondary goals

- Reduce manual follow-up for contract expiry and approval SLA.
- Provide employee/attendance exports for authorized HR use.
- Offer a consistent, accessible, responsive internal UI.
- Establish a modular monolith foundation that can grow without premature microservices.

## Non-goals

The current phase does not include:

- Full recruitment workflow.
- Recruitment-cost calculation.
- Benefit management.
- Employee self-service profile editing.
- Attendance reminders.
- Overtime compensation calculation.
- Travel budget approval.
- Email/SMS/WhatsApp/push notification channels.
- Facial recognition or biometric identity matching.
- Payroll processing.
- Arbitrary workflow/role builders beyond the approved scope.

UI may show **Coming Soon** only for Hiring Progress, Recruitment Cost, and Benefit.

## Actors and organization

### Organization model

```text
Company
|-- Departments
|   |-- Positions
|   `-- Employees
`-- Employees
    `-- Direct supervisor (employees.atasan_id, nullable)
```

The direct-supervisor relation determines supervisor-stage approval and contract
notifications.

### Roles

#### Karyawan

- Read own profile and current personal metrics.
- Check in/out for own attendance.
- Create/read own leave and overtime requests.
- Read/dismiss own notifications.
- Cannot browse employee database, HR dashboard, access administration, or audit.

#### Atasan

- Include Karyawan abilities.
- Read/decide active requests from direct reports.
- Delegate eligible supervisor-stage decisions to HR.
- Cannot see unrelated employee requests or HR administration.

#### HR

- Manage employee and organization data.
- View HR dashboard, live attendance, reports, exports, and org chart.
- Perform final approval for employee/supervisor requests.
- Manage master leave types, roles/permissions, and view audit.
- Submit own requests, which route to Top Management.

#### Top Management

- Exactly one user.
- Read-only monitoring for employee database, dashboard, attendance reports, access, and audit.
- No Personal Metrics.
- May perform final decision only for requests submitted by HR.
- Must not receive ordinary mutation controls.

## Functional scope

### 1. Authentication and session

Features:

- Public login.
- Public rate-limited self-reset with current-password verification.
- Authenticated logout.
- Authenticated current-user restoration and own-password change.
- JWT Bearer with eight-hour lifetime.
- Active session cross-check in Redis.
- Five consecutive failures lock the account.
- Logout/lockout invalidates active session immediately.
- Default authenticated rate limit: 120 requests/minute/user.

Requirements:

- Never log or expose password/token.
- Self-reset uses generic failures, account+IP rate limiting, the login failure counter,
  revokes all sessions on success, and never exposes a password to HR.
- Forgot-password via email/OTP remains outside scope.
- Frontend must not display protected content before auth resolution.
- Backend remains final authorization authority.
- No refresh endpoint is approved.

### 2. Organization master

Features:

- Read departments.
- Read positions.
- Read active office locations for WFO selection.
- Use department/position relations in employee forms, filters, org chart, and validation.

Requirements:

- Position/department selections must remain consistent with schema.
- Employees may choose any active office for WFO; there is no permanent employee-office
  assignment.
- HR owns organization data changes only where API Contract permits them.

### 3. Employee database

Features:

- HR employee list, search/filter, pagination, detail, create, update, soft-delete.
- HR export in XLSX/PDF without watermark.
- Employee document upload/list through backend to Nextcloud.
- Top Management read-only monitoring.

Data areas include:

- Core employee/organization/status.
- Address.
- KTP/identity.
- Contract.
- BPJS.
- NPWP.
- Emergency contacts.
- Education.
- Position history.
- Salary.
- Documents.

Requirements:

- Exact fields/types come from Database Schema/API Contract.
- Unique identities/contracts follow database constraints.
- Soft delete sets inactive status and `deleted_at`; history remains.
- Files are maximum 5 MB and never stored as DB binary.
- Karyawan/Atasan cannot access global employee database/documents.

### 4. Own profile and personal metrics

Features:

- Read-only own profile.
- Personal attendance/leave and approved metrics.
- Salary display limited to current month.

Requirements:

- No self-service data editing.
- Top Management has no Personal Metrics.
- Identity is resolved from authenticated user, never request body.

### 5. HR dashboard

Required metrics/views:

- Headcount.
- New joiners.
- Resignations.
- Turnover.
- Attendance.
- Leave/absence.
- Payroll cost.
- Gender ratio.
- Department composition.
- Organization chart.

Requirements:

- HR full view; Top Management read-only.
- Period filter supports daily, weekly (Monday-Sunday), monthly, and yearly calendar ranges
  using an anchor date and Asia/Jakarta boundaries.
- Attendance counts valid check-ins only; check-ins after 09:00:00 WIB are late.
- Active and inactive employee counts/department composition are presented separately.
- Missing gender is shown as `belum_diisi`, never counted as male or female.
- Period and no-data/zero meaning must be explicit.
- Hiring Progress, Recruitment Cost, and Benefit remain Coming Soon.
- Do not fabricate dashboard records.

### 6. Attendance

Features:

- Photo check-in/out.
- Work modes WFO, WFH, WFA.
- GPS/location/address data.
- Local time watermark and server/network time.
- Camera live flow and watermarked file-upload fallback.
- HR live feed and attendance report.
- HR XLSX/PDF export.

Requirements:

- Server/network time is authoritative.
- WFO requires distance <=100 meters from approved office coordinates.
- WFH/WFA do not use office-radius restriction.
- Multiple offices use `office_locations`; employees may choose any active office for WFO,
  with no permanent employee-office assignment.
- Official addresses/coordinates remain pending seed configuration and must not be fabricated.
- Regular workdays are Monday-Friday, 09:00-18:00 `Asia/Jakarta`. Check-in after 09:00 is
  late; early checkout is allowed and recorded.
- JPG/PNG only, maximum 5 MB.
- Prevent duplicate check-in and checkout without open check-in.
- No attendance reminder.
- Delete physical photos after three months but keep attendance row with `foto_url = NULL`.

### 7. Leave/absence

Types:

- Cuti.
- Izin.
- Perjalanan Dinas.

Features:

- Submit request and supporting document.
- Own history/detail.
- Approver inbox/detail.
- Approve/reject.
- Supervisor delegation to HR.
- Master leave-type management by HR.

Requirements:

- Supporting document is required for all leave/absence types.
- Perjalanan Dinas also requires destination and assignment purpose.
- Reject requires note.
- Budget approval is outside scope.

### 8. Overtime

Features:

- Submit with time/duration and optional document.
- Approver list/detail/decision.
- HR recap.

Requirements:

- Follow the same role-routing principles.
- Supporting document is optional.
- Duration may be stored/reported.
- Compensation is calculated manually outside GSNpeeps.

### 9. Approval routing

```text
Karyawan with supervisor -> Atasan -> HR
Karyawan without supervisor -> HR
Atasan own request -> HR
HR own request -> Top Management
```

Requirements:

- Reject ends workflow.
- Supervisor approve advances to HR.
- Supervisor may delegate to HR.
- Auto-escalation after 2x24 hours applies only supervisor -> HR.
- No HR -> Top Management automatic escalation.
- Decisions are atomic; one concurrent winner.
- Stale/concurrent loser receives `409 ALREADY_DECIDED`.
- Approval/delegation/escalation history is preserved.

### 10. Notifications

Features:

- Recipient-scoped list.
- Unread count.
- Mark read.
- Dismiss.
- Safe internal deep link.
- Events for submission, transition, decision, delegation, escalation, and contracts ending
  exactly 30 calendar days later in timezone Asia/Jakarta.

Requirements:

- Notifications are created server-side from events.
- Deterministic `event_key`.
- `UNIQUE(recipient_user_id, event_key)` prevents duplicates.
- Dismiss sets `dismissed_at`, not hard delete.
- Retried dismissed event must not reappear.
- H-30 goes to the active direct supervisor plus every eligible active HR except the subject;
  if no eligible HR exists, it falls back to the single active Top Management account.

### 11. Access administration

Features:

- Role list.
- Permission catalog/read.
- HR permission update.
- Top Management read-only view.

Requirements:

- Four role identifiers remain stable.
- Permission updates invalidate cached capability/session-derived UI promptly.
- Backend enforces every route/row/action.

### 12. Audit log

Required operations include:

- Login/logout and relevant security events.
- Employee create/update/delete.
- Approval/reject/delegate.
- Relevant downloads.
- Permission changes.

Requirements:

- Append-only.
- Application DB user cannot UPDATE/DELETE audit rows.
- Store actor, action, resource, time, request ID, IP where available, and safe change context.
- Never store secrets, password/token, document content, or unredacted sensitive data.

## Business rules

### Employee

- UUID primary keys via `gen_random_uuid()`.
- Foreign keys default `ON DELETE RESTRICT`.
- NIP and approved identity/contract fields unique.
- Salary unique by `(employee_id, periode)`.
- Direct supervisor is nullable employee self-reference.

### Leave balance

- Unique by `(user_id, tahun)`.
- Consumption/calculation must follow exact schema/PRD rules; do not infer missing formulas.

### Attendance

- Business-day/timezone boundary must be configured consistently.
- Client-calculated time/distance cannot override server result.
- Storage upload and DB metadata require compensation/reconciliation.

### Approval

- Active-stage actor and relationship determine permission.
- Use transaction and conditional update/row lock.
- History and notification event must not be lost relative to state change.

### Notification

- Recipient/event uniqueness is the final idempotency defense.
- Read and dismiss are owner-only and idempotent.

## Data and security requirements

- PostgreSQL 16 is durable business source.
- Redis 7 stores active session, rate limit, short-lived cache/locks only.
- Nextcloud WebDAV stores physical files; PostgreSQL stores safe locator/metadata.
- Browser communicates with Nextcloud only through an approved backend-controlled access
  mechanism; credentials are never exposed.
- Only Nginx exposes a public port.
- Parameterized SQL and explicit columns.
- Typed validation at API boundaries.
- Role plus row/state scope on backend.
- Structured logs with request/job correlation and redaction.
- Synthetic data only in tests, screenshots, demos, and documentation.

## Non-functional requirements

### Reliability

- Graceful API/worker shutdown.
- Context cancellation and dependency timeouts.
- Idempotent worker/jobs/events.
- Bounded retries and batches.
- Migration and rollback/forward-fix planning.
- Backup/restore notes for PostgreSQL and Nextcloud.

### Performance

- Server-side pagination/filtering.
- Index common filters/joins/worker scans.
- Avoid N+1 query/request waterfalls.
- Lazy-load heavy frontend camera/chart/export functionality when supported.
- Use reasonable polling and stop it on logout/unmount.

### Accessibility

- Keyboard-operable critical workflows.
- Visible labels/focus, semantic landmarks/headings.
- Status not communicated by color alone.
- Mobile/narrow viewport and 200% zoom.
- Accessible chart/table alternative, camera, geolocation, timeline, and notification controls.

### Privacy

- Minimize returned/cached employee data.
- No sensitive browser persistence.
- Clear protected cache/state at logout/401.
- No PII in logs, fixtures, screenshots, or error details.

### Maintainability

- Modular monolith with API and worker composition roots.
- Feature-oriented React with shared accessible primitives.
- OpenAPI as frontend/backend contract.
- Frontend baseline: React JavaScript/JSX, Vite, React Router, Tailwind CSS, Axios,
  TanStack Query, React Hook Form, Zod, Vitest, Testing Library, Playwright, and pnpm.
- Backend baseline: Go, `net/http` + `gorilla/mux`, pgx, Goose,
  go-playground/validator, golang-jwt/jwt/v5, go-redis, `slog`, and Testify.
- Additional non-baseline libraries require a focused need and must not duplicate an
  approved responsibility.
- Automated format/lint/test/build/contract checks.

## Information architecture

Expected navigation, filtered by role:

```text
Login
Dashboard
Employee Database
Profile / Personal Metrics
Attendance
Attendance Live Feed / Reports
Leave / Absence
Overtime
Notifications
Access
  |-- Roles
  |-- Permissions
  `-- Audit Log
```

Do not show inaccessible navigation, but still enforce backend authorization.

## Implementation phases

### Phase 0 — Foundation

- Monorepo, stack decisions, Docker/Nginx, API/worker/frontend skeleton.

### Phase A — Contract

- OpenAPI for approved 46 operations in contract revision 0.4.0.

### Phase B — Vertical slices

1. Auth/RBAC.
2. Employee/Profile/Dashboard.
3. Attendance.
4. Leave/Overtime Approval.
5. Notifications/Access/Audit.

### Phase C — Integration/release

- Real API integration, four-role E2E, security, accessibility, operations, rollback.

## Success criteria

- Employee data is consistent and role-protected.
- HR can perform complete employee administration and exports.
- Employees can complete attendance/request workflows with accessible failure recovery.
- Every approval route follows organization/role rules and is race-safe.
- Notifications are recipient-scoped and non-duplicating.
- Audit is immutable and meaningful.
- Dashboard metrics are approved, period-aware, and not fabricated.
- Critical four-role negative authorization tests pass.
- Production assets/services follow Nginx-only public topology.
- No critical secret/PII/data-integrity/concurrency issue remains.

## Open decisions

Product-level subset; the canonical complete list is in `document-index.md`:

- Exact browser token-persistence strategy without a refresh endpoint.
- File download/URL authorization mechanism.
- Official office names, addresses, and coordinates for seed data.
- Company/public holiday calendar.
- Task-specific optional libraries not covered by the approved baseline, only when needed.

Resolve through `document-index.md`; do not silently add behavior.

## Traceability

- Role/action detail: `access-matrix.md`.
- Paths/envelopes/tables: `api-data-summary.md`.
- State machines/sequences: `workflows.md`.
- Source versions/conflicts/gaps: `document-index.md`.
- Execution milestones: `.claude/skills/1.TASK.md` through `8.TASK_INTEGRATION_RELEASE.md`.
