# Action Items — `missing-features.md`

Turns each line of `docs/testing/missing-features.md` into a concrete, independently
shippable action item (migration + backend + frontend + contract + tests), grounded in the
current code.

## Decisions

- **HR password reset**: HR types a new password in a form; backend applies it, unlocks the
  account, revokes all target sessions. Contract docs that currently forbid this are revised.
- **Master-data "delete"**: deactivate only (`is_active = false`). No hard delete — the FKs
  are `ON DELETE RESTRICT` and history must be preserved.
- **Timesheet "jam kantor"**: reuse the existing raw check-in→check-out span; show approved
  overtime as a separate column; add a combined total. No lunch-break deduction.

## Status pengerjaan (diperiksa 2 September 2026; kontrak + test dilengkapi 2 September 2026)

Legenda: `[x]` selesai, `[ ]` belum selesai, `[~]` sebagian dikerjakan namun belum siap
dirilis. Status di bawah didasarkan pada implementasi yang ada saat ini, bukan hanya rencana
pada dokumen ini.

| Item | Status | Catatan pengerjaan |
| --- | --- | --- |
| 1. Master Jenis Dokumen | `[x]` | List/tambah/edit/nonaktif di route, API, UI. OpenAPI 0.9.0 menambah `/master/jenis-dokumen` GET/POST + `/{id}` PUT/DELETE (D-041). Test: `tests/document_type_integration_test.go` (HR gate, 404, 409, audit CREATE/UPDATE/DELETE), `DocumentTypesPage.test.jsx` di-rework agar create/update/deactivate terpisah. |
| 2. Hapus kolom alpha | `[x]` | Alpha sudah dihapus dari domain, query laporan, export, schema frontend, OpenAPI, dan fixture. |
| 3. Reset password oleh HR | `[x]` | Endpoint, service aman, audit tanpa password, pencabutan seluruh sesi, dialog HR. OpenAPI 0.9.0 `POST /karyawan/{id}/reset-password` + revisi deskripsi `resetOwnPassword` (D-039); `CLAUDE.md` §7/§8 direvisi. Test: `employee_password_reset_test.go` (service), `employee_password_reset_handler_test.go` (422/400/403/404), `EmployeeDetailPage.test.jsx` (HR-only + submit + mismatch). |
| 4. Uraian pekerjaan absensi | `[x]` | Migrasi, validasi multipart, penyimpanan, respons API, form, OpenAPI, dan test form telah diperbarui. |
| 5. Total jam kantor/lembur | `[x]` | Query laporan menghitung jam kantor dan lembur disetujui, UI serta export menampilkan tiga total, dan alpha dihapus. |
| 6. Uraian di Live Feed | `[x]` | Uraian check-in/check-out mengalir ke Live Feed frontend dan export. |
| 7. Master lokasi kantor WFO | `[x]` | CRUD backend, audit, halaman HR, API frontend, route, navigasi. OpenAPI 0.9.0 `POST /master/lokasi-kantor` + `/{id}` PUT/DELETE (D-041). Test: `office_location_service_test.go` (HR gate + audit + 404), `office_location_handler_test.go` (koordinat invalid, 403/404), `OfficeLocationsPage.test.jsx`. |
| 8. Sesi login bersamaan | `[x]` | Redis per-token key, logout per-token, security event revoke-all. Keputusan D-040 di `docs/openapi-decisions.md`; `CLAUDE.md` §8 di-reword; deskripsi `POST /auth/logout` di OpenAPI direvisi. Test: `session_store_test.go` (skip tanpa `TEST_REDIS_URL`), `auth_service_test.go` (`fakeSessions` melacak set (user,fingerprint); login dua kali, logout per-token, revoke-all pada ganti/reset password + lockout). |

> Catatan: jangan menandai item `[x]` sebelum perubahan kode, kontrak OpenAPI, dan test yang
> disebut pada item tersebut selesai serta lolos verifikasi.

---

## Shared conventions (every backend item)

- Route helpers in `backend/internal/router/router.go`: `protected(fn)` = auth + rate limit
  only (role checked in service/handler); `guarded(module,action,fn)` = permission matrix
  (only `/media` and `/akses/*` use it). Master data uses `protected` + an in-code HR check.
- HR check idiom: `if identity.Role != domain.RoleHR { return domain.ErrForbidden }`
  (service) or `requireHR(w, r)` (`uat_handler.go:201`). `ErrForbidden` → 403 via
  `response.FromError`.
- Audit: service-layer pattern `s.tx.Within(ctx, func(txCtx){ <repo mutation>;
  return s.audit.Append(txCtx, domain.AuditEntry{...}) })` — see
  `backend/internal/service/leave_service.go:92-179`. `audit_logs` is append-only;
  `aksi VARCHAR(30)`, `modul VARCHAR(50)`.
- Next migration number is `00020`; goose format (`-- +goose Up` / `-- +goose Down`).
- `backend/internal/router/authorization_matrix_test.go` `contractOperations()` and
  `backend/internal/router/router_test.go` must list every new path+method.
