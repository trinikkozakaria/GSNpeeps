# GSNpeeps — API and Data Contract Summary

This working specification inventories the approved API modules and database groups. Read
the API Contract v1.1 and Database Schema v1.1 PDFs before writing exact payload fields,
column types, constraints, or indexes.

## Contents

[Contract conventions](#contract-conventions) · [Authentication and transport](#authentication-and-transport) ·
[Response envelopes](#response-envelopes) · [Pagination and filtering](#pagination-and-filtering) ·
[Error model](#error-model) · [Operation inventory](#operation-inventory) ·
[Operation-to-data map](#operation-to-data-map) · [Database conventions](#database-conventions) ·
[Table catalog](#table-catalog) · [Relationship map](#relationship-map) ·
[Critical constraints and indexes](#critical-constraints-and-indexes) ·
[File and export contract](#file-and-export-contract) ·
[Consistency boundaries](#consistency-boundaries) ·
[Contract gaps](#contract-gaps) · [Implementation checklist](#implementation-checklist)

## Contract conventions

- API base path: `/api/v1`.
- Candidate production host from source: `https://api.janjikupadamu.id`; treat as
  configuration, not product identity.
- JSON field names: English `snake_case`.
- Path/query/status/payload: API Contract v1.1 is authoritative.
- Database: PostgreSQL 16.
- Physical files: Nextcloud via backend WebDAV adapter.
- Session/rate limit/locks: Redis 7.
- Every protected route requires Bearer authentication and active Redis session.
- Do not add an operation because it seems conventional.

## Authentication and transport

Public operations:

- `GET /health`.
- `POST /api/v1/auth/login`.

All other operations require:

```text
Authorization: Bearer <JWT>
```

JWT:

- Lifetime eight hours.
- Minimal claims include `user_id`, `role`, `exp`.
- Cross-checked against `session:<user_id>` in Redis.
- Invalidated by logout/lockout.
- No approved refresh endpoint.

Content types:

- JSON for normal requests/responses.
- `multipart/form-data` for photo/document upload.
- Binary/file response for approved XLSX/PDF exports.

## Response envelopes

### Success

```json
{
  "success": true,
  "data": {},
  "message": "Deskripsi singkat hasil"
}
```

### Paginated list

```json
{
  "success": true,
  "data": [],
  "message": "Data berhasil dimuat",
  "meta": {
    "page": 1,
    "limit": 20,
    "total_data": 134,
    "total_page": 7
  }
}
```

### Error

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Data belum valid",
    "fields": {
      "email": "Format email tidak valid"
    }
  }
}
```

Rules:

- `error.fields` is for field-level validation only.
- Do not expose SQL, stack, internal path, password/token, or storage credential.
- Every handler/client/mock must preserve the contract shape.
- Do not treat an error as an empty success list.

## Pagination and filtering

Approved list convention:

- Request: `page`, `limit`.
- Response: `meta.page`, `meta.limit`, `meta.total_data`, `meta.total_page`.
- Feature filters/search/sort must be listed by API Contract.
- Use deterministic ordering including stable tie-breaker.
- Validate min/max/enum/date values.
- Do not load all rows for client-side pagination/filtering.

The frontend should keep shareable list filters/page in the URL and reset page when filters
change.

## Error model

Use exact codes/statuses from API Contract. Known cross-feature rules include:

| Condition | Expected contract behavior |
|---|---|
| Invalid JSON/parameter | `400`-class documented input error |
| Missing/invalid/expired/inactive session | `401` |
| Valid session without access | `403` |
| Resource absent/out of approved scope | Contract-defined `404`/`403` |
| Duplicate/invalid current state | `409` where documented |
| Field/business validation | `422`, optional `error.fields` |
| Fifth login failure/account locked | `429 ACCOUNT_LOCKED` |
| WFO outside 100 m | `422 OUT_OF_RADIUS` |
| Concurrent/stale approval decision | `409 ALREADY_DECIDED` |
| Unexpected dependency/server failure | `500`-class safe internal error |

Do not invent resource-specific codes; transcribe them from the PDF/OpenAPI.

## Operation inventory

API Contract PDF v1.1 defines **42 operations in 13 modules**. Contract revision 0.4.0
adds four approved operations, so the active OpenAPI contains **46 operations**.

Revision 0.4.0 additions:

| Method | Path | Access | Purpose |
|---|---|---|---|
| GET | `/api/v1/auth/me` | Authenticated | Restore current identity and role |
| PATCH | `/api/v1/auth/me/password` | Authenticated | Change own password and revoke sessions |
| POST | `/api/v1/auth/reset-password` | Public, rate-limited | Self-reset after current-password verification |
| GET | `/api/v1/master/lokasi-kantor` | Authenticated | Active trusted office coordinates for WFO |

### 1. System — 1

| # | Method | Path | Access | Purpose |
|---:|---|---|---|---|
| 1 | GET | `/health` | Public | Service/dependency health according to contract |

### 2. Authentication — 2

| # | Method | Path | Access | Purpose |
|---:|---|---|---|---|
| 2 | POST | `/api/v1/auth/login` | Public | Authenticate, apply lockout, issue JWT/session |
| 3 | POST | `/api/v1/auth/logout` | Authenticated | Invalidate current active session |

The PDF lists no current-user or reset operation; revision 0.4.0 adds them. Refresh,
registration, HR password reset, and forgot-password email/OTP remain out of scope.

### 3. Organization master — 2

| # | Method | Path | Access | Purpose |
|---:|---|---|---|---|
| 4 | GET | `/api/v1/master/departemen` | Authenticated/role-scoped | Department reference |
| 5 | GET | `/api/v1/master/jabatan` | Authenticated/role-scoped | Position reference |

### 4. Employee database — 8

| # | Method | Path | Access | Purpose |
|---:|---|---|---|---|
| 6 | GET | `/api/v1/karyawan` | HR, Top Management read-only | Paginated employee list |
| 7 | GET | `/api/v1/karyawan/{id}` | HR, Top Management read-only | Employee detail |
| 8 | POST | `/api/v1/karyawan` | HR | Create employee |
| 9 | PUT | `/api/v1/karyawan/{id}` | HR | Update employee per contract semantics |
| 10 | DELETE | `/api/v1/karyawan/{id}` | HR | Soft-deactivate employee |
| 11 | GET | `/api/v1/karyawan/export` | HR | Export approved employee data |
| 12 | POST | `/api/v1/karyawan/{id}/dokumen` | HR | Upload employee document |
| 13 | GET | `/api/v1/karyawan/{id}/dokumen` | HR; TM only if contract permits | List/read document metadata |

### 5. Own profile and metrics — 2

| # | Method | Path | Access | Purpose |
|---:|---|---|---|---|
| 14 | GET | `/api/v1/profil/saya` | Karyawan, Atasan, HR | Own read-only profile |
| 15 | GET | `/api/v1/profil/saya/metrik` | Karyawan, Atasan, HR | Own approved metrics/current salary |

Top Management has no Personal Metrics.

### 6. Dashboard — 1

| # | Method | Path | Access | Purpose |
|---:|---|---|---|---|
| 16 | GET | `/api/v1/dashboard/metrik` | HR, Top Management read-only | HR aggregate metrics/org chart payload |

### 7. Attendance — 2

| # | Method | Path | Access | Purpose |
|---:|---|---|---|---|
| 17 | POST | `/api/v1/absensi/checkin` | Karyawan, Atasan, HR | Attendance action defined by contract |
| 18 | GET | `/api/v1/absensi/livefeed` | HR, Top Management read-only | Global attendance live feed |

Requirements mention check-in and checkout, but the inventory lists only the `checkin`
operation. Verify the request/action semantics in API Contract before implementation; do not
create `/checkout` silently.

### 8. Attendance reports — 2

| # | Method | Path | Access | Purpose |
|---:|---|---|---|---|
| 19 | GET | `/api/v1/laporan/kehadiran` | HR, Top Management read-only | Filtered/paginated report |
| 20 | GET | `/api/v1/laporan/kehadiran/export` | HR | XLSX/PDF attendance export |

### 9. Leave/absence — 6

| # | Method | Path | Access | Purpose |
|---:|---|---|---|---|
| 21 | POST | `/api/v1/ketidakhadiran` | Karyawan, Atasan, HR | Submit own request |
| 22 | GET | `/api/v1/ketidakhadiran` | Active related approver | Approver inbox/list |
| 23 | GET | `/api/v1/ketidakhadiran/{id}` | Applicant/related approver | Scoped detail |
| 24 | GET | `/api/v1/ketidakhadiran/saya` | Karyawan, Atasan, HR | Own history |
| 25 | PUT | `/api/v1/ketidakhadiran/{id}/decision` | Active approver | Atomic approve/reject |
| 26 | PUT | `/api/v1/ketidakhadiran/{id}/delegate` | Active direct supervisor | Delegate to HR |

### 10. Master leave type — 3

| # | Method | Path | Access | Purpose |
|---:|---|---|---|---|
| 27 | GET | `/api/v1/master/jenis-izin` | HR per contract | List master leave types |
| 28 | POST | `/api/v1/master/jenis-izin` | HR | Create master leave type |
| 29 | PUT | `/api/v1/master/jenis-izin/{id}` | HR | Update master leave type |

### 11. Overtime — 5

| # | Method | Path | Access | Purpose |
|---:|---|---|---|---|
| 30 | POST | `/api/v1/lembur` | Karyawan, Atasan, HR | Submit own overtime request |
| 31 | GET | `/api/v1/lembur` | Active related approver | Approver list |
| 32 | GET | `/api/v1/lembur/{id}` | Applicant/related approver | Scoped detail |
| 33 | PUT | `/api/v1/lembur/{id}/decision` | Active approver | Atomic approve/reject |
| 34 | GET | `/api/v1/lembur/rekap` | HR | Overtime recap, no compensation calculation |

### 12. Access and audit — 4

| # | Method | Path | Access | Purpose |
|---:|---|---|---|---|
| 35 | GET | `/api/v1/akses/role` | HR, Top Management read-only | Role list |
| 36 | GET | `/api/v1/akses/permission` | HR, Top Management read-only | Permission catalog/mapping |
| 37 | PUT | `/api/v1/akses/permission` | HR | Update approved permission mapping |
| 38 | GET | `/api/v1/akses/audit-log` | HR, Top Management read-only | Filtered immutable audit log |

### 13. Notifications — 4

| # | Method | Path | Access | Purpose |
|---:|---|---|---|---|
| 39 | GET | `/api/v1/notifikasi` | Recipient only | Paginated notification list |
| 40 | GET | `/api/v1/notifikasi/unread-count` | Recipient only | Non-dismissed unread count |
| 41 | PUT | `/api/v1/notifikasi/{id}/read` | Owner recipient | Mark read idempotently |
| 42 | DELETE | `/api/v1/notifikasi/{id}` | Owner recipient | Soft-dismiss via `dismissed_at` |

## Operation-to-data map

| Module | Primary tables | External dependency |
|---|---|---|
| Auth | users, roles, employees, audit_logs | Redis |
| Organization | office_locations, departments, positions | — |
| Employee | employees and employee detail tables | Nextcloud for documents |
| Profile/dashboard | employees, salaries, attendance, leave, organization | — |
| Attendance/report | attendances, employees | Nextcloud for photos |
| Leave | leave_types, leave_balances, leave_requests, leave_approvals | Nextcloud for documents |
| Overtime | overtime_requests, overtime_approvals | Nextcloud optional document |
| Notifications | notifications plus source request/contract data | Worker/Redis lock |
| Access/audit | roles, permissions, users, audit_logs | Redis/cache invalidation |

## Database conventions

- Total implementation target: **26 explicitly named tables**.
- Primary keys: UUID generated with `gen_random_uuid()`.
- Foreign keys: `ON DELETE RESTRICT` by default.
- Timestamps: approved `created_at`/`updated_at`; use exact schema.
- Status/type: PostgreSQL enum or `CHECK` according to schema.
- Soft delete only where schema/requirements define it.
- Migrations are explicit/reviewable; no production auto-migrate.
- Never edit an already-applied migration.
- File bytes are never stored in PostgreSQL.
- Audit log is append-only.

## Table catalog

### Organization and account — 6

| # | Table | Purpose |
|---:|---|---|
| 1 | `office_locations` | Active office code/name/address and trusted WFO coordinates |
| 2 | `departments` | Organization departments |
| 3 | `positions` | Positions linked to organization structure |
| 4 | `roles` | Four stable role definitions |
| 5 | `employees` | Employee core, status, organization, direct supervisor |
| 6 | `users` | Authentication account linked to employee/role |

### Employee detail — 10

| # | Table | Purpose |
|---:|---|---|
| 7 | `employee_addresses` | Employee address information |
| 8 | `employee_ktp` | KTP identity details |
| 9 | `employee_contracts` | Employment contract and expiry data |
| 10 | `employee_bpjs` | BPJS information |
| 11 | `employee_npwp` | Tax/NPWP information |
| 12 | `employee_emergency_contacts` | Emergency contacts |
| 13 | `employee_education` | Education history |
| 14 | `employee_position_history` | Position/department movement history |
| 15 | `employee_salaries` | Salary by period |
| 16 | `employee_documents` | Nextcloud file metadata/locator |

### Attendance and leave — 5

| # | Table | Purpose |
|---:|---|---|
| 17 | `attendances` | Check-in/out, photo URL, mode, time, location |
| 18 | `leave_types` | Master leave/absence types |
| 19 | `leave_balances` | Annual user leave balances |
| 20 | `leave_requests` | Leave/absence request current state |
| 21 | `leave_approvals` | Immutable approval/delegation/escalation history |

### Overtime — 2

| # | Table | Purpose |
|---:|---|---|
| 22 | `overtime_requests` | Overtime request current state/duration |
| 23 | `overtime_approvals` | Overtime approval history |

### Access and audit — 2

| # | Table | Purpose |
|---:|---|---|
| 24 | `permissions` | Permission catalog/mapping data per schema |
| 25 | `audit_logs` | Append-only audit evidence |

### Notifications — 1

| # | Table | Purpose |
|---:|---|---|
| 26 | `notifications` | Recipient event, read and dismissed state |

Product decision D-013 resolves the source count mismatch by defining `office_locations` as
table 26. It has UUID `id`, unique `code`, `name`, nullable `address`, `latitude`, `longitude`,
`is_active`, `created_at`, and `updated_at`. Do not add permanent employee-office assignment.

## Relationship map

Conceptual relationships:

```text
office_locations -> attendances.office_location_id (WFO only)
departments -> positions
departments/positions -> employees
employees -> employees.atasan_id
roles -> users -> employees

employees -> employee_* detail/history/documents
employees/users -> attendances
users -> leave_balances
users/employees -> leave_requests -> leave_approvals
users/employees -> overtime_requests -> overtime_approvals
users -> notifications
users -> audit_logs.actor
roles <-> permissions (exact mapping table from schema)
```

Use exact foreign-key ownership and key names from Database Schema.

## Critical constraints and indexes

Known constraints:

- Employee NIP unique.
- User email unique.
- Relevant identity and contract numbers unique per schema.
- Employee direct supervisor nullable self-reference.
- Employee delete is inactive status plus `deleted_at`.
- Salary unique `(employee_id, periode)`.
- Leave balance unique `(user_id, tahun)`.
- Notification unique `(recipient_user_id, event_key)`.
- Audit log cannot be updated/deleted by application DB user.

Required index categories:

- Employee search/status/department/position/supervisor/deleted.
- Contract expiry scan.
- Attendance employee/date/open record/live-feed/report period.
- Leave/overtime applicant/status/active stage/approver/SLA.
- Notification recipient/dismissed/read/created.
- Audit time/actor/module/resource/action filters.

Transcribe exact index definitions from Database Schema v1.1.

## File and export contract

Upload:

- Multipart, not base64.
- Maximum 5 MB.
- Attendance photo JPG/PNG.
- Other document allowlist from contract.
- Backend validates size, MIME, extension, signature, ownership.
- Backend uploads to Nextcloud using server credentials.
- DB stores safe metadata/locator only.

Download/export:

- Authorization occurs before file response.
- HR employee/attendance export in approved XLSX/PDF.
- No watermark for authorized HR export.
- Safe filename/content type.
- Frontend revokes temporary object URLs.
- Do not expose Nextcloud credentials or permanent public URLs.

## Consistency boundaries

Use one DB transaction for:

- Multi-table employee create/update plus audit.
- Approval state/history plus durable event.
- Permission update plus audit.
- Other business writes requiring atomic state.

For Nextcloud + DB:

- Upload first only with compensation if DB fails, or use another approved durable pattern.
- Retention deletion plus URL clearing requires retry/reconciliation.

For notifications:

- Persist event/notification in the same transaction or durable outbox/handoff.
- Unique event key is final retry defense.

For concurrency:

- Attendance duplicate/open-state checks are atomic.
- Approval decision uses conditional update/row lock.
- Stale loser gets documented conflict.

## Contract gaps

Remaining decisions:

1. Exact file download/access mechanism.
2. Multi-office schema, coordinates, employee assignment, and WFO office-selection rule.
3. Exact formulas for dashboard joiners, resignations, turnover, leave, payroll cost, and
   organization chart; period/attendance/inactive/gender rules are resolved by D-015.

## Implementation checklist

- [ ] Exact active operation count remains 45.
- [ ] Exact table count/names reconcile with Database Schema.
- [ ] Payloads and columns are transcribed, not guessed.
- [ ] API and DB nullability/optionality match.
- [ ] Authorization uses `access-matrix.md`.
- [ ] State transitions use `workflows.md`.
- [ ] OpenAPI, DTOs, frontend schemas, mocks, and tests use one contract.
- [ ] Migrations include constraints/indexes/rollback.
- [ ] Files, events, audit, and concurrency consistency are explicit.
