# Prompt — Fase A: OpenAPI Spec Lengkap GSNpeeps

**Agent**: Backend  
**Branch**: `chore/openapi-gsnpeeps`  
**Estimasi**: 0.5–1 hari

## Prompt untuk Claude Code

```text
Tolong buatkan OpenAPI 3.1 specification lengkap untuk GSNpeeps.

Sebelum mulai:
- Baca `CLAUDE.md`.
- Baca `.claude/specs/document-index.md`.
- Baca API Contract v1.1 sebagai sumber kebenaran endpoint, payload, response, role, side effect, dan HTTP status.
- Baca Database Schema v1.1 untuk field entity, tipe data, enum, dan constraint.
- Baca PRD v1.2 dan User Story v1.2 untuk aturan role dan acceptance criteria.
- Gunakan skill `.claude/skills/hris-backend/SKILL.md`.
- Gunakan skill `.claude/skills/hris-git/SKILL.md` hanya untuk tahap Git.

Konteks:
- Nama produk: GSNpeeps.
- Cakupan: Employee Database, Profil Saya, Dashboard HR, Absensi, Ketidakhadiran, Lembur, Notifikasi, Akses, dan Audit Log.
- Backend: Go REST API.
- Base path: `/api/v1`.
- Format field JSON: `snake_case`.
- Auth: Bearer JWT dengan masa berlaku 8 jam dan session cross-check di Redis.
- Tidak ada refresh-token endpoint pada API Contract v1.1.
- Upload file memakai `multipart/form-data`, bukan base64.
- Maksimum ukuran file: 5 MB.
- Role: `karyawan`, `atasan`, `hr`, `top_management`.
- Pagination: `page` dan `limit`.
- Baseline PDF: 42 operasi dalam 13 modul; kontrak aktif 0.4.0: 46 operasi.

Format response sukses:

{
  "success": true,
  "data": {},
  "message": "Deskripsi singkat hasil"
}

Untuk endpoint list, tambahkan:

{
  "meta": {
    "page": 1,
    "limit": 20,
    "total_data": 134,
    "total_page": 7
  }
}

Format response error:

{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Pesan yang aman ditampilkan ke pengguna",
    "fields": {
      "field_name": "Pesan validasi"
    }
  }
}

`error.fields` hanya muncul pada validation error yang relevan.

Output:
- Buat file `docs/openapi.yaml`.
- Gunakan `openapi: 3.1.0`.
- Isi `info.title` dengan `GSNpeeps API`.
- Isi versi kontrak aktif dengan `0.4.0`.
- Definisikan server development dan server production dari konfigurasi, tanpa hardcode credential.
- Dokumentasikan seluruh 46 endpoint berikut.

PUBLIC/SYSTEM:
  GET    /health

AUTH:
  POST   /api/v1/auth/login
  POST   /api/v1/auth/reset-password
  POST   /api/v1/auth/logout
  GET    /api/v1/auth/me
  PATCH  /api/v1/auth/me/password

MASTER ORGANISASI:
  GET    /api/v1/master/departemen
  GET    /api/v1/master/jabatan
  GET    /api/v1/master/lokasi-kantor

DATA KARYAWAN:
  GET    /api/v1/karyawan
          Query: search, department_id, status, page, limit
          Role: HR dan Top Management read-only
  POST   /api/v1/karyawan
  GET    /api/v1/karyawan/{id}
          Role: HR dan Top Management read-only
  PUT    /api/v1/karyawan/{id}
          Partial update, hanya HR
  DELETE /api/v1/karyawan/{id}
          Soft-delete, hanya HR
  GET    /api/v1/karyawan/export
          Query: format=xlsx|pdf, id opsional, filter list
          Response berupa file stream, hanya HR
  POST   /api/v1/karyawan/{id}/dokumen
          multipart/form-data, hanya HR
  GET    /api/v1/karyawan/{id}/dokumen
          Role mengikuti API Contract

PROFIL SAYA DAN METRIK PERSONAL:
  GET    /api/v1/profil/saya
          Karyawan, Atasan, HR; data milik sendiri
  GET    /api/v1/profil/saya/metrik
          Karyawan, Atasan, HR; Top Management mendapat 403

DASHBOARD:
  GET    /api/v1/dashboard/metrik
          Query: periode=harian|mingguan|bulanan|tahunan, tanggal_acuan=YYYY-MM-DD
          HR dan Top Management read-only

ABSENSI — KEHADIRAN:
  POST   /api/v1/absensi/checkin
          multipart/form-data
          Field: tipe, mode_kerja, gps_lat, gps_long, foto
          Karyawan, Atasan, HR
  GET    /api/v1/absensi/livefeed
          Query: tanggal
          HR dan Top Management read-only

LAPORAN KEHADIRAN:
  GET    /api/v1/laporan/kehadiran
          Query: periode, tanggal_mulai, tanggal_selesai, department_id
          HR dan Top Management read-only
  GET    /api/v1/laporan/kehadiran/export
          Query laporan + format=xlsx|pdf
          Response berupa file stream, hanya HR

KETIDAKHADIRAN:
  POST   /api/v1/ketidakhadiran
          multipart/form-data
          Field: jenis_izin_id, tanggal_mulai, tanggal_selesai, alasan,
          dokumen_pendukung, lokasi_tujuan, keperluan_tugas
          Karyawan, Atasan, HR
  GET    /api/v1/ketidakhadiran
          Query: status, page, limit
          Approver terkait dengan row-level authorization
  GET    /api/v1/ketidakhadiran/{id}
          Pemohon atau approver terkait
  GET    /api/v1/ketidakhadiran/saya
          Query: page, limit
          Data milik sendiri
  PUT    /api/v1/ketidakhadiran/{id}/decision
          Atasan, HR, atau Top Management sesuai tahap aktif
  PUT    /api/v1/ketidakhadiran/{id}/delegate
          Atasan terkait

MASTER JENIS IZIN:
  GET    /api/v1/master/jenis-izin
          Query: aktif
          HR
  POST   /api/v1/master/jenis-izin
          HR
  PUT    /api/v1/master/jenis-izin/{id}
          HR

LEMBUR:
  POST   /api/v1/lembur
          multipart/form-data; dokumen pendukung opsional
          Karyawan, Atasan, HR
  GET    /api/v1/lembur
          Query: status, page, limit
          Approver terkait
  GET    /api/v1/lembur/{id}
          Pemohon atau approver terkait
  PUT    /api/v1/lembur/{id}/decision
          Approver sesuai tahap aktif
  GET    /api/v1/lembur/rekap
          Query: periode, department_id, karyawan_id, page, limit
          HR

AKSES DAN AUDIT:
  GET    /api/v1/akses/role
          HR dan Top Management read-only
  GET    /api/v1/akses/permission
          Query: role
          HR dan Top Management read-only
  PUT    /api/v1/akses/permission
          Hanya HR
  GET    /api/v1/akses/audit-log
          Query: tanggal_mulai, tanggal_selesai, user_id, page, limit
          HR dan Top Management read-only

NOTIFIKASI:
  GET    /api/v1/notifikasi
          Query: is_read, page, limit
          Recipient sendiri
  GET    /api/v1/notifikasi/unread-count
          Recipient sendiri
  PUT    /api/v1/notifikasi/{id}/read
          Recipient sendiri
  DELETE /api/v1/notifikasi/{id}
          Soft-dismiss permanen untuk recipient sendiri

Wajib gunakan tags:
- System
- Authentication
- Master Organization
- Employees
- My Profile
- Dashboard
- Attendance
- Attendance Reports
- Leave Requests
- Leave Types
- Overtime
- Access Administration
- Notifications

Wajib komponen reusable pada `components/schemas`:

RESPONSE:
- SuccessResponse
- ErrorResponse
- ErrorObject
- ValidationFields
- PaginationMeta
- DeleteSuccessResponse
- FileUploadResponse

AUTH:
- LoginRequest
- LoginResponse
- AuthUser

MASTER:
- Department
- Position
- LeaveType
- CreateLeaveTypeRequest
- UpdateLeaveTypeRequest

EMPLOYEE:
- EmployeeSummary
- EmployeeDetail
- EmployeeAddress
- EmployeeKTP
- EmployeeContract
- EmployeeBPJS
- EmployeeNPWP
- EmergencyContact
- EducationHistory
- PositionHistory
- CurrentSalary
- EmployeeDocument
- CreateEmployeeRequest
- UpdateEmployeeRequest
- EmployeeListResponse
- EmployeeDetailResponse

PROFILE/DASHBOARD:
- MyProfile
- PersonalMetrics
- ClockHistory
- DashboardMetrics
- GenderRatio
- DepartmentComposition

ATTENDANCE:
- Attendance
- AttendanceCheckRequest
- AttendanceCheckResponse
- AttendanceLiveFeedItem
- AttendanceReportItem

LEAVE:
- LeaveRequestSummary
- LeaveRequestDetail
- CreateLeaveRequest
- ApprovalHistory
- DecisionRequest
- DelegateRequest
- LeaveRequestListResponse

OVERTIME:
- OvertimeRequestSummary
- OvertimeRequestDetail
- CreateOvertimeRequest
- OvertimeRecapItem
- OvertimeRequestListResponse

ACCESS/AUDIT:
- RoleSummary
- Permission
- UpdatePermissionRequest
- AuditLog
- AuditLogListResponse

NOTIFICATION:
- Notification
- NotificationListResponse
- UnreadCountResponse

Gunakan nama schema yang konsisten. Jika nama di atas perlu disesuaikan supaya lebih tepat, jelaskan alasannya dan jangan mengubah bentuk kontrak.

Wajib komponen reusable pada `components/parameters`:
- PageParam
- LimitParam
- SearchParam
- EmployeeIdPath
- LeaveRequestIdPath
- OvertimeRequestIdPath
- NotificationIdPath
- DateFromParam
- DateToParam

Wajib komponen reusable pada `components/responses`:
- BadRequest
- Unauthorized
- Forbidden
- NotFound
- Conflict
- ValidationError
- FileTooLarge
- UnsupportedFileType
- TooManyRequests
- InternalError

Wajib komponen security scheme:
- `BearerAuth`
  - type: http
  - scheme: bearer
  - bearerFormat: JWT

Jangan membuat `RefreshCookie` karena API Contract GSNpeeps tidak memiliki refresh-token endpoint.

Error code yang harus didokumentasikan sesuai endpoint:

UMUM:
- 400 VALIDATION_ERROR
- 401 UNAUTHORIZED
- 403 FORBIDDEN
- 404 NOT_FOUND
- 413 FILE_TOO_LARGE
- 415 UNSUPPORTED_FILE_TYPE
- 429 TOO_MANY_REQUESTS
- 500 INTERNAL_ERROR

AUTH:
- 401 INVALID_CREDENTIALS
- 429 ACCOUNT_LOCKED

EMPLOYEE:
- 409 CONFLICT untuk email, NIP, atau nomor KTP yang sudah digunakan

ATTENDANCE:
- 422 OUT_OF_RADIUS
- 422 DUPLICATE_CHECKIN
- 422 CHECKOUT_WITHOUT_CHECKIN

LEAVE:
- 422 INSUFFICIENT_LEAVE_BALANCE
- 422 QUOTA_EXCEEDED
- 409 ALREADY_DECIDED

OVERTIME:
- 409 ALREADY_DECIDED

Aturan dokumentasi:
- Setiap operation wajib memiliki `operationId` yang unik.
- Setiap operation wajib memiliki summary, description, tags, security, parameters,
  requestBody bila ada, success response, dan seluruh error response yang relevan.
- Cantumkan role dan row-level authorization pada description operation.
- Cantumkan side effect notifikasi pada create, decision, delegate, dan escalation-related flow.
- Gunakan UUID format untuk seluruh ID.
- Gunakan `date`, `date-time`, `email`, dan format lain yang sesuai.
- Gunakan enum sesuai Database Schema dan API Contract.
- Status database, query, dan response memakai enum lowercase/underscore yang sama;
  label Bahasa Indonesia hanya tanggung jawab frontend.
- `PUT /karyawan/{id}` tetap didokumentasikan sebagai partial update sesuai kontrak;
  semua field pada update request harus optional.
- Endpoint file export wajib memakai media type XLSX/PDF, bukan JSON.
- Endpoint upload wajib memakai multipart schema dengan field binary.
- Tambahkan contoh request dan response yang sintetis, tanpa data personal nyata.
- Jangan menambah refresh token atau endpoint lain di luar revisi kontrak 0.4.0.
- Jika ditemukan selisih dokumen, catat pada `docs/openapi-decisions.md`; jangan menebak.

Setelah selesai:
1. Validasi YAML syntax.
2. Jalankan `redocly lint docs/openapi.yaml` jika Redocly tersedia.
3. Jika Redocly belum tersedia, gunakan validator OpenAPI 3.1 yang sudah ada di project.
4. Jangan menambah dependency validator secara global tanpa izin.
5. Tidak boleh ada lint error. Warning harus diperbaiki atau dijelaskan.
6. Verifikasi jumlah operation tepat 46.
7. Verifikasi seluruh `$ref` dapat di-resolve.
8. Laporkan validator, command, hasil, warning, dan keputusan kontrak.

Tahap Git hanya dilakukan jika pengguna meminta commit/push atau task aktif memang memberi otorisasi:
- Branch: `chore/openapi-gsnpeeps`
- Commit: `chore(openapi): add GSNpeeps API specification`
- Tag kandidat: `openapi-v0.1`
- Jangan push branch atau tag tanpa otorisasi eksplisit.
```