- `docs/openapi.yaml` is the contract of record; update it for every request/response shape
  change (repo has OpenAPI conformance tests). Record decisions in `docs/openapi-decisions.md`
  (next id D-039) and bump the contract note in `CLAUDE.md` §7.

---

## Item 1 — Master Jenis Dokumen: edit + deactivate

**Status: `[~] Sebagian dikerjakan`**

**Catatan:** daftar dan tambah jenis dokumen sudah tersedia. Belum ada edit, deactivate/
reactivate, audit mutation, maupun cakupan kontrak dan test untuk operasi tersebut.

> Line 1: "belum ada edit delete di Master Jenis Dokumen, untuk mengubah dan menghapus nama Jenis Dokumen."

Today only `GET` + `POST /master/jenis-dokumen` exist (`uat_handler.go:210,231`).

**Backend**

- `backend/internal/router/router.go` (in the `if uat.Handler != nil` block, after line 131):
  add `PUT /master/jenis-dokumen/{id}` → `uat.Handler.UpdateDocumentType`,
  `DELETE /master/jenis-dokumen/{id}` → `uat.Handler.DeleteDocumentType`. Pattern to copy:
  `/company-feed/{id}` PUT/DELETE (lines 124-125), `/master/jenis-izin/{id}` PUT (line 151).
- `backend/internal/handler/uat_handler.go`:
  - `UpdateDocumentType`: `requireHR`; parse `{id}` with `uuid.Parse(mux.Vars(r)["id"])`;
    reuse `documentTypeInput` (line 194: `kode`, `nama`, `wajib`, `is_active *bool`);
    `UPDATE document_types SET kode=$1, nama=$2, wajib=$3, is_active=COALESCE($4,is_active),
    updated_at=NOW() WHERE id=$5`; `RowsAffected()==0` → 404 `NOT_FOUND`; pg `23505`
    (`pgconn.PgError`, already imported) → 409 `CONFLICT "Kode atau nama jenis dokumen sudah
    digunakan"`. Deactivate = this same endpoint with `is_active:false` (no separate route).
  - `DeleteDocumentType`: **not a hard delete** — `requireHR`; `UPDATE document_types SET
    is_active=false, updated_at=NOW() WHERE id=$1`; `RowsAffected()==0` → 404. Keeps
    `employee_documents.document_type_id` FK intact.
  - Wrap both mutations in an audit-writing tx like `withFeedAudit` (`uat_handler.go:454`),
    but that helper hard-codes `modul='company_feed'`. Add a `withUATAudit(w, r, module,
    action, mutate, errMsg)` variant (or parameterize `withFeedAudit`) writing
    `modul='master_jenis_dokumen'`, `aksi` in `CREATE|UPDATE|DELETE`. Also add audit to the
    existing `CreateDocumentType` for consistency (currently writes none).
- Optional: make `ListDocumentTypes` return `is_active` (it already selects it) and keep
  returning all rows so HR can see/reactivate deactivated types.

**Frontend** (`frontend/src/modules/uat/`)

- `api/uat-api.js`: add `updateDocumentTypeRequest(id, payload)` and
  `deleteDocumentTypeRequest(id)` — copy `updateFeedRequest`/`deleteFeedRequest` (lines 11-12).
- `pages/DocumentTypesPage.jsx`: add an edit affordance per row and a
  deactivate/reactivate button. Follow `frontend/src/modules/leave/pages/LeaveTypesPage.jsx`
  (edit-as-modal with a second `useForm`, `role="dialog" aria-modal="true"`, 409 → root
  error). Add `useMutation`s for update + delete and
  `queryClient.invalidateQueries({ queryKey: ["document-types"] })` on success. Reuse
  `components/data-table/DataTable`, `components/feedback/ConfirmDialog`,
  `components/ui/Button`, `useAuth()` for `isHR`.
- Route + nav already wired (`router.jsx` HR-only block; `navigation.js` "Master Data").

**Contract / tests**

- `docs/openapi.yaml`: add `put`/`delete` for `/master/jenis-dokumen/{id}`.
- `authorization_matrix_test.go` + `router_test.go`: add the two new operations (and the
  existing GET/POST if the matrix omits them).
- `frontend/src/modules/uat/tests/DocumentTypesPage.test.jsx`: current mock returns one
  shared `useMutation` stub — rework to mock the page's hooks so create/update/delete are
  distinguishable; add edit + deactivate cases.
- New backend handler test in `backend/internal/handler/uat_handler_test.go` for update/
  deactivate (HR gate, 404, 409).

---

## Item 2 — Remove the "alpha" column from Laporan Kehadiran

**Status: `[~] Sebagian dikerjakan`**

**Catatan:** UI tabel laporan sudah tidak menampilkan alpha. Field `alpha` masih dikirim dan
dipakai oleh backend, export, OpenAPI, schema frontend, dan fixture test; karena itu item
belum selesai.

