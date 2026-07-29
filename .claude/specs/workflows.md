# GSNpeeps — Critical Workflows and State Transitions

Use this specification for service orchestration, transaction boundaries, worker behavior,
frontend states, integration tests, and failure recovery. Exact request/response fields remain
governed by API Contract v1.1.

## Contents

[Cross-workflow principles](#cross-workflow-principles) ·
[Authentication](#authentication) · [Employee lifecycle](#employee-lifecycle) ·
[Employee document upload](#employee-document-upload) ·
[Attendance check-in/out](#attendance-check-inout) ·
[Attendance photo retention](#attendance-photo-retention) ·
[Leave and overtime submission](#leave-and-overtime-submission) ·
[Approval routing](#approval-routing) · [Approval decision](#approval-decision) ·
[Delegation](#delegation) · [Auto-escalation](#auto-escalation) ·
[Notification lifecycle](#notification-lifecycle) ·
[Contract H-30 reminder](#contract-h-30-reminder) ·
[Permission update](#permission-update) · [Audit workflow](#audit-workflow) ·
[Exports and downloads](#exports-and-downloads) ·
[Failure and compensation matrix](#failure-and-compensation-matrix) ·
[Idempotency keys and locks](#idempotency-keys-and-locks) ·
[End-to-end test scenarios](#end-to-end-test-scenarios) · [Open decisions](#open-decisions)

## Cross-workflow principles

- Derive actor, role, employee relation, recipient, and server time from trusted server state.
- Validate authentication and authorization before reading/uploading sensitive data.
- Keep business transitions in services/use cases.
- Use PostgreSQL transactions for atomic durable state.
- Avoid slow external calls inside DB transactions when compensation/durable handoff is safer.
- Use conditional updates/row locks for concurrent state decisions.
- Use deterministic event keys and unique constraints for retry safety.
- Propagate context cancellation, request/job ID, and deadlines.
- Write audit/event data without secrets or file bodies.
- Frontend must render pending/error/retry states without predicting authoritative success.

## Authentication

### Login success

```text
Browser
  -> POST /auth/login
API
  -> validate body/rate limit
  -> load user/role
  -> verify not locked/active
  -> verify password hash
  -> reset consecutive failure counter
  -> issue eight-hour JWT
  -> write Redis session:<user_id> with bounded TTL
  -> write approved login audit
  -> return token + approved identity data
Browser
  -> store token/session according to approved strategy
  -> resolve role/capability
  -> mount protected shell
```

Requirements:

- Do not create JWT if Redis active session cannot be established under fail-closed policy.
- Do not return password hash or session storage internals.
- Do not invent refresh token.

### Login failure and lockout

```text
invalid credentials
  -> atomically increment consecutive failures
  -> if count < 5: safe invalid-credentials response
  -> if count == 5:
       lock account
       invalidate Redis session
       audit lockout
       return 429 ACCOUNT_LOCKED
```

Concurrent failures must not lose increments. Login success or approved reset-password
operation resets the counter.

### Authenticated request

```text
parse Bearer JWT
  -> validate signature/exp
  -> load session:<user_id> from Redis
  -> compare active session state
  -> attach actor identity
  -> coarse permission
  -> service row/state authorization
```

Redis/session mismatch fails authentication.

### Logout

```text
authenticated POST /auth/logout
  -> invalidate Redis session idempotently
  -> write approved logout audit
  -> return success
frontend
  -> cancel protected requests/polling
  -> clear protected cache and user-scoped client state
  -> release media/object URLs
  -> replace navigation to login
```

## Employee lifecycle

### Create

```text
HR request
  -> authenticate + HR permission
  -> decode/validate exact API fields
  -> validate department-position/supervisor/unique identities
  -> transaction:
       insert employee core
       insert approved detail rows
       create/link user when contract requires
       insert position history
       insert audit
  -> commit
  -> return approved employee projection
```

On any transaction failure, no partial employee/detail/user/history should remain.

### Update

```text
HR PUT /karyawan/{id}
  -> load active employee
  -> validate update semantics and relations
  -> detect material position/department change
  -> transaction:
       update approved fields
       append position history when required
       update detail rows
       append redacted audit before/after
  -> commit
```

Verify full-versus-partial PUT semantics from API Contract. Do not overwrite omitted values
without contract authority.

### Soft-deactivate

```text
HR DELETE /karyawan/{id}
  -> validate target/state and protected-role constraints
  -> transaction:
       status = nonaktif
       deleted_at = server time
       invalidate/disable linked session as required
       append audit
  -> commit
```

Do not hard-delete employee history. Define downstream behavior for open approvals/session
from source requirements before implementation.

## Employee document upload

```text
Browser -> multipart to Backend
Backend:
  -> authenticate HR and employee scope
  -> bound body and validate file size/type/signature
  -> generate safe object name
  -> upload to Nextcloud
  -> transaction:
       insert employee_documents metadata
       append safe audit
     if DB fails:
       delete uploaded object or queue reconciliation
  -> return approved metadata/access locator
```

Never expose Nextcloud credentials or accept client path traversal. Exact Top Management read
scope requires contract confirmation.

## Attendance check-in/out

The inventory lists `POST /absensi/checkin` while product requirements include check-in and
checkout. Verify whether the request carries an action/state before implementation.

### Check-in

```text
authenticated employee request
  -> validate multipart/photo/work mode/location
  -> validate JPG/PNG <= 5 MB and signature
  -> get server time
  -> if WFO:
       require office_location_id chosen from active office master
       load that office's trusted coordinates
       calculate distance server-side
       reject 422 OUT_OF_RADIUS when >100 m
  -> if WFH/WFA:
       do not apply office-radius rejection
  -> atomically ensure no duplicate/open attendance
  -> upload photo to Nextcloud
  -> persist attendance + photo URL
     if DB fails: compensate uploaded photo
  -> return server-authoritative result/time
```

Client local time is watermark context only. Client-computed distance/time is not trusted.
Employees may choose any active office; no permanent employee-office assignment is required.
Official office seed coordinates remain operational configuration and must never be fabricated.

### Checkout

```text
authenticated employee request/action
  -> load one open attendance for actor/business day
  -> reject if none
  -> validate required checkout evidence per contract
  -> use server time
  -> atomically close record
  -> return authoritative result
```

Concurrent check-in/out requests must not create duplicate/invalid state.

### Frontend permission failures

- Explain camera purpose before requesting access.
- Camera denied/unavailable -> approved watermarked upload fallback.
- Geolocation denied/unavailable/timeout -> actionable state per contract.
- Stop streams/watches and revoke object URLs on cancel/unmount/logout.

## Attendance photo retention

Daily worker:

```text
acquire distributed singleton lock
  -> select bounded batch where photo older than 3 months and foto_url not null
  -> for each row:
       delete Nextcloud object idempotently
       set foto_url = NULL
       keep attendance row
       record metrics/error
  -> repeat until batch limit/policy
  -> release lock
```

Required:

- Repeat-run safe.
- Missing object reconciles without deleting attendance.
- Partial storage/DB failure is retryable and observable.
- No deletion of recent photos.

## Leave and overtime submission

### Validate

Leave/absence:

- Type is approved Cuti/Izin/Perjalanan Dinas.
- Supporting document required.
- Date range valid.
- Perjalanan Dinas destination and purpose required.
- Balance behavior follows exact schema/PRD.

Overtime:

- Time/duration valid.
- Supporting document optional.
- No compensation calculation.

### Route and persist

```text
authenticated applicant
  -> validate request/document
  -> upload document when present
  -> resolve employee role + direct supervisor
  -> calculate initial stage from routing matrix
  -> transaction:
       create request
       create initial approval/history
       create durable notification event
       create audit
     if DB fails:
       compensate uploaded document
  -> return request/current stage
```

Never accept applicant/approver/current stage from client as authority.

## Approval routing

| Applicant | Initial stage | Next/final stage |
|---|---|---|
| Karyawan with supervisor | `menunggu_atasan` | `menunggu_hr` |
| Karyawan without supervisor | `menunggu_hr` | Final HR |
| Atasan own request | `menunggu_hr` | Final HR |
| HR own request | `menunggu_top_management` | Final Top Management |

Top Management cannot decide non-HR requests. HR cannot self-approve an HR request.

## Approval decision

```text
PUT .../{id}/decision
  -> authenticate actor
  -> load request/current stage/applicant relation
  -> authorize active actor:
       direct supervisor for menunggu_atasan
       HR for menunggu_hr
       Top Management for HR applicant at menunggu_top_management
  -> validate approve/reject; reject note required
  -> transaction + row lock/conditional update:
       compare current stage/not decided
       write immutable approval action
       update request state/stage
       create deterministic durable notification event
       append audit
  -> commit
```

Concurrent behavior:

```text
decision A wins -> success
decision B observes changed version/stage -> 409 ALREADY_DECIDED
```

Frontend refetches and shows latest state; do not optimistically mark final.

## Delegation

```text
active direct supervisor
  -> request is menunggu_atasan and not decided
  -> transaction:
       append delegate history
       move to menunggu_hr
       create HR/applicant notification event
       append audit
  -> commit
```

Only the active direct supervisor may delegate. Delegation is not a final approval and must
remain visible in history/timeline.

## Auto-escalation

Worker:

```text
acquire job lock
  -> server time cutoff = now - 2x24 hours
  -> select/claim bounded requests:
       status/stage == menunggu_atasan
       stage entered before cutoff
  -> per request transaction:
       conditional check still menunggu_atasan
       append auto_escalate history
       move to menunggu_hr
       create deterministic notification event
       append system audit if required
  -> metrics and retry
```

Do not escalate:

- `menunggu_hr`.
- `menunggu_top_management`.
- Decided/rejected requests.
- Requests that have not reached cutoff.

Repeat run must create no duplicate history/event.

## Notification lifecycle

### Creation

```text
domain transition
  -> derive recipient(s)
  -> derive event_type/resource/stage
  -> deterministic event_key
  -> insert notification/durable event
  -> UNIQUE(recipient_user_id, event_key)
       new -> created
       duplicate -> safe no-op/existing result
```

No public notification-create endpoint.

### List and unread

```text
recipient_user_id = actor.user_id
AND dismissed_at IS NULL
```

Unread additionally requires `read_at IS NULL`.

### Mark read

Owner only; set `read_at` if null; repeat is success/idempotent.

### Dismiss

Owner only; set `dismissed_at`; never hard-delete. The record/event key remains so producer
retry cannot recreate it.

### Deep link

Generate from controlled event mapping. Do not store/redirect arbitrary external URLs.

## Contract H-30 reminder

Daily worker:

```text
acquire lock
  -> compute today in Asia/Jakarta
  -> select active contracts ending exactly 30 calendar days from today
  -> add active direct supervisor when present
  -> add every active HR except the employee whose contract expires
  -> if no eligible HR exists, add the single active Top Management user
  -> deduplicate by recipient_user_id and remove self recipient
  -> generate recipient-specific deterministic event key
  -> insert notification idempotently
  -> emit metrics
```

If the Top Management fallback is required but no active Top Management exists, fail and
measure that job item; never silently mark it successful or notify the subject.

## Permission update

```text
HR PUT /akses/permission
  -> authenticate/authorize HR
  -> validate role, permissions, invariant/protected access
  -> transaction:
       replace/update mapping exactly as contract
       append redacted audit before/after
  -> commit
  -> invalidate Redis/permission cache
  -> frontend refetches identity/capabilities
```

Top Management is read-only. Do not optimistically apply permission changes in UI.

## Audit workflow

For an audited operation:

```text
business service
  -> build safe audit entry:
       actor
       action
       resource type/id
       safe before/after or change summary
       server timestamp
       request/job ID
       IP where available
  -> insert append-only record in same transaction when required
```

Database application privilege must prevent audit UPDATE/DELETE. Never log password/token,
document/photo content, storage credential, or unredacted sensitive identity/salary.

## Exports and downloads

```text
authenticated request
  -> authorize role and requested scope
  -> validate filters/format
  -> query approved projection
  -> generate bounded XLSX/PDF
  -> audit download/export when required
  -> stream with safe filename/content type
frontend
  -> create short-lived object URL
  -> trigger download
  -> revoke object URL
```

HR employee/attendance exports have no watermark. Do not cache sensitive export persistently.

## Failure and compensation matrix

| Workflow | Failure | Required outcome |
|---|---|---|
| Login | Redis session write fails | No usable unchecked login |
| Employee multi-table write | Any DB step fails | Full transaction rollback |
| File upload | Nextcloud fails | No DB metadata/business success |
| File upload | DB fails after upload | Delete object or durable orphan cleanup |
| Attendance | Duplicate/concurrent check-in | One success, no duplicate row/orphan file |
| Approval | Concurrent decision | One success, loser `409 ALREADY_DECIDED` |
| Event creation | Retry | Unique key prevents duplicate |
| Dismissed event | Producer retries | Remains dismissed/not recreated |
| Retention | File missing | Reconcile idempotently; preserve attendance row |
| Retention | DB URL clear fails | Retry/reconcile deleted-object reference |
| Permission update | Cache invalidation fails | Fail/retry safely; do not retain misleading access indefinitely |
| Worker | Process stops mid-batch | Claimed items recover and repeat safely |

## Idempotency keys and locks

Use:

- Notification: `(recipient_user_id, event_key)` unique.
- Approval: request version/current-stage conditional update and unique/history policy.
- Attendance: employee/business-date/open-state uniqueness/conditional update.
- Worker singleton: Redis lock with owner token, TTL, safe renewal/release.
- Batch claim: DB conditional claim/locking; not process memory.
- Logout/read/dismiss: idempotent state set/delete behavior.

Never use Redis lock as the only durable business invariant.

## End-to-end test scenarios

### Auth

- Login -> protected request -> logout -> old token fails.
- Four invalid attempts then fifth lockout -> active session invalid.

### Employee/storage

- HR create/update/upload/export/soft-delete.
- Top Management read only.
- Karyawan direct employee/document request forbidden with no side effect.
- Upload DB failure removes orphan.

### Attendance

- WFO inside/at/outside 100 m.
- WFH/WFA no radius rejection.
- Camera denied fallback.
- Duplicate check-in and checkout without open record.
- Retention repeat run.

### Approval

- Every routing-matrix row.
- Reject note.
- Direct versus unrelated supervisor.
- Delegate and 2x24-hour escalation.
- Two simultaneous decisions.

### Notification/access/audit

- Event once, read, dismiss, producer retry.
- Cross-recipient denial.
- Contract H-30 before/at/after boundary, subject exclusion, all-HR fan-out, and Top
  Management fallback.
- HR permission change; Top Management mutation forbidden.
- Audit insert visible; app update/delete denied.

## Open decisions

This is the workflow subset of the canonical gaps in `document-index.md`:

- Browser token persistence strategy; identity restoration uses approved `/auth/me`.
- Exact permission-cache failure policy.
- File access/download URL mechanism.
- Official office seed coordinates and company/public holiday calendar.
