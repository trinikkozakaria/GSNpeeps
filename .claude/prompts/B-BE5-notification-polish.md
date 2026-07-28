# Prompt — Epic B-BE.5: Notifications, Access, Audit & Hardening

**Agent**: Backend  
**Branch**: `feat/be-notifications-access`  
**Estimasi**: 3–4 hari  
**Prerequisite**: Epic B-BE.4 selesai dan domain events approval tersedia

## Prompt untuk Claude Code

```text
Implementasikan Notifications, administrasi Role/Permission, Audit Log read API,
contract H-30 scheduler, dan backend hardening GSNpeeps.

Sebelum mulai:
- Baca `CLAUDE.md`, `docs/openapi.yaml`, dan decision log.
- Baca `.claude/skills/7.TASK_NOTIFICATIONS_ACCESS.md`.
- Baca API Contract v1.1 bagian 13–14.
- Baca Database Schema v1.1 ERD-5, ERD-6, dan scheduler.
- Baca PRD/User Story v1.2 untuk FR/US notifikasi dan AKSES.
- Baca event boundary hasil BE.4.
- Gunakan skill `hris-backend`.

Scope endpoint:
- GET `/api/v1/akses/role`
- GET `/api/v1/akses/permission`
- PUT `/api/v1/akses/permission`
- GET `/api/v1/akses/audit-log`
- GET `/api/v1/notifikasi`
- GET `/api/v1/notifikasi/unread-count`
- PUT `/api/v1/notifikasi/{id}/read`
- DELETE `/api/v1/notifikasi/{id}`

Scope worker:
- Notification creation dari event leave/overtime.
- Contract expiry notification H-30.
- Penyelesaian notification side effect escalation/delegation.

Non-scope:
- Email, SMS, push notification, atau reminder absensi.
- Endpoint create notification publik.
- Edit/delete Audit Log.
- Mutation AKSES oleh Top Management.

1. CONTRACT DAN MIGRATION

   Tambahkan `notifications` sesuai Database Schema:
   - id UUID PK.
   - recipient_user_id FK users, NOT NULL.
   - tipe, pesan.
   - referensi_id nullable.
   - referensi_tipe nullable.
   - event_key NOT NULL.
   - is_read default false.
   - dismissed_at nullable.
   - created_at.
   - UNIQUE `(recipient_user_id, event_key)`.
   - Index recipient/is_read/dismissed dan event_key.

   Gunakan permissions/audit_logs existing; tambahkan constraint/index/privilege yang
   belum diterapkan tanpa menduplikasi migration.

2. NOTIFICATION DOMAIN DAN EVENT KEY

   Definisikan tipe event:
   - ketidakhadiran_baru.
   - lembur_baru.
   - keputusan_approve.
   - keputusan_reject.
   - auto_escalate.
   - delegasi.
   - kontrak_akan_habis.

   Event key harus:
   - Deterministik.
   - Memuat jenis aggregate, aggregate ID, tahap/event version, dan recipient.
   - Tidak memuat pesan yang dapat berubah.
   - Aman terhadap retry dan concurrent consumer.

   Contoh pola konseptual:
   `<event_type>:<reference_id>:<stage_or_cycle>:<recipient_id>`

   Untuk contract H-30, cycle harus mengidentifikasi kontrak dan tanggal berakhir agar
   satu siklus menghasilkan satu notifikasi per recipient.

3. NOTIFICATION WRITER

   - Consume domain event BE.4.
   - Resolve recipient berdasarkan role, direct supervisor, dan tahap aktif.
   - Insert dengan `ON CONFLICT`/mekanisme setara yang tidak membuat duplikat.
   - Jangan menghidupkan kembali row dengan `dismissed_at`.
   - Simpan pesan Bahasa Indonesia yang aman dan ringkas.
   - Jangan masukkan PII berlebihan.
   - Pastikan failure/retry strategy terdokumentasi.
   - Jika memakai outbox, migration dan lifecycle harus lengkap; jangan membuat
     in-memory event yang hilang saat process crash tanpa decision eksplisit.

   Recipient:
   - Submission Karyawan dengan Atasan -> Atasan.
   - Submission Karyawan tanpa Atasan/Atasan -> HR.
   - Submission HR -> Top Management.
   - Decision/status -> pemohon.
   - Tahap berikutnya -> approver tahap berikutnya.
   - Delegation/escalation -> HR dan info pemohon.

4. CONTRACT H-30 WORKER

   Harian:
   - Query active contract yang berakhir tepat dalam window H-30 sesuai timezone
     Asia/Jakarta dan policy dokumen.
   - Kirim ke atasan langsung employee dan seluruh HR yang ditentukan policy.
   - Jangan kirim ke Top Management.
   - Jika employee adalah HR tanpa atasan, jangan self-notify; arahkan ke HR lain atau
     Top Management sesuai API Contract note.
   - Gunakan unique event key.
   - Multi-worker safe, batched, retryable, dan idempotent.
   - Repeat run pada hari yang sama tidak menambah row.

5. NOTIFICATION READ API

   GET list:
   - Selalu filter `recipient_user_id` dari JWT.
   - Exclude `dismissed_at IS NOT NULL`.
   - Filter optional `is_read`.
   - Urut terbaru.
   - Pagination sesuai OpenAPI.

   Unread count:
   - Recipient sendiri.
   - `is_read=false` dan tidak dismissed.
   - Query/index efficient.

   Mark read:
   - Recipient owner saja.
   - Atomic update.
   - Idempotent.
   - Non-owner -> 403 tanpa leak.

   Dismiss:
   - Isi `dismissed_at`.
   - Jangan DELETE fisik.
   - Recipient owner saja.
   - Event yang sama tidak boleh dibuat ulang.

6. ROLE DAN PERMISSION READ

   GET role:
   - HR dan Top Management.
   - Return empat role dan jumlah user sesuai contract.

   GET permission:
   - Query role wajib.
   - Return module/action capability.
   - HR dan Top Management read-only.

   Karyawan/Atasan:
   - Menu tidak terlihat di frontend.
   - Backend tetap return 403 pada direct access.

7. UPDATE PERMISSION

   - Hanya HR.
   - Top Management selalu 403.
   - Validasi role, module, action, dan boolean.
   - Cegah privilege configuration yang melanggar invariant produk, misalnya memberi
     mutation AKSES kepada Top Management, tanpa decision eksplisit.
   - Gunakan upsert/transaction aman.
   - Append audit before/after.
   - Invalidate permission cache agar perubahan langsung berlaku.
   - Karena permission tidak berada di JWT, tidak perlu menunggu JWT expired.

8. AUDIT LOG READ API

   - HR dan Top Management read-only.
   - Filter tanggal mulai/selesai, user_id, page, limit.
   - Default order terbaru.
   - Explicit columns; jangan return detail secret.
   - Karyawan/Atasan 403.
   - Tidak ada PUT/DELETE endpoint.
   - DB application role untuk runtime write hanya memiliki INSERT/SELECT yang perlu;
     revoke UPDATE/DELETE.
   - Verifikasi trigger/library ORM tidak mencoba update Audit Log.

9. AUDIT COVERAGE

   Review seluruh modul dan pastikan:
   - LOGIN, LOGOUT.
   - CREATE, UPDATE, DELETE/soft-delete.
   - APPROVE, REJECT, DELEGATE, AUTO_ESCALATE.
   - DOWNLOAD.
   - PERMISSION UPDATE.

   Audit harus memiliki actor/system identity, module, data ID, timestamp, IP bila ada,
   dan before/after aman. Redact password, JWT, session, document, salary detail yang
   tidak diperlukan, dan Authorization header.

10. HARDENING AUTHORIZATION

    Buat matrix test otomatis untuk 42 endpoint:
    - Public.
    - Karyawan.
    - Atasan sendiri/bawahan/bukan bawahan.
    - HR.
    - Top Management read dan mutation.
    - Missing/invalid/expired/logged-out token.

    Pastikan:
    - Query repository tetap scoped, bukan hanya handler check.
    - Forbidden mutation tidak memberi side effect.
    - ID valid milik user lain tidak bocor.
    - Soft-deleted/nonaktif resource tidak dapat diakses.

11. HARDENING INPUT DAN FILE

    Review:
    - Unknown JSON fields.
    - Body limit.
    - Pagination maximum.
    - Search/filter normalization.
    - UUID/date/time validation.
    - File extension/MIME/signature/size.
    - Path traversal.
    - Spreadsheet formula injection.
    - SQL injection.
    - Unsafe error/log output.
    - CORS production.
    - Rate limit default 120/minute.

12. OBSERVABILITY DAN FAILURE BEHAVIOR

    - Request ID pada response/log.
    - Scheduler job run ID, counters, duration, success/failure aggregate.
    - Jangan log notification message yang mengandung PII bila tidak perlu.
    - Redis unavailable -> auth/permission/session fail-closed.
    - Notification retry tidak menggandakan row.
    - Worker shutdown graceful.
    - Health endpoint tetap hanya mengekspos status aman DB/Redis sesuai OpenAPI.

13. TEST

    Unit:
    - Event key stability dan uniqueness antar stage/cycle/recipient.
    - Recipient resolution seluruh role.
    - Dismissed event tidak dibuat ulang.
    - Permission invariant dan cache invalidation.
    - Audit redaction.

    Integration/concurrency:
    - Dua writer event sama -> satu notification.
    - Scheduler H-30 repeat/concurrent -> satu per recipient.
    - Read/unread/dismiss row-level security.
    - Dismiss kemudian retry event -> tetap tidak muncul.
    - Permission update HR berhasil; Top Management ditolak.
    - Audit UPDATE/DELETE SQL ditolak DB.
    - Pagination/filter/index behavior.
    - 42-endpoint authorization matrix.

Quality gates:
1. Format, tidy, vet, unit/integration/concurrency tests, linter.
2. API/worker build.
3. OpenAPI lint dan tepat 42 operation.
4. Migration up/down-one.
5. Docker config dan repeat-run worker smoke test.
6. Security/secret/PII/log scan.
7. Query plan review untuk notification/audit list pada data representatif.
8. Tidak ada unresolved high/critical authorization issue.

Git hanya bila diotorisasi:
- Branch: `feat/be-notifications-access`
- Commit contoh:
  - `feat(notification): add idempotent notification store`
  - `feat(notification): connect approval domain events`
  - `feat(worker): add contract expiry notifications`
  - `feat(access): add role permission administration`
  - `feat(audit): enforce append-only audit logs`
  - `test(security): add endpoint authorization matrix`
- PR: `feat(notification): add notifications access and audit hardening`
- Jangan push/open PR tanpa izin eksplisit.

Aturan akhir:
- Tidak ada create-notification endpoint publik.
- Dismiss selalu soft-delete.
- Audit Log immutable.
- Top Management read-only pada AKSES.
- Notification dan scheduler wajib idempotent.
- Jangan menambah kanal email/SMS/push.
- Jangan menutup temuan test dengan melemahkan security.
```

