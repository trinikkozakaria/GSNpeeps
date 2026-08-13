# GSNpeeps — Access Control Matrix

Gunakan spesifikasi ini untuk backend authorization, frontend route/action UX, query scope,
test fixtures, dan negative authorization tests.

## Contents

[Authorization model](#authorization-model) · [Role definitions](#role-definitions) ·
[Capability matrix](#capability-matrix) · [Endpoint access matrix](#endpoint-access-matrix) ·
[Row and state scope](#row-and-state-scope) · [Approval authorization](#approval-authorization) ·
[Sensitive-field policy](#sensitive-field-policy) ·
[Frontend behavior](#frontend-behavior) · [Backend enforcement](#backend-enforcement) ·
[Negative test matrix](#negative-test-matrix) · [Open decisions](#open-decisions)

## Authorization model

Authorize every protected operation using:

```text
valid JWT
  + active Redis session
  + role/capability
  + resource ownership or organization relation
  + active resource/workflow state
```

Rules:

- Role comes from server-authenticated identity, never request body/query.
- Route permission is only the coarse first gate.
- Service/use-case enforces ownership, direct-report, active approval stage, and read-only
  behavior after loading authoritative data.
- A forbidden request must not create/update database rows, files, events, notifications, or
  business audit side effects.
- Security-denial audit may be written only according to approved audit policy.
- Frontend hiding/redirect is UX, not security.

## Role definitions

### `karyawan`

- Own profile and current personal metrics.
- Own attendance actions and result/history exposed by contract.
- Own leave/overtime submissions and history.
- Own notifications.
- No global Employee Database, HR Dashboard, live feed/report, Access, or Audit.

### `atasan`

- Includes Karyawan scope for self.
- Sees and decides only direct-report requests at active supervisor stage.
- May delegate an eligible active supervisor-stage request to HR.
- No unrelated team/global employee data.

### `hr`

- Employee and organization administration according to API Contract.
- HR dashboard, attendance live feed/report/export.
- Final approval for Karyawan/Atasan requests.
- Own requests route to Top Management.
- Master leave type and permission administration.
- Audit read.

### `top_management`

- Exactly one user.
- Read-only monitoring for allowed Employee, Dashboard, Attendance Report, Access, and Audit
  operations.
- No Personal Metrics.
- No ordinary create/update/delete/export unless API Contract explicitly allows a read/export.
- Only mutation exception: final decision on active requests submitted by HR.

## Capability matrix

Legend:

- `Own` — authenticated user's resource only.
- `Direct` — direct reports only.
- `Read` — global read-only according to operation fields.
- `Full` — permitted operations still subject to validation/state.
- `No` — backend rejects and frontend must not fetch.
- `Conditional` — exact contract/state relation required.

| Area | Karyawan | Atasan | HR | Top Management |
|---|---|---|---|---|
| Login/logout/self-reset/change password | Own | Own | Own | Own |
| Own profile | Own read | Own read | Own read | No |
| Personal metrics | Own | Own | Own | No |
| Department/position/office reference | Read for allowed forms/views | Read for allowed forms/views | Read | Read where route uses it |
| Employee database | No | No | Full CRUD + export | Read |
| Employee documents | No global | No global | Upload/read | Conditional read only |
| HR dashboard/org chart | No | No | Read/full monitoring | Read |
| Own attendance | Own | Own | Own | No unless contract says otherwise |
| Attendance live feed/report | No | No | Read + export | Read, no mutation |
| Own leave/overtime | Create/read own | Create/read own | Create/read own | No regular flow |
| Supervisor approvals | No | Direct active stage | No as supervisor unless relation/role rules say so | No |
| HR final approvals | No | No | Active Karyawan/Atasan requests | No |
| HR-request final approval | No | No | Applicant only | Conditional active decision |
| Delegation to HR | No | Direct active supervisor stage | No | No |
| Master leave type | No | No | Full per API | No unless read operation explicitly allows |
| Overtime recap | No | No | Read | Read only if API Contract permits |
| Notifications | Own | Own | Own | Own |
| Role/permission | No | No | Read/update per API | Read |
| Audit log | No | No | Read | Read |

## Endpoint access matrix

Exact payload/response fields remain governed by API Contract v1.1.

### System and authentication

| Method/path | Karyawan | Atasan | HR | Top Management | Scope |
|---|---|---|---|---|---|
| `GET /health` | Public | Public | Public | Public | No user data |
| `POST /auth/login` | Public | Public | Public | Public | Account being authenticated |
| `POST /auth/reset-password` | Public | Public | Public | Public | Account verified by email + current password |
| `POST /auth/logout` | Own | Own | Own | Own | Current active session |
| `GET /auth/me` | Own | Own | Own | Own | Current authenticated identity |
| `PATCH /auth/me/password` | Own | Own | Own | Own | Current account; revoke all sessions |

### Organization and employee

| Method/path | Karyawan | Atasan | HR | Top Management | Scope |
|---|---|---|---|---|---|
| `GET /master/departemen` | Authenticated reference | Authenticated reference | Read | Read | Return approved master fields |
| `GET /master/jabatan` | Authenticated reference | Authenticated reference | Read | Read | Return approved master fields |
| `GET /master/lokasi-kantor` | Read | Read | Read | Read | Active trusted WFO locations |
| `GET /karyawan` | No | No | Read | Read | HR/TM projection; TM read-only |
| `GET /karyawan/{id}` | No | No | Read | Read | Approved detail projection |
| `POST /karyawan` | No | No | Create | No | HR only |
| `PUT /karyawan/{id}` | No | No | Update | No | HR only |
| `DELETE /karyawan/{id}` | No | No | Soft-delete | No | HR only |
| `GET /karyawan/export` | No | No | Export | No | HR only |
| `POST /karyawan/{id}/dokumen` | No | No | Upload | No | HR only |
| `GET /karyawan/{id}/dokumen` | No | No | Read | Read | Approved metadata; TM read-only |

### Profile and dashboard

| Method/path | Karyawan | Atasan | HR | Top Management | Scope |
|---|---|---|---|---|---|
| `GET /profil/saya` | Own | Own | Own | No | Identity from JWT relation |
| `GET /profil/saya/metrik` | Own | Own | Own | No | Current personal scope |
| `GET /dashboard/metrik` | No | No | Read | Read | TM read-only |

### Attendance and reports

| Method/path | Karyawan | Atasan | HR | Top Management | Scope |
|---|---|---|---|---|---|
| `POST /absensi/checkin` | Own | Own | Own | No | Own attendance action defined by contract |
| `GET /absensi/livefeed` | No | No | Read | Read | Global monitoring |
| `GET /laporan/kehadiran` | No | No | Read | Read | Global report |
| `GET /laporan/kehadiran/export` | No | No | Export | No unless contract says otherwise | HR export |

### Leave/absence

| Method/path | Karyawan | Atasan | HR | Top Management | Scope |
|---|---|---|---|---|---|
| `POST /ketidakhadiran` | Own create | Own create | Own create | No regular create | Applicant from JWT |
| `GET /ketidakhadiran` | No | Direct active inbox | HR active inbox | HR-owned active inbox | Approver scope |
| `GET /ketidakhadiran/{id}` | Own | Own or direct active approver | Own or HR approver | HR applicant active approver | Ownership/stage |
| `GET /ketidakhadiran/saya` | Own | Own | Own | No | Applicant only |
| `PUT /ketidakhadiran/{id}/decision` | No | Direct active supervisor stage | Active HR stage | Active HR-applicant final stage | One atomic decision |
| `PUT /ketidakhadiran/{id}/delegate` | No | Direct active supervisor stage | No | No | Supervisor -> HR only |

### Master leave type

| Method/path | Karyawan | Atasan | HR | Top Management | Scope |
|---|---|---|---|---|---|
| `GET /master/jenis-izin` | Read active | Read active | Read | Read active | Non-HR dibatasi jenis izin aktif |
| `POST /master/jenis-izin` | No | No | Create | No | HR only |
| `PUT /master/jenis-izin/{id}` | No | No | Update | No | HR only |

### Overtime

| Method/path | Karyawan | Atasan | HR | Top Management | Scope |
|---|---|---|---|---|---|
| `POST /lembur` | Own create | Own create | Own create | No regular create | Applicant from JWT |
| `GET /lembur` | No | Direct active inbox | HR active inbox | HR-owned active inbox | Approver scope |
| `GET /lembur/{id}` | Own | Own or direct active approver | Own or HR approver | HR applicant active approver | Ownership/stage |
| `PUT /lembur/{id}/decision` | No | Direct active supervisor stage | Active HR stage | Active HR-applicant final stage | One atomic decision |
| `GET /lembur/rekap` | No | No | Read | Read | No compensation calculation; TM read-only |

### Access and audit

| Method/path | Karyawan | Atasan | HR | Top Management | Scope |
|---|---|---|---|---|---|
| `GET /akses/role` | No | No | Read | Read | Approved role projection |
| `GET /akses/permission` | No | No | Read | Read | Approved permission projection |
| `PUT /akses/permission` | No | No | Update | No | HR only |
| `GET /akses/audit-log` | No | No | Read | Read | Filtered/paginated and redacted |

### Notifications

| Method/path | Karyawan | Atasan | HR | Top Management | Scope |
|---|---|---|---|---|---|
| `GET /notifikasi` | Own | Own | Own | Own | `recipient_user_id = actor` |
| `GET /notifikasi/unread-count` | Own | Own | Own | Own | Own non-dismissed |
| `PUT /notifikasi/{id}/read` | Own | Own | Own | Own | Owner only, idempotent |
| `DELETE /notifikasi/{id}` | Own | Own | Own | Own | Owner only, soft-dismiss |

## Row and state scope

### Own-resource scope

Resolve user/employee from JWT context. Ignore/reject a body/query `user_id` that attempts to
change ownership.

Applies to:

- Profile and personal metrics.
- Own attendance.
- Own request history/detail.
- Own notifications.

### Direct-report scope

Atasan may approve only when:

```text
request.applicant_employee.atasan_id == actor.employee_id
AND request.current_stage == menunggu_atasan
AND request not already decided
```

Do not grant access from role alone.

### HR scope

HR has organization-wide administration/monitoring where listed, but:

- HR own leave/overtime requests cannot be self-approved.
- HR request routes to Top Management.
- HR still follows validation, state transition, and audit rules.

### Top Management scope

- Global monitoring read-only.
- No personal metrics.
- No employee/access mutations.
- Decision only when applicant role is HR and active stage is Top Management.

## Approval authorization

| Applicant condition | Active actor | Allowed actions |
|---|---|---|
| Karyawan with supervisor, `menunggu_atasan` | That direct Atasan | Approve, reject, delegate |
| Karyawan without supervisor, `menunggu_hr` | HR | Approve, reject |
| Atasan applicant, `menunggu_hr` | HR | Approve, reject |
| Karyawan after supervisor approval/delegation/escalation, `menunggu_hr` | HR | Approve, reject |
| HR applicant, `menunggu_top_management` | Top Management | Approve, reject |
| Any completed/rejected request | None | Read according to ownership; no decision |

Two concurrent actors cannot both decide. The stale loser receives `409 ALREADY_DECIDED`.

## Sensitive-field policy

Return the minimum fields necessary.

Never return:

- Password hash or reset secret.
- JWT/session/Redis values.
- Nextcloud credentials.
- Raw internal storage credential/path when not contract-approved.
- Unredacted audit secrets/document content.

Restrict identity, salary, address, BPJS, NPWP, KTP, photo, and document fields according to
operation/role contract. A role with employee-list access does not automatically receive all
sensitive detail fields.

## Frontend behavior

- Resolve auth/capability before protected request/render.
- Build navigation from centralized capabilities.
- Hide irrelevant disallowed actions.
- Use disabled state with explanation for temporarily unavailable allowed actions.
- Render Top Management monitor pages read-only.
- Never fetch hidden sensitive columns.
- Handle `401` by clearing session/cache; handle `403` without logout.
- On `409 ALREADY_DECIDED`, refetch and display the latest state.
- Do not persist permission/sensitive response across logout.

## Backend enforcement

Apply:

1. Authentication middleware.
2. Coarse role/capability middleware.
3. Service-level ownership/relation/state policy.
4. Repository queries scoped by actor where practical.
5. Transaction/conditional update for mutations.
6. Audit of approved security/business operations.

Never accept role, applicant, approver, recipient, or ownership from client input when it can
be derived from authenticated/server state.

## Negative test matrix

At minimum test:

- Anonymous calls every protected module.
- Karyawan direct URL/API to Employee, Dashboard, Live Feed, Report, Access, Audit.
- Atasan reads/decides unrelated employee request.
- Atasan decides request after stage advanced.
- HR self-approves own request.
- Top Management mutates employee/permission/master data.
- Top Management decides non-HR request.
- User lists/reads/dismisses another recipient's notification.
- User changes path/body ID to another employee/resource.
- Forbidden upload/export/download creates no file/result.
- Permission change takes effect despite prior Redis/frontend cache.

Assert response plus absence of database/storage/event side effects.

## Open decisions

This is the access-control subset of the canonical gaps in `document-index.md`:

- Exact scoped-not-found versus forbidden response per operation.
- Exact permission identifier catalog from schema/seed.
- Exact file delivery URL/proxy authorization mechanism.

Resolve using API Contract/PRD rather than expanding access from this summary.
