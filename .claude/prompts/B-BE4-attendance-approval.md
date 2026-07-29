# Prompt — Epic B-BE.4: Attendance, Leave & Overtime Approval

**Agent**: Backend  
**Branch**: `feat/be-attendance-approval`  
**Estimasi**: 4–6 hari  
**Prerequisite**: Epic B-BE.3 selesai; employee, organization, auth, dan storage tersedia

## Prompt untuk Claude Code

```text
Implementasikan backend Kehadiran, Laporan Kehadiran, Ketidakhadiran, Master Jenis
Izin, Lembur, dan workflow approval GSNpeeps.

Sebelum mulai:
- Baca `CLAUDE.md`, `docs/openapi.yaml`, dan `docs/openapi-decisions.md`.
- Baca `.claude/skills/5.TASK_ATTENDANCE.md` dan `6.TASK_APPROVALS.md`.
- Baca API Contract v1.1 bagian 8–12.
- Baca Database Schema v1.1 ERD-3, ERD-4, dan scheduler section.
- Baca Sequence Diagram v1.1 untuk urutan decision/delegation/escalation.
- Baca `.claude/specs/workflows.md`.
- Gunakan skill `hris-backend` dan pola existing.

Scope endpoint:
- POST `/api/v1/absensi/checkin`
- GET `/api/v1/absensi/livefeed`
- GET `/api/v1/laporan/kehadiran`
- GET `/api/v1/laporan/kehadiran/export`
- POST `/api/v1/ketidakhadiran`
- GET `/api/v1/ketidakhadiran`
- GET `/api/v1/ketidakhadiran/{id}`
- GET `/api/v1/ketidakhadiran/saya`
- PUT `/api/v1/ketidakhadiran/{id}/decision`
- PUT `/api/v1/ketidakhadiran/{id}/delegate`
- GET `/api/v1/master/jenis-izin`
- POST `/api/v1/master/jenis-izin`
- PUT `/api/v1/master/jenis-izin/{id}`
- POST `/api/v1/lembur`
- GET `/api/v1/lembur`
- GET `/api/v1/lembur/{id}`
- PUT `/api/v1/lembur/{id}/decision`
- GET `/api/v1/lembur/rekap`

Non-scope:
- Reminder absensi.
- Kompensasi/uang lembur.
- Approval budget Perjalanan Dinas.
- Modul Benefit.
- Notification persistence/UI; siapkan event contract untuk BE.5.

Kerjakan per vertical slice: Kehadiran -> Laporan -> Ketidakhadiran -> Lembur ->
Scheduler. Selesaikan test slice sebelum berpindah.

1. CONTRACT DAN MIGRATION

   Tambahkan tabel:
   - `attendances`
   - `leave_types`
   - `leave_balances`
   - `leave_requests`
   - `leave_approvals`
   - `overtime_requests`
   - `overtime_approvals`

   Ikuti seluruh tipe, enum, FK, generated field, UNIQUE, dan index Database Schema v1.1.
   Wajib:
   - UUID.
   - Attendance index `(user_id, tanggal)`.
   - Leave balance unik `(user_id, tahun)`.
   - Request status terbatas pada state kontrak.
   - Approval history menyimpan tahap dan keputusan.
   - `approver_id` nullable untuk auto-escalation sistem.
   - Migration up/down aman; tanpa AutoMigrate.

2. ATTENDANCE CHECK-IN/CHECK-OUT

   Request multipart:
   - `tipe`: check_in/check_out.
   - `mode_kerja`: WFO/WFH/WFA.
   - `gps_lat`, `gps_long`.
   - `foto`: JPG/PNG maksimal 5 MB.

   Aturan:
   - Identity selalu dari JWT.
   - `waktu_network` berasal dari server dan menjadi sumber kebenaran.
   - `waktu_local` mengikuti kontrak; jangan menambah request field tanpa update OpenAPI.
   - Koordinat wajib untuk WFO, WFH, dan WFA.
   - WFO wajib mengirim `office_location_id` dan berada maksimal 100 meter dari koordinat
     tepercaya lokasi kantor aktif tersebut.
   - Karyawan bebas memilih kantor aktif; tidak ada assignment kantor permanen.
   - Data alamat/koordinat resmi akan di-seed kemudian; jangan membuat lokasi fiktif.
   - WFH/WFA tidak menjalankan radius rejection.
   - Hari kerja reguler Senin-Jumat zona `Asia/Jakarta`; akhir pekan ditolak dengan
     `422 NON_WORKING_DAY`.
   - Check-in tepat 09:00 tidak terlambat; setelah 09:00 berstatus `terlambat`.
   - Checkout sebelum 18:00 tetap valid dan dicatat sebagai `pulang_cepat`.
   - Gunakan formula geodesic yang teruji dan dokumentasikan unit/precision.
   - Cegah check-in kedua pada tanggal kerja yang sama.
   - Cegah checkout tanpa check-in.
   - Check-in tepat 09:00:00 WIB belum telat; setelah 09:00:00 WIB berstatus terlambat.
   - Reverse geocoding harus berada di balik interface, timeout, dan tidak menjadi
     alasan menyimpan credential di frontend.
   - Upload foto melalui Nextcloud path yang dibuat server.
   - Cleanup file bila transaction DB gagal.
   - Append Audit Log.

   Error:
   - 400 VALIDATION_ERROR.
   - 413 FILE_TOO_LARGE.
   - 415 UNSUPPORTED_FILE_TYPE.
   - 422 OUT_OF_RADIUS.
   - 422 DUPLICATE_CHECKIN.
   - 422 CHECKOUT_WITHOUT_CHECKIN.

3. WATERMARK DAN FALLBACK

   - Kamera live dan fallback upload menggunakan kontrak endpoint yang sama.
   - Jangan mempercayai timestamp pada image sebagai sumber waktu attendance.
   - Jika backend menghasilkan/menormalisasi watermark, gunakan waktu server dan metadata
     yang disetujui serta hapus metadata sensitif yang tidak diperlukan.
   - Jika watermark adalah tanggung jawab frontend, backend tetap memvalidasi file dan
     mencatat waktu server.
   - Catat ownership watermark di OpenAPI decision log; jangan membuat field baru diam-diam.

4. LIVE FEED

   - HR full read dan Top Management read-only.
   - Karyawan/Atasan 403.
   - Query tanggal default hari ini dalam timezone Asia/Jakarta.
   - Gabungkan kehadiran dan status ketidakhadiran sesuai response contract.
   - Hindari N+1.
   - Jangan menampilkan foto/record employee nonaktif di luar policy retensi.

5. LAPORAN DAN EXPORT

   - Filter harian, mingguan, bulanan, dan custom.
   - Custom mewajibkan tanggal mulai/selesai dan range yang valid.
   - HR dan Top Management dapat membaca laporan.
   - Hanya HR dapat export XLSX/PDF sesuai API Contract.
   - Export tanpa watermark.
   - Streaming dan spreadsheet formula-injection protection.
   - Audit DOWNLOAD.
   - Gunakan Asia/Jakarta untuk boundary tanggal.

6. MASTER JENIS IZIN DAN SALDO

   - Seed 15 jenis hanya dari daftar sumber yang valid; jangan menebak nama/kuota.
   - HR dapat GET/POST/PUT.
   - Role lain ditolak sesuai OpenAPI.
   - Nama unik, kuota non-negatif, active flag.
   - Leave balance dihitung per user/tahun.
   - Jangan kurangi saldo saat submit.
   - Kurangi saldo secara atomic hanya ketika final approval jenis yang memakai saldo.
   - Reject tidak mengurangi saldo.
   - Cegah saldo negatif dan double deduction.

7. CREATE KETIDAKHADIRAN

   Request multipart:
   - jenis_izin_id, tanggal_mulai, tanggal_selesai, alasan.
   - dokumen_pendukung wajib: PDF/JPG/PNG maksimal 5 MB.
   - lokasi_tujuan dan keperluan_tugas wajib untuk Perjalanan Dinas.

   Validasi:
   - Date range.
   - Active leave type.
   - Quota dan saldo.
   - Overlap policy hanya bila ada pada dokumen; bila belum ada, catat decision.
   - Employee aktif.

   Routing initial status:
   - Karyawan dengan atasan -> `menunggu_atasan`.
   - Karyawan tanpa atasan -> `menunggu_hr`.
   - Atasan -> `menunggu_hr`.
   - HR -> `menunggu_top_management`.

   Simpan request dan event notification intent dalam transaction boundary yang disepakati.
   BE.5 akan menyediakan persistence notification; jangan silently drop side effect.

8. LIST, DETAIL, DAN HISTORI SENDIRI

   - Pemohon melihat request sendiri.
   - Atasan hanya inbox bawahan langsung pada tahap Atasan.
   - HR hanya request pada tahap HR dan monitoring yang diizinkan.
   - Top Management hanya request HR pada tahap Top Management.
   - Detail mencakup approval history berurutan.
   - Forbidden harus 403 tanpa existence leak.
   - Pagination dan status filter sesuai OpenAPI.

9. DECISION KETIDAKHADIRAN

   Gunakan transaction + row lock/conditional update:
   - Verifikasi request masih pada tahap aktif.
   - Verifikasi actor adalah approver yang benar.
   - Reject wajib memiliki catatan.
   - Approve Atasan -> `menunggu_hr`.
   - Approve final HR/Top Management -> `disetujui`.
   - Reject tahap mana pun -> `ditolak`.
   - Insert approval history.
   - Update leave balance hanya pada final approval yang relevan.
   - Append Audit Log.
   - Emit notification intent untuk pemohon dan approver berikutnya.
   - Competing decision yang kalah -> 409 ALREADY_DECIDED.

10. DELEGATION

    - Hanya Atasan yang sedang menjadi approver request bawahannya.
    - Ubah status ke `menunggu_hr`.
    - Insert history dengan keputusan `delegate`.
    - Catatan optional.
    - Audit dan notification intent.
    - Race dengan decision/escalation -> satu pemenang; lainnya 409.

11. AUTO-ESCALATION

    Worker:
    - Hanya request `menunggu_atasan`.
    - Threshold 2x24 jam dari waktu request diterima.
    - Tidak berlaku untuk HR -> Top Management.
    - Claim batch dengan locking yang aman untuk multi-worker.
    - Update ke `menunggu_hr`.
    - Insert approval history `auto_escalate`, approver_id NULL.
    - Audit actor sistem.
    - Emit notification intent ke HR dan pemohon.
    - Idempotent dan aman dijalankan ulang.

12. LEMBUR

    Create multipart:
    - tanggal, jam_mulai, jam_selesai, alasan.
    - dokumen_pendukung optional PDF/JPG/PNG maksimal 5 MB.
    - `jam_selesai` harus setelah `jam_mulai`.
    - Durasi dihitung server sesuai Database Schema.

    Routing, decision, approval history, delegation/escalation policy, concurrency,
    authorization, audit, dan notification intent mengikuti Ketidakhadiran.
    Jika endpoint delegate Lembur tidak ada di API Contract:
    - Jangan menambah endpoint.
    - Terapkan hanya flow yang dapat dipicu kontrak atau catat gap.

    Rekap:
    - HR saja.
    - Filter periode, department, employee, pagination.
    - Jangan menghitung kompensasi lembur.

13. DOMAIN EVENT/NOTIFICATION BOUNDARY

    Definisikan event typed:
    - LeaveSubmitted.
    - LeaveDecisionChanged.
    - LeaveDelegated.
    - LeaveAutoEscalated.
    - OvertimeSubmitted.
    - OvertimeDecisionChanged.
    - OvertimeAutoEscalated.

    Event memuat ID dan actor minimum, bukan PII berlebihan.
    Pastikan transaction/event delivery strategy terdokumentasi. Jangan mengklaim
    notification side effect selesai sampai BE.5 menghubungkannya secara idempotent.

14. PHOTO RETENTION WORKER

    Harian:
    - Select attendance lebih dari 3 bulan dengan `foto_url` tidak NULL.
    - Claim batch secara aman.
    - Hapus file Nextcloud.
    - Set `foto_url=NULL` setelah delete berhasil.
    - Jangan hapus row attendance.
    - Retry failure tanpa menghapus URL sebelum file berhasil ditangani.
    - Log aggregate tanpa PII.

15. TEST

    Unit:
    - WFO distance boundary 99.99/100/100.01 meter.
    - WFH/WFA tidak radius-rejected.
    - Duplicate check-in dan checkout without check-in.
    - Semua routing pemohon.
    - Reject requires note.
    - Final approval deducts once.
    - Insufficient balance/quota.
    - Delegation dan escalation state machine.
    - Overtime duration.

    Integration/concurrency:
    - Migration dan constraints.
    - Upload cleanup failure.
    - Decision vs decision.
    - Decision vs delegate.
    - Decision vs auto-escalation.
    - Scheduler repeat run.
    - Photo cleanup repeat/failure.
    - Row-level authorization seluruh role.
    - Export and audit.

Quality gates:
1. Format, tidy, vet, unit/integration/concurrency tests, linter.
2. Build API dan worker.
3. OpenAPI lint dan operation conformance.
4. Migration up/down-one.
5. Docker Compose config dan worker smoke test.
6. Scan secret, PII, orphan files, goroutine/resource leak.

Git hanya bila diotorisasi:
- Branch: `feat/be-attendance-approval`
- Commit contoh:
  - `feat(attendance): add check-in and live feed`
  - `feat(report): add attendance report and export`
  - `feat(leave): add leave request and approval workflow`
  - `feat(overtime): add overtime workflow and recap`
  - `feat(worker): add escalation and photo retention jobs`
  - `test(approval): cover concurrent decisions`
- PR: `feat(attendance): implement attendance and approval workflows`
- Jangan push/open PR tanpa izin eksplisit.

Aturan akhir:
- Tidak ada reminder absensi.
- Tidak ada budget approval Perjalanan Dinas.
- Tidak ada kompensasi lembur.
- Tidak ada endpoint tambahan tanpa OpenAPI update.
- Semua decision atomic dan fail-closed.
- File selalu melalui backend/Nextcloud.
```