## Acceptance Criteria

- [ ] Migration notifications dan seluruh constraint/index sesuai schema.
- [ ] Event key deterministik dan unique per recipient/event.
- [ ] Event approval/delegation/escalation menghasilkan recipient yang benar.
- [ ] Scheduler H-30 idempotent dan tidak salah self-notify.
- [ ] List/count/read/dismiss selalu row-scoped.
- [ ] Dismissed event tidak pernah dibuat ulang.
- [ ] HR dapat update permission; Top Management hanya read.
- [ ] Permission cache invalidated segera.
- [ ] Audit Log lengkap, ter-redact, dan immutable di level DB.
- [ ] Authorization matrix seluruh 42 endpoint lulus.
- [ ] Seluruh hardening dan quality gate lulus tanpa temuan critical.

## Files yang Akan Dibuat atau Disesuaikan

```text
backend/
├── internal/
│   ├── domain/{notification,permission,audit_log}.go
│   ├── repository/{notification,permission,audit}_repository.go
│   ├── service/{notification,access,audit}_service.go
│   ├── handler/{notification,access,audit}_handler.go
│   ├── worker/{notification_consumer,contract_expiry}.go
│   └── middleware/{rbac,rate_limit}.go
├── migrations/*_create_notifications_and_lock_audit.sql
├── tests/{notification,access,security_matrix}_integration_test.go
└── cmd/{api,worker}/main.go

docs/openapi.yaml
docs/openapi-decisions.md
```

Ikuti arsitektur existing dan jangan membuat implementasi paralel.