> Line 2: "kolom alpha di report laporan kehadiran dihapus."

The report **table UI already has no alpha column** (`AttendanceReportPage.jsx` columns are
`nama, departemen, hadir, terlambat, izin`). Removal work is in the export + contract + schema:

- `backend/internal/service/attendance_service.go:570` — drop `"Alpha"` from `Headers`;
  `:580` — drop the `fmt.Sprintf("%d", item.Absent)` row cell. `attendanceReportTable` feeds
  both XLSX and PDF.
- `backend/internal/domain/attendance.go:122` — remove `Absent int \`json:"alpha"\`` from
  `AttendanceReportItem` (JSON field disappears from `GET /laporan/kehadiran`).
- `backend/internal/repository/attendance_repository.go:259-262` — the
  `item.Absent = workingDays - Present - Leave` computation and the `workingDays` plumbing
  (`workingDaysBetween`, `attendance_service.go:479`) become dead; remove them. Note
  `attendance_service_test.go:441 TestAttendanceReportPassesWorkingDayCount` asserts on the
  passed working-day count — delete/replace that test.
- `docs/openapi.yaml:2703` (`required: [... alpha]`) and `:2711` (`alpha` property) — remove.
- `frontend/src/modules/attendance/schemas/attendance-schema.js:49` — remove `alpha` from
  `attendanceReportItemSchema` (currently required; `.parse` throws otherwise).
- Test fixtures: `frontend/src/modules/attendance-reports/tests/AttendanceReportPage.test.jsx:48`
  (`alpha: 0`). Unrelated: `LeaveTypesPage.jsx:186` mentions "Alpha" as a leave concept —
  leave alone.

---

## Item 3 — HR-initiated password reset

**Status: `[ ] Belum dikerjakan`**

**Catatan:** aplikasi hanya mendukung reset password mandiri. Reset target karyawan oleh HR
belum mempunyai route, service, UI, audit, maupun perubahan kontrak.

> Line 3: "reset password dari role HR, jika user lupa password maka HR bisa melakukan reset password."

**Contract conflict to resolve first** — `CLAUDE.md` §7/§8, `docs/openapi-decisions.md` D-011,
`docs/openapi.yaml` (`resetOwnPassword` description), `docs/release/handover-signoff.md:90`
all state HR must never set another user's password. Revise these: new decision **D-039** in
`docs/openapi-decisions.md`, update `CLAUDE.md` §8 ("Login Lockout") and §7 contract note,
adjust the openapi description text. Chosen behavior: **HR types a new password**; account is
unlocked and all target sessions revoked; the value is never echoed back.

Placement: follow the `PUT /karyawan/{id}/foto` precedent — a service-guarded route under
`/karyawan/{id}`, consistent with every other `/karyawan` mutation (all use `protected` +
`identity.Role != domain.RoleHR`). `EmployeeService` already holds `sessionStore` and
`passwordHasher` (`backend/cmd/api/main.go:86-93`).

**Backend**

- Migration: none (no schema change — `users.password_hash`, `failed_login_count`,
  `account_locked` already exist).
- `backend/internal/router/router.go` (employee group): add
  `POST /karyawan/{id}/reset-password` → `employees.Handler.ResetEmployeePassword`
  (`protected`).
- `backend/internal/dto/employee.go` (or `dto/auth.go`): `ResetEmployeePasswordRequest{
  new_password: "required,min=12,max=128", new_password_confirmation:
  "required,min=12,max=128,eqfield=NewPassword" }`.
- `backend/internal/handler/employee_handler.go`: `ResetEmployeePassword` — decode, validate
  (422 `VALIDATION_ERROR`; mismatch → field error like self-reset), parse `{id}`, call
  service, return `response.Success` with `{ password_reset: true, account_unlocked: true,
  sessions_revoked: true }` (no password in the body).
- `backend/internal/repository/`: add a method to resolve `user_id` from `employee_id` and an
  admin `SetPassword(ctx, userID, hash)` that also sets `failed_login_count=0,
  account_locked=FALSE, updated_at=NOW()` (mirror `AuthRepository.UpdatePassword`, keyed by
  the resolved user id). Put it on `EmployeeRepository` (owns the `users` INSERT) or extend
  `AuthRepository` and inject it.
- `backend/internal/service/employee_service.go`: `ResetEmployeePassword(ctx, identity,
  employeeID, req)` — HR-only; resolve target user; `passwords.Hash(new)`; within a tx:
  `SetPassword` then `audit.Append({ Action:"PASSWORD_RESET" (<30 chars), Module:"karyawan",
  DataID:&userID, Detail:{field:"password", by:"hr", sessions_revoked:true, request_id} })`;
  after commit `sessionStore.Revoke(targetUserID)`. Never log the password. Refuse resetting
  your own account (self-service path exists); decide whether HR may reset `top_management`.

**Frontend** (`frontend/src/modules/employees/`)

