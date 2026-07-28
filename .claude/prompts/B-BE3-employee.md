# Prompt — Epic B-BE.3: Employee Database, Profile & Dashboard

**Agent**: Backend  
**Branch**: `feat/be-employee-data`  
**Estimasi**: 3–4 hari  
**Prerequisite**: Epic B-BE.2 selesai dan authentication/RBAC sudah aktif

## Prompt untuk Claude Code

```text
Implementasikan modul Employee Database, Profil Saya, Metrik Personal, dan Dashboard
untuk backend GSNpeeps.

Sebelum mulai:
- Baca `CLAUDE.md` dan `docs/openapi.yaml`.
- Baca `.claude/skills/4.TASK_EMPLOYEE_DATA.md`.
- Baca API Contract v1.1 bagian Master Organisasi, Data Karyawan, Profil, dan Dashboard.
- Baca Database Schema v1.1 ERD-1 dan ERD-2.
- Baca PRD v1.2 untuk matriks akses dan pembatasan gaji/dokumen.
- Gunakan skill `.claude/skills/hris-backend/SKILL.md` beserta references.
- Pertahankan pola repository/service/handler dari epic sebelumnya.

Scope endpoint:

MASTER:
- GET `/api/v1/master/departemen`
- GET `/api/v1/master/jabatan`

EMPLOYEE:
- GET `/api/v1/karyawan`
- POST `/api/v1/karyawan`
- GET `/api/v1/karyawan/{id}`
- PUT `/api/v1/karyawan/{id}` sebagai partial update
- DELETE `/api/v1/karyawan/{id}` sebagai soft-delete
- GET `/api/v1/karyawan/export`
- POST `/api/v1/karyawan/{id}/dokumen`
- GET `/api/v1/karyawan/{id}/dokumen`

PROFILE:
- GET `/api/v1/profil/saya`
- GET `/api/v1/profil/saya/metrik`

DASHBOARD:
- GET `/api/v1/dashboard/metrik`

Non-scope:
- Self-service edit data pribadi.
- Histori gaji penuh di Profil Saya.
- Hiring Progress.
- Recruitment Cost.
- Benefit.
- Attendance, leave, overtime, dan notification implementation.

Kerjakan dengan urutan:

1. CONTRACT REVIEW

   - Cocokkan seluruh operation, query, request, response, enum, dan error terhadap
     API Contract v1.1.
   - Pastikan field CreateEmployeeRequest lengkap sesuai kontrak.
   - Pastikan UpdateEmployeeRequest seluruh field optional.
   - Jangan menambahkan PATCH bila kontrak menetapkan PUT.
   - Pastikan export memakai file stream XLSX/PDF, bukan JSON.
   - Catat gap pada `docs/openapi-decisions.md`; jangan membuat kontrak kedua.

2. MIGRATION DETAIL KARYAWAN

   Gunakan tabel organisasi/auth existing dan tambahkan:
   - `employee_addresses`
   - `employee_ktp`
   - `employee_contracts`
   - `employee_bpjs`
   - `employee_npwp`
   - `employee_emergency_contacts`
   - `employee_education`
   - `employee_position_history`
   - `employee_salaries`
   - `employee_documents`

   Wajib:
   - UUID, FK, UNIQUE, CHECK, timestamps, dan index sesuai Database Schema v1.1.
   - FK default `ON DELETE RESTRICT`.
   - Satu address/KTP/BPJS/NPWP per employee.
   - Banyak contract, emergency contact, education, position history, salary, document.
   - Salary unik per `(employee_id, periode)`.
   - Contract end-date memiliki index untuk scheduler H-30.
   - Tidak menyimpan binary file.
   - Migration up/down aman dan tidak memakai AutoMigrate.

3. DOMAIN, DTO, DAN VALIDATION

   Implementasikan entity/value object dan DTO untuk:
   - Department dan Position.
   - Employee summary/detail.
   - Address, KTP, contract, BPJS, NPWP.
   - Emergency contact, education, position history.
   - Current salary dan document.
   - Create/Update employee.
   - Profile dan personal metrics.
   - Dashboard metrics.

   Aturan:
   - JSON snake_case.
   - Tanggal memakai format `date`; timestamp memakai RFC 3339.
   - Email, NIP, nomor KTP, contract number, dan enum divalidasi.
   - Position harus berada pada department yang dipilih.
   - `atasan_id` tidak boleh menunjuk employee sendiri.
   - Atasan harus employee aktif.
   - Jangan bocorkan internal URL credential atau password_hash.

4. MASTER ORGANISASI

   Repository/service/handler:
   - List departemen aktif.
   - List jabatan dengan filter optional `department_id`.
   - Semua role terautentikasi boleh membaca untuk dropdown.
   - Tidak ada endpoint mutasi master organisasi pada API Contract.
   - Urutan output stabil dan testable.

5. CREATE EMPLOYEE

   Dalam satu transaction:
   - Validasi uniqueness NIP, email, dan nomor KTP.
   - Validasi department, position, dan atasan.
   - Insert employee dan seluruh nested data yang dikirim.
   - Buat user/account hanya bila flow tersebut dinyatakan dalam OpenAPI/decision;
     jangan menebak password awal.
   - Append Audit Log tanpa merekam data sensitif penuh.

   Mapping:
   - Validation -> 400 sesuai OpenAPI.
   - Duplicate -> 409 CONFLICT.
   - Bukan HR -> 403.

6. LIST DAN DETAIL EMPLOYEE

   List:
   - Filter `search`, `department_id`, `status`, `page`, `limit`.
   - Search hanya pada field kontrak, yaitu nama/NIP.
   - Gunakan query terparameterisasi dan bounded limit.
   - Exclude soft-deleted record secara default.
   - Hindari N+1 query.

   Detail:
   - Return Point 1–12 sesuai API Contract.
   - HR mendapat full authorized detail.
   - Top Management read-only.
   - Karyawan/Atasan mendapat 403 dan tidak boleh mengetahui existence record.

7. UPDATE EMPLOYEE

   - PUT bersifat partial sesuai kontrak.
   - Bedakan field tidak dikirim dan nilai kosong/null.
   - Validasi uniqueness hanya saat nilai berubah.
   - Update nested one-to-one dan collection secara eksplisit.
   - Jangan menghapus histori contract/position/salary secara tidak sengaja.
   - Catat before/after field yang aman di Audit Log.
   - Gunakan transaction dan optimistic/locking strategy bila repository menetapkannya.

8. SOFT-DELETE EMPLOYEE

   - Hanya HR.
   - Set `status='nonaktif'` dan `deleted_at`.
   - Jangan hard-delete child/history.
   - Invalidasi Redis session user terkait agar token langsung tidak berlaku.
   - Cegah employee nonaktif menjadi approver atau atasan baru.
   - Append Audit Log.
   - Operasi berulang mengikuti error/idempotency behavior OpenAPI.

9. NEXTCLOUD DOCUMENT STORAGE

   Upload:
   - Endpoint multipart.
   - Field `jenis_dokumen` dan `file`.
   - Format: PDF, JPG, PNG, DOC, DOCX, XLS, XLSX, PPT, PPTX.
   - Tolak ZIP/RAR.
   - Maksimum 5 MB.
   - Validasi extension, MIME, dan file signature.
   - Path harus namespaced per employee dan tidak menerima path dari client.
   - Upload melalui backend ke Nextcloud.
   - Simpan URL/path setelah upload berhasil.
   - Cleanup orphan file bila transaction DB gagal.

   Read:
   - List document metadata sesuai authorization.
   - Jangan expose technical WebDAV credential.
   - Ikuti kontrak URL/download yang telah disetujui.

10. EXPORT EMPLOYEE

    - Hanya HR sesuai API Contract.
    - Format `xlsx` dan `pdf`.
    - Filter sama dengan list; `id` optional untuk satu employee.
    - File tanpa watermark.
    - Streaming response dengan media type dan filename aman.
    - Formula injection protection untuk spreadsheet.
    - Jangan menulis export besar seluruhnya ke memory bila dapat di-stream.
    - Catat DOWNLOAD di Audit Log.
    - Filter tanpa hasil -> error sesuai OpenAPI.

11. PROFIL SAYA

    - Resolve employee dari authenticated user, bukan ID request.
    - Karyawan, Atasan, dan HR dapat membaca data sendiri.
    - Top Management tidak memiliki Metrik Personal.
    - Profil read-only.
    - Gaji hanya periode bulan berjalan.
    - Jangan return dokumen Point 12 bila PRD melarangnya pada role tersebut.
    - Histori gaji penuh tidak tersedia lewat endpoint ini.

12. METRIK PERSONAL

    Return:
    - Saldo cuti.
    - Lama kerja hari ini.
    - Riwayat jam check-in/check-out.

    Karena attendance/leave mungkin belum tersedia:
    - Jangan membuat data palsu.
    - Gunakan repository interface yang dapat diimplementasikan saat epic Attendance.
    - Jika endpoint belum dapat lengkap, implementasikan hanya setelah dependency tersedia
      atau gunakan empty state yang benar-benar ditetapkan kontrak.
    - Catat dependency, jangan mengubah response schema.

13. DASHBOARD METRIK

    Role:
    - HR full read.
    - Top Management read-only.
    - Karyawan/Atasan 403.

    Return agregasi:
    - Headcount.
    - Join bulan berjalan.
    - Resign bulan berjalan.
    - Turnover rate.
    - Leave.
    - Payroll cost bulan berjalan.
    - Gender ratio.
    - Department composition.
    - Org chart.

    Aturan:
    - Query periode `YYYY-MM`.
    - Definisi formula harus bersumber dari PRD/OpenAPI atau dicatat sebagai decision.
    - Jangan mengimplementasikan Hiring Progress dan Recruitment Cost.
    - Hindari query per employee.
    - Test timezone Asia/Jakarta dan boundary awal/akhir bulan.

14. AUTHORIZATION

    - Gunakan middleware/auth service dari BE.2.
    - HR: CRUD/read/export sesuai operation.
    - Top Management: GET/read-only saja.
    - Karyawan/Atasan: Profil Saya dan Metrik Personal sendiri.
    - Test setiap endpoint dengan seluruh role, termasuk direct URL bypass.
    - Forbidden request tidak boleh melakukan query sensitif, upload, mutation, atau audit
      seolah-olah operasi berhasil.

15. ERROR MAPPING

    Dokumentasikan dan test:
    - 400 VALIDATION_ERROR.
    - 401 UNAUTHORIZED.
    - 403 FORBIDDEN.
    - 404 NOT_FOUND.
    - 409 CONFLICT.
    - 413 FILE_TOO_LARGE.
    - 415 UNSUPPORTED_FILE_TYPE.
    - 429 TOO_MANY_REQUESTS.
    - 500 INTERNAL_ERROR.

    Jangan expose SQL, WebDAV response, stack trace, atau PII.

16. TEST

    Unit:
    - Create valid/invalid/duplicate.
    - Position-department mismatch.
    - Self supervisor.
    - Partial update semantics.
    - Soft-delete + session invalidation.
    - Current-month salary only.
    - Authorization matrix.
    - File type/size/path validation.
    - Dashboard formulas dan date boundaries.

    Integration:
    - Migration up/down.
    - Transaction rollback saat nested insert gagal.
    - List filters, search, pagination, soft-delete exclusion.
    - Detail tanpa N+1 yang tidak wajar.
    - Nextcloud upload success dan cleanup failure dengan fake/test server.
    - XLSX/PDF export media type dan spreadsheet injection.
    - Audit CREATE/UPDATE/DELETE/DOWNLOAD.

Quality gates:
1. Format, `go mod tidy`, `go vet ./...`, test, integration test, dan linter.
2. Build API dan worker.
3. Lint `docs/openapi.yaml`.
4. Jalankan migration up/down-one pada database test.
5. Jalankan `docker compose config`.
6. Scan secret, PII, debug code, dan orphan temporary file.
7. Verifikasi seluruh endpoint terhadap OpenAPI.

Git hanya bila diotorisasi:
- Branch: `feat/be-employee-data`
- Commit contoh:
  - `feat(employee): add employee detail migrations`
  - `feat(employee): implement employee CRUD and soft delete`
  - `feat(storage): add employee document upload`
  - `feat(profile): add self profile and personal metrics`
  - `feat(dashboard): add HR dashboard metrics`
  - `feat(export): add employee xlsx and pdf export`
- PR: `feat(employee): implement employee database and dashboard`
- Jangan push/open PR tanpa izin eksplisit.

Aturan akhir:
- Jangan menambah endpoint di luar OpenAPI.
- Jangan membuat self-service edit.
- Jangan hard-delete employee/history.
- Jangan menampilkan histori gaji penuh di Profil Saya.
- Jangan implementasikan modul Coming Soon.
- Jangan menyimpan file binary di PostgreSQL.
- Jangan memakai data personal nyata pada seed/test.
```