## Acceptance Criteria

- [ ] File `docs/openapi.yaml` menggunakan OpenAPI 3.1 dan valid.
- [ ] Tepat 46 operation GSNpeeps terdokumentasi.
- [ ] Seluruh request, response, query, path, upload, dan file stream memiliki schema.
- [ ] Error umum dan error domain didokumentasikan per endpoint.
- [ ] Security scheme `BearerAuth` tersedia dan diterapkan dengan benar.
- [ ] Empat role dan row-level authorization dicantumkan pada operation terkait.
- [ ] Komponen reusable menggunakan `$ref` dan seluruh referensi dapat di-resolve.
- [ ] Contoh data bersifat sintetis dan tidak mengandung PII nyata.
- [ ] Spec lulus lint tanpa error.
- [ ] Konflik/keputusan kontrak, jika ada, tercatat di `docs/openapi-decisions.md`.

## Catatan untuk Frontend Agent

Setelah Fase A disetujui, frontend agent **wajib** membaca `docs/openapi.yaml` sebagai sumber kebenaran untuk:

- Membuat validation schema yang mencerminkan request schema OpenAPI.
- Membuat mock handler yang mencerminkan response dan error OpenAPI.
- Membuat type/JSDoc DTO sesuai language frontend yang telah dipilih.
- Membuat API client dan query key tanpa mengubah kontrak.

Jangan membuat keputusan kontrak baru di frontend tanpa memperbarui dan memvalidasi OpenAPI terlebih dahulu.