- `api/employee-api.js`: `resetEmployeePasswordRequest(id, payload)` → `POST
  /karyawan/{id}/reset-password`.
- `hooks/useEmployees.js`: `useResetEmployeePassword(id)` mutation (invalidate
  `employeeKeys.detail(id)`).
- `pages/EmployeeDetailPage.jsx`: in the `isHR` header action cluster (lines 88-102), next to
  Edit / Nonaktifkan, add a "Reset Password" button opening a small form dialog
  (`new_password` + confirmation, React Hook Form + Zod, modal pattern from `LeaveTypesPage`).
  Show success, map 422 field errors, prevent double submit.
- Optional: surface `users.account_locked` on `EmployeeDetail` (backend
  `domain.EmployeeDetail` + `employee-schema.js`) so HR sees when a reset is needed — separate
  small sub-task.

**Contract / tests**

- `docs/openapi.yaml`: new `POST /karyawan/{id}/reset-password`; edit `resetOwnPassword`
  description. `docs/openapi-decisions.md` D-039. `CLAUDE.md` §7 note + §8.
- `authorization_matrix_test.go` + `router_test.go`: new operation (HR only; 403 for
  karyawan/atasan/top_management as decided).
- Backend: service test (HR gate, hashing, counter/lock cleared, session revoked, audit row
  has no password, self-reset refused); handler test (422 paths).
- Frontend: `EmployeeDetailPage` test — button visible only for HR, submit calls mutation,
  validation errors render.

---

## Item 4 — "Uraian pekerjaan" at clock-in / clock-out

**Status: `[ ] Belum dikerjakan`**

**Catatan:** belum ada kolom `uraian_pekerjaan` pada migrasi/model, input pada form Absensi,
atau pemrosesan API dan testnya.

> Line 4: "penambahan uraian pekerjaan ketika clock in / clock out pada halaman /app/absensi."

One nullable free-text column on `attendances` covers both events (check-in and check-out are
separate rows).

**Migration** `backend/migrations/00020_add_attendance_work_description.sql`

- Up: `ALTER TABLE attendances ADD COLUMN uraian_pekerjaan VARCHAR(500);`
- Down: `ALTER TABLE attendances DROP COLUMN uraian_pekerjaan;`
- (There is an unused `alamat VARCHAR(255)` column on the table — do not repurpose it.)

**Backend**

- `backend/internal/domain/attendance.go`: add `WorkDescription *string
  \`json:"uraian_pekerjaan"\`` to `Attendance` (62-75); add `WorkDescription string` to
  `RecordAttendance` (85-96) and `AttendanceRow` (99-112).
- `backend/internal/handler/attendance_handler.go` `Record` (71-187): read
  `strings.TrimSpace(request.FormValue("uraian_pekerjaan"))` near the other form values;
  pass into the `domain.RecordAttendance{}` literal (170-181). Recommend **required, max
  500** (add to the validation `fields` map).
- `backend/internal/service/attendance_service.go` `Record` (69-175): set
  `row.WorkDescription` when building `AttendanceRow` (86-97); add `uraian` to the audit
  detail map (156-161).
- `backend/internal/repository/attendance_repository.go` `Create` (103-130): add
  `uraian_pekerjaan` to the INSERT column list + `$13` arg; add to `RETURNING` + `Scan`.

**Frontend** (`frontend/src/modules/attendance/`)

- `schemas/attendance-schema.js`: add `uraian_pekerjaan: z.string().nullable().optional()` to
  `attendanceSchema` (20-33); `liveFeedItemSchema` inherits it (Item 6).
- `api/attendance-api.js` `recordAttendanceRequest` (17-29): `form.append("uraian_pekerjaan",
  input.uraian_pekerjaan)`.
- `pages/CheckInPage.jsx`: add `workDescription` state + a labelled `<textarea>` in the left
  card (used for both check-in and check-out); include it in the `record.mutateAsync({...})`
  payload (106-113); client-side required check via the existing `formError` pattern;
  optionally show in the result panel (316-343).
- `tests/CheckInPage.test.jsx`: add a case asserting the field is sent.

**Contract**

- `docs/openapi.yaml`: `Attendance` schema (L2638) + `AttendanceCheckRequest` (L2659) gain
  `uraian_pekerjaan`.

---

## Item 5 — Timesheet totals in Laporan Kehadiran (office vs overtime) + export

**Status: `[ ] Belum dikerjakan`**

**Catatan:** report belum memisahkan jam kantor dan jam lembur. Nilai
`total_jam_kerja` yang ada belum memenuhi definisi total gabungan pada item ini.

> Line 5: "tracking timesheet: nambahi di Laporan Kehadiran total jam kerja (office hour dan lembur dipisah) dan ditambahkan ke export report Laporan Kehadiran."

"Jam kantor" = existing raw span (`kehadiran.total_jam`). "Jam lembur" = `SUM(durasi_jam)` of
approved (`status='disetujui'`) `overtime_requests` in range. Add a combined total.