## Acceptance Criteria

- [ ] Migration 10 tabel detail karyawan sesuai Database Schema.
- [ ] Master departemen/jabatan read-only berfungsi.
- [ ] Employee CRUD, filter, pagination, detail, dan soft-delete sesuai OpenAPI.
- [ ] HR dapat CRUD/export; Top Management hanya read; role lain ditolak.
- [ ] Upload dokumen tervalidasi, maksimal 5 MB, dan disimpan melalui Nextcloud.
- [ ] Kegagalan DB setelah upload membersihkan orphan file.
- [ ] Profil Saya selalu mengambil identity sendiri dan read-only.
- [ ] Gaji Profil Saya hanya bulan berjalan.
- [ ] Dashboard metric terotorisasi dan formula terdokumentasi.
- [ ] Export XLSX/PDF tanpa watermark dan aman dari formula injection.
- [ ] Audit Log mencatat write dan download.
- [ ] Unit/integration/security tests serta seluruh quality gate lulus.

## Files yang Akan Dibuat atau Disesuaikan

```text
backend/
├── internal/
│   ├── domain/employee*.go
│   ├── dto/{employee,profile,dashboard}_*.go
│   ├── repository/{organization,employee,profile,dashboard}_repository.go
│   ├── service/{organization,employee,profile,dashboard}_service.go
│   ├── handler/{master,employee,profile,dashboard}_handler.go
│   ├── platform/nextcloud/
│   └── export/
├── migrations/*_create_employee_detail_tables.sql
├── tests/employee_integration_test.go
└── cmd/api/main.go

docs/openapi.yaml
docs/openapi-decisions.md
```

Ikuti naming dan lokasi final dari arsitektur existing; jangan membuat package paralel.