## Acceptance Criteria

- [ ] Tujuh tabel attendance/leave/overtime sesuai schema.
- [ ] Check-in/out, mode kerja, file validation, dan WFO 100 m sesuai kontrak.
- [ ] Live feed/report/export memiliki authorization yang benar.
- [ ] Dokumen Ketidakhadiran wajib; dokumen Lembur opsional.
- [ ] Empat jalur routing pemohon benar.
- [ ] Decision, delegation, dan escalation atomic serta tercatat di history/audit.
- [ ] Competing operation menghasilkan satu pemenang dan 409 untuk yang kalah.
- [ ] Saldo cuti tidak negatif dan hanya dikurangi sekali pada final approval.
- [ ] Worker escalation dan photo retention idempotent.
- [ ] Event boundary siap dihubungkan ke notifikasi BE.5.
- [ ] Seluruh test dan quality gate lulus.

## Files yang Akan Dibuat atau Disesuaikan

```text
backend/
├── internal/
│   ├── domain/{attendance,leave,overtime,approval_event}.go
│   ├── dto/{attendance,leave,overtime}_*.go
│   ├── repository/{attendance,leave,overtime}_repository.go
│   ├── service/{attendance,leave,overtime}_service.go
│   ├── handler/{attendance,report,leave,overtime}_handler.go
│   ├── worker/{auto_escalation,photo_retention}.go
│   └── export/attendance_exporter.go
├── migrations/*_create_attendance_leave_overtime_tables.sql
├── tests/{attendance,approval,worker}_integration_test.go
└── cmd/{api,worker}/main.go

docs/openapi.yaml
docs/openapi-decisions.md
```

Ikuti struktur existing dan hindari package duplikat.