**Backend**

- `backend/internal/domain/attendance.go` `AttendanceReportItem` (115-124): rename/keep
  `TotalHours` as `OfficeHours \`json:"jam_kantor"\``; add `OvertimeHours
  \`json:"jam_lembur"\`` and `TotalWorkHours \`json:"total_jam_kerja"\``
  (= office + overtime; `total_jam_kerja` already exists in openapi L2712).
- `backend/internal/repository/attendance_repository.go` `Report` query (199-242): add a
  `lembur` CTE mirroring the `izin` CTE (216-227):

  ```sql
  lembur AS (
    SELECT u.employee_id, COALESCE(SUM(o.durasi_jam),0)::float8 AS jam_lembur
    FROM overtime_requests o JOIN users u ON u.id = o.user_id
    WHERE o.status = 'disetujui' AND o.tanggal BETWEEN $1::date AND $2::date
    GROUP BY u.employee_id
  )
  ```

  `LEFT JOIN lembur ON lembur.employee_id = e.id`; select `COALESCE(lembur.jam_lembur,0)`;
  extend `Scan` (252-257). Same aggregation pattern as `OvertimeRepository.Recap`
  (`overtime_repository.go:244`) and `PersonalMetrics` (`attendance_repository.go:311`).
  Compute `TotalWorkHours = OfficeHours + OvertimeHours` in Go.
- `backend/internal/service/attendance_service.go` `attendanceReportTable` (561-585): after
  removing `"Alpha"` (Item 2), headers become
  `["Nama","Departemen","Hadir","Terlambat","Izin","Jam Kantor","Jam Lembur","Total Jam Kerja"]`;
  row cells `fmt.Sprintf("%.2f", ...)` for the three hour fields.

**Frontend** (`frontend/src/modules/attendance-reports/`)

- `pages/AttendanceReportPage.jsx` `columns` (71-77): add `jam_kantor`, `jam_lembur`,
  `total_jam_kerja` columns using the already-imported `formatNumber`.
- `frontend/src/modules/attendance/schemas/attendance-schema.js` `attendanceReportItemSchema`
  (42-51): add `jam_kantor`, `jam_lembur` numbers; keep `total_jam_kerja`.
- Export XLSX/PDF content is 100% server-built — no `ReportExportMenu` change needed.
- Test fixtures: `AttendanceReportPage.test.jsx` `reportRow` (42-49) gains the hour fields.

**Contract**

- `docs/openapi.yaml` `AttendanceReportItem` (2701-2712): drop `alpha` (Item 2), add
  `jam_kantor`, `jam_lembur`; keep `total_jam_kerja`.

**Backend tests**: repository test for the `lembur` CTE aggregation (approved only, range
bounded); service test for the export header/row set.

---

## Item 6 — "Uraian pekerjaan" in Live Feed Absensi + its export

**Status: `[ ] Belum dikerjakan`**

**Catatan:** karena Item 4 belum tersedia, uraian belum dapat dipilih, disimpan, ditampilkan,
atau diekspor dari Live Feed.

> Line 6: "uraian pekerjaan ditampilkan di Live Feed Absensi termasuk report Live Feed."

Depends on Item 4's `attendances.uraian_pekerjaan` column + domain field.

**Backend**

- `backend/internal/repository/attendance_repository.go` `LiveFeed` query (133-168): add
  `a.uraian_pekerjaan` to the SELECT and to the `Scan` (157-162). `AttendanceLiveFeedItem`
  embeds `Attendance`, so the field from Item 4 flows through.
- `backend/internal/service/attendance_service.go`:
  - `liveFeedRow` struct (325-331) already carries the whole `CheckIn`/`CheckOut` item — no
    change, or add explicit `CheckInDesc`/`CheckOutDesc` for clarity.
  - `liveFeedTable` (365-392): add `"Uraian Masuk"` + `"Uraian Pulang"` headers (369) and row
    cells from `row.CheckIn.WorkDescription` / `row.CheckOut.WorkDescription` (381-389).

**Frontend** (`frontend/src/modules/attendance-reports/pages/LiveFeedPage.jsx`)

- `groupByEmployeeDay` (21-43) already stores the full item on `checkIn`/`checkOut` — no
  change needed to carry `uraian_pekerjaan`.
- `columns` (90-144): add a column rendering `row.checkIn?.uraian_pekerjaan` /
  `row.checkOut?.uraian_pekerjaan`.
- Export XLSX content is server-built — no `ExportButton` change.
- `liveFeedItemSchema` already inherits the field from `attendanceSchema` (Item 4).

**Contract**

- `docs/openapi.yaml` `AttendanceLiveFeedItem` (L2688): add `uraian_pekerjaan` (or note it
  inherits from `Attendance`).

**Tests**: livefeed repository test (field selected), service test (export headers include
the two columns), `LiveFeedPage` component test (column renders).

---

## Item 7 — Office location (WFO) master: create + edit + deactivate

**Status: `[~] Sebagian dikerjakan`**

**Catatan:** daftar lokasi kantor aktif (`GET /master/lokasi-kantor`) sudah dipakai saat WFO.
Fitur tambah, ubah, deactivate/reactivate untuk HR beserta route, UI, audit, kontrak, dan test
belum ada.

> Line 7: "fitur untuk tambah, hapus, edit lokasi kantor WFO."

Today only `GET /master/lokasi-kantor` exists (`attendance_handler.go:47`,
`ListActiveOfficeLocations` in `attendance_repository.go:53`). No admin UI. No `radius`
column — WFO radius is the global constant `domain.OfficeRadiusMeters` (100); keep it global.

**Migration**: none — `office_locations` already has
`id, kode (UNIQUE), nama, alamat (nullable), latitude, longitude, is_active, created_at,
updated_at` with lat/long CHECK constraints (`00005_...:3-15`).

**Backend**

- `backend/internal/router/router.go` (employee/master group, near line 102): add
  `POST /master/lokasi-kantor`, `PUT /master/lokasi-kantor/{id}`,
  `DELETE /master/lokasi-kantor/{id}` → new `attendances.Handler` methods (`protected`).
- `backend/internal/dto/`: `OfficeLocationRequest` with validator tags (`kode` required/max
  50, `nama` required/max 150, `alamat` omitempty/max 255, `latitude`
  required/min=-90/max=90, `longitude` required/min=-180/max=180, `is_active`).
- `backend/internal/repository/attendance_repository.go`: add `CreateOfficeLocation`,
  `UpdateOfficeLocation` (dynamic `SET`, `RowsAffected()==0` → `ErrNotFound`),
  `DeactivateOfficeLocation` (`UPDATE office_locations SET is_active=false, updated_at=NOW()
  WHERE id=$1`). Map pg `23505` (dup `kode`) / `23514` (lat-long CHECK) →
  `repository.ErrConflict` (reuse `mapEmployeeMutationError`, `employee_repository.go:530`).
  **No hard delete** (FK `attendances.office_location_id` is `ON DELETE RESTRICT`; deactivate
  removes it from the WFO dropdown and blocks new check-ins via the existing `is_active =
  TRUE` filter while history stays).
- `backend/internal/service/attendance_service.go`: add `CreateOfficeLocation` /
  `UpdateOfficeLocation` / `DeactivateOfficeLocation` — HR-only
  (`identity.Role != domain.RoleHR` → `ErrForbidden`); add the HR gate only on the mutations,
  not on `ListOfficeLocations` (serves every role's dropdown). Wrap each in `s.tx.Within` +
  `s.audit.Append({ Module: "master_lokasi_kantor", Action: CREATE|UPDATE|DELETE, DataID:&id,
  Detail:{...,request_id} })` — the service already has `s.tx` and `s.audit`. Add the methods
  to the handler's `AttendanceService` interface (`attendance_handler.go:19`).
- `backend/internal/handler/attendance_handler.go`: `CreateOfficeLocation` /
  `UpdateOfficeLocation` / `DeactivateOfficeLocation` — HR check, decode+validate,
  `h.requestMeta(request)` for IP/request-id, map errors via `response.FromError`,
  `response.Success` (201 for create).
- Behavioral note to document: editing lat/long changes which future WFO check-ins pass the
  100 m test; deactivating a location makes new WFO check-ins against it fail with
  `ErrInvalidOfficeLocation`. History is unaffected.

**Frontend**

- New page `frontend/src/modules/attendance/pages/OfficeLocationsPage.jsx` (HR admin),
  modeled on `LeaveTypesPage.jsx` (DataTable + create form + edit modal + deactivate/
  reactivate). Fields: kode, nama, alamat, latitude, longitude, is_active.
- `frontend/src/modules/attendance/api/attendance-api.js`: add
  `createOfficeLocationRequest`, `updateOfficeLocationRequest(id, payload)`,
  `deactivateOfficeLocationRequest(id)`.
- `frontend/src/modules/attendance/schemas/attendance-schema.js`: add an
  `officeLocationInputSchema` (lat/long as numbers with range refinement); keep
  `officeLocationSchema` for reads.
- `frontend/src/modules/attendance/hooks/useAttendance.js`: add mutation hooks; on success
  invalidate `attendanceKeys.offices` (`["organization","office-locations"]`) so
  `CheckInPage`'s dropdown refreshes.
- `frontend/src/routes/router.jsx`: import `OfficeLocationsPage`; add
  `{ path: "master/lokasi-kantor", element: <OfficeLocationsPage /> }` inside the HR-only
  nested `RoleRoute` block next to the other master routes.
- `frontend/src/routes/navigation/navigation.js`: add
  `{ label: "Master Lokasi Kantor", path: "/app/master/lokasi-kantor", roles: hrOnly }` to
  the "Master Data" group; update `navigation.test.js`.

**Contract / tests**

- `docs/openapi.yaml`: add `post /master/lokasi-kantor` and `put`/`delete`
  `/master/lokasi-kantor/{id}`.
- `authorization_matrix_test.go` + `router_test.go`: add the three operations (HR only).
- Backend: repository test (create/update/deactivate, dup `kode` → conflict, deactivated
  location excluded from `ListActiveOfficeLocations` and rejected by `Record` WFO path);
  service test (HR gate + audit).
- Frontend: `OfficeLocationsPage` component test; `CheckInPage` test still passes.

---

## Item 8 — Concurrent sessions: keep the previous token valid after a new login

**Status: `[ ] Belum dikerjakan`**

**Catatan:** session store masih menggunakan satu key `session:<user_id>`. Login kedua
menimpa sesi pertama, dan logout masih mencabut semua sesi pengguna.

> Not in `missing-features.md`; raised separately. Today a second login **overwrites** the
> single Redis key `session:<user_id>`, so the earlier token immediately starts failing
> `sessions.Validate` → 401 `INVALID_TOKEN`. Goal: multiple concurrently-issued tokens for
> the same user all stay valid until their own expiry, an explicit logout, or a security
> revoke.

Current mechanism (reference):

- `backend/internal/platform/redis/session_store.go` — `Save` = `SET session:<user_id>
  <fingerprint> EX ttl` (one key per user; overwrite); `Validate` = `GET` + constant-time
  compare; `Revoke` = `DEL session:<user_id>`.
- `backend/internal/platform/token/jwt.go:36-56` — each JWT gets a random `jti`;
  `fingerprint = sha256(jti)`.
- `backend/internal/middleware/authentication.go:29-37` — `Verify` → fingerprint →
  `sessions.Validate(userID, fingerprint)`; the fingerprint is discarded afterwards.
- `backend/internal/service/auth_service.go` — `Login` calls `Save` (line 154); `Logout`
  (~line 191) and `replacePassword` (change-password + self-reset) call `Revoke(userID)`;
  lockout on the 5th failed login calls `Revoke(userID)` (line 134).
- `backend/internal/service/employee_service.go` — `Deactivate` and `Update` (email/role
  change) call the injected `SessionRevoker.Revoke(userID)`.

**Decision needed — logout scope**: with multi-session, `POST /auth/logout` should revoke
**only the calling token** (recommended), not every device. That means threading the
fingerprint from the auth middleware into the request context and into the logout
handler/service. Security revokes (lockout, change-password, self-reset, HR reset from
Item 3, employee deactivate, email/role change) must still revoke **all** of the user's
sessions.

**Backend**

- Migration: none (Redis only).
- `backend/internal/platform/redis/session_store.go` — one key per token instead of one per
  user:
  - `Save(ctx, userID, fingerprint, ttl)` → `SET session:<user_id>:<fingerprint> "1"
    EX <ttl>`. Each key self-expires with its own 8h JWT TTL, so stale entries clean
    themselves.
  - `Validate(ctx, userID, fingerprint)` → `EXISTS session:<user_id>:<fingerprint>` (the key
    name is the secret-derived fingerprint, so the value compare is no longer needed).
  - `Revoke(ctx, userID)` (revoke-all) → loop `SCAN MATCH session:<user_id>:* COUNT 100` +
    `UNLINK`/`DEL`. Do not use `KEYS`.
  - Add `RevokeToken(ctx, userID, fingerprint)` → `DEL session:<user_id>:<fingerprint>` for
    single-device logout.
  - Optional device cap: companion `ZSET session:<user_id>` of `fingerprint → issuedAt`; on
    `Save`, `ZADD` then trim entries beyond N (e.g. 5) and `DEL` their keys. Skip unless a
    cap is actually wanted.
- Interfaces to update: `SessionStore` (in `auth_service.go`) and `SessionRevoker` (in
  `employee_service.go`) gain `RevokeToken`; `middleware.SessionValidator` signature is
  unchanged.
- `backend/internal/middleware/authentication.go` — after a successful `Validate`, stash the
  fingerprint in context (`WithSessionFingerprint` / `SessionFingerprintFromContext`,
  mirroring `WithIdentity`).
- `backend/internal/service/auth_service.go`:
  - `Login` — no logic change; `Save` now writes a per-token key instead of overwriting, so
    earlier tokens keep working.
  - `Logout` — take the current fingerprint (from context, via the handler) and call
    `RevokeToken(userID, fingerprint)` instead of `Revoke(userID)`. Keep the
    `context.WithoutCancel` + timeout wrapper and the `"LOGOUT"` audit.
  - `replacePassword` and the lockout path (line 134) — keep `Revoke(userID)` (revoke-all);
    adjust the `sessions_revoked` wording/semantics.
- `backend/internal/handler/auth_handler.go` `Logout` — read the fingerprint from context
  and pass it to `service.Logout`.
- `backend/internal/service/employee_service.go` — `Deactivate` / `Update` keep calling
  `Revoke` (revoke-all); only the interface changes.
- Item 3 (HR password reset) — its "revoke all target sessions" step stays correct (it uses
  revoke-all).

**Frontend**

- No functional change. Optionally adjust login copy to note that signing in on a new device
  no longer signs you out elsewhere.
- A "logout everywhere" affordance, if wanted later, is a new endpoint
  (`POST /auth/logout-all`) — out of scope here.

**Contract / docs**

- `CLAUDE.md` §8 "JWT dan Session" — reword "Token aktif di-cross-check melalui Redis key
  `session:<user_id>`" to the per-token key scheme; state that concurrent sessions are
  allowed and which events revoke all vs one.
- `docs/openapi-decisions.md` — new decision **D-040** (D-039 is taken by Item 3): concurrent
  sessions allowed; logout is per-token; security events revoke all.
- `docs/openapi.yaml` — `POST /auth/logout` description: "mengakhiri sesi perangkat saat ini"
  (not all devices). Update `docs/release/handover-signoff.md` if it asserts single-session.

**Tests**

- `backend/internal/platform/redis/session_store_test.go` (miniredis / real Redis): two
  `Save` for one user → both `Validate` pass; `RevokeToken` → only that one fails; `Revoke`
  → all fail; each key has a TTL.
- `backend/internal/service/auth_service_test.go` — `fakeSessions` (currently a no-op `Save`
  at line 109) must track a set of `(userID, fingerprint)`; add: login twice → first token
  still validates; logout revokes only the current fingerprint; change-password / self-reset
  / 5th failed login revoke all.
- `backend/internal/middleware` — auth test asserting the fingerprint reaches context.
- Invert any test that asserts "second login invalidates the first" (grep found none in
  `auth_service_test.go`).

**Risks**

- Redis key growth — bounded by the per-key 8h TTL and, if adopted, the optional device cap.
- `SCAN`-based revoke-all is O(sessions per user) but runs only on security events, never on
  hot paths.

---

## Suggested sequencing

1. **Item 4** (migration + attendance write path) — unblocks Item 6.
2. **Item 2** (remove alpha) then **Item 5** (timesheet columns) — both touch
   `attendanceReportTable` / `AttendanceReportItem` / the report query; do together.
3. **Item 6** (live feed uraian).
4. **Item 1** (jenis dokumen) and **Item 7** (office location) — parallel, independent, same
   master-data pattern.
5. **Item 3** (HR password reset) — includes the contract-doc revision (D-039).
6. **Item 8** (concurrent sessions) — independent; touches the Redis session store, the auth
   middleware, and `Logout`. Do alongside or after Item 3, since both revise the session
   contract (Item 3's revoke-all path is unaffected by the change).

One Conventional Commit per item, e.g.
`feat(attendance): capture uraian pekerjaan on check-in/out`,
`feat(report): split office and overtime hours in laporan kehadiran`,
`feat(auth): allow HR to reset an employee password`.

---

## Verification

Per touched area (`CLAUDE.md` §13):

- Backend: `cd backend && make fmt && make vet && make lint && make test && make build`.
  Migration up/down: `make goose up` then `make goose down` on `00020`.
- Frontend: `cd frontend && pnpm run lint && pnpm run test && pnpm run build`.
- Root: `docker compose config`.
- OpenAPI conformance / `authorization_matrix_test.go` / `router_test.go` must pass after
  every route addition.

End-to-end smoke (`docker compose up`):

- **Item 1**: as HR, create a jenis dokumen, rename it, deactivate it; confirm it disappears
  from the active list and the employee-document upload picker; non-HR gets 403.
- **Item 2**: `GET /laporan/kehadiran` has no `alpha`; XLSX export has no Alpha column.
- **Item 3**: as HR, reset an employee's password on the detail page; confirm the employee's
  existing token is rejected, the new password logs in, the lockout counter is cleared, and
  the audit row contains no password value; non-HR cannot call the endpoint.
- **Item 4**: check-in and check-out on `/app/absensi` with an uraian; confirm it persists on
  both `attendances` rows and round-trips in the check-in response.
- **Item 5**: `GET /laporan/kehadiran` shows `jam_kantor`, `jam_lembur`, `total_jam_kerja`;
  create an approved overtime request in-range and confirm `jam_lembur` reflects it and
  `total_jam_kerja = jam_kantor + jam_lembur`; XLSX export has the three columns.
- **Item 6**: the uraian text shows in Live Feed for both check-in and check-out rows and in
  the Live Feed XLSX export.
- **Item 7**: as HR, add/edit/deactivate an office location; confirm the WFO dropdown on
  `/app/absensi` updates, a WFO check-in against an active location within 100 m succeeds, and
  a check-in against a deactivated location is rejected; non-HR cannot mutate.
- **Item 8**: log in as one user on client A, then log in again as the same user on client B;
  confirm A's token still returns 200 on `GET /auth/me`. Log out on B → A still works. Then
  self-reset the password → both A and B get 401.
