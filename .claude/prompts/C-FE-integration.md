# Prompt — Fase C: Full Integration, Verification & Release Readiness

**Agent**: Full-stack / QA  
**Branch**: `chore/integration-release`  
**Estimasi**: 3–5 hari  
**Prerequisite**: B-BE.1–B-BE.5 dan B-FE.1–B-FE.5 selesai

## Prompt untuk Claude Code

```text
Lakukan integrasi penuh, verifikasi kontrak, hardening, dan release-readiness review
untuk GSNpeeps.

Sebelum mulai:
- Baca CLAUDE.md.
- Baca docs/openapi.yaml dan seluruh decision log/architecture docs.
- Baca `.claude/skills/8.TASK_INTEGRATION_RELEASE.md`.
- Gunakan skill hris-backend, hris-frontend, dan hris-git sesuai area.
- Baca seluruh migration, Compose, environment examples, dan test configuration.
- Jangan menambah feature baru pada fase ini.

Tujuan:
- Membuktikan frontend, API, worker, PostgreSQL, Redis, Nextcloud, dan Nginx bekerja
  sebagai satu sistem.
- Membuktikan tepat 42 endpoint mengikuti OpenAPI.
- Membuktikan empat role dan row-level authorization.
- Menghasilkan evidence release yang dapat direview.

1. BASELINE DAN CHANGE FREEZE

   - Catat commit/branch baseline.
   - Pastikan worktree dipahami; jangan menimpa perubahan user.
   - Daftar seluruh known issue.
   - Freeze feature scope.
   - Hanya perbaiki mismatch, defect, security, accessibility, observability, test,
     configuration, dan release documentation.

2. OPENAPI CONFORMANCE

   - Lint OpenAPI 3.1 tanpa error.
   - Hitung tepat 42 operation.
   - Resolve seluruh `$ref`.
   - Bandingkan route registry backend dengan OpenAPI:
     - Tidak ada endpoint undocumented.
     - Tidak ada endpoint OpenAPI yang hilang.
     - Method/path/status/media type sama.
   - Bandingkan frontend API calls/mocks/DTO/schema dengan OpenAPI.
   - Verifikasi JSON snake_case, pagination meta, file upload, dan file stream.
   - Perbaiki OpenAPI lebih dulu jika keputusan kontrak yang disetujui berubah.

3. MIGRATION DAN DATA INTEGRITY

   - Jalankan migration dari database kosong sampai latest.
   - Verifikasi 26 tabel, constraint, FK, CHECK, UNIQUE, dan index.
   - Jalankan rollback satu per satu pada database test lalu migrate up kembali.
   - Jangan menjalankan destructive rollback pada data/production.
   - Verifikasi seed empat role idempotent.
   - Verifikasi hanya satu Top Management pada fixture.
   - Verifikasi soft-delete, audit append-only, notification uniqueness, leave balance,
     dan approval state constraints.

4. DOCKER TOPOLOGY

   Verifikasi:
   - Nginx.
   - Frontend.
   - Backend API.
   - Cron worker.
   - PostgreSQL 16.
   - Redis 7.
   - Nextcloud.

   Requirements:
   - Hanya Nginx expose public 80/443 pada topology production.
   - Service lain internal network.
   - Healthchecks valid.
   - Containers run non-root bila memungkinkan.
   - Secrets dari environment/secret mechanism, bukan image/Compose.
   - Volume persistence benar.
   - Graceful stop API/worker.
   - `docker compose config` dan builds lulus.

5. ENVIRONMENT VALIDATION

   - `.env.example` lengkap tanpa secret nyata.
   - Backend/worker/frontend/Nginx menggunakan nama variable konsisten.
   - Tidak ada production default credential.
   - CORS production hanya origin GSNpeeps.
   - JWT TTL 8 jam.
   - Rate limits 5 login failure dan 120/min default.
   - Nextcloud technical credential tidak pernah masuk frontend.
   - Domain deployment configurable; nama produk tetap GSNpeeps.

6. END-TO-END AUTH

   Test:
   - Login valid untuk empat role.
   - Invalid credential tidak enumerate account.
   - Failure ke-1–4 -> 401.
   - Failure ke-5 -> 429 ACCOUNT_LOCKED.
   - Locked account tetap locked dengan password benar.
   - Logout -> token lama invalid.
   - Missing/invalid/expired/session-missing token.
   - Redis unavailable -> fail-closed.
   - Frontend clears user cache on logout/401.
   - Tidak ada refresh flow.

7. END-TO-END EMPLOYEE

   HR:
   - Create employee lengkap.
   - Duplicate NIP/email/KTP.
   - List/search/filter/page.
   - Detail.
   - Partial update.
   - Document upload valid/invalid/oversize.
   - Export XLSX/PDF.
   - Soft-delete dan session invalidation.

   Top Management:
   - Read list/detail/dashboard.
   - Semua mutation ditolak.

   Karyawan/Atasan:
   - Employee DB forbidden.
   - Profil sendiri read-only.
   - Gaji hanya bulan berjalan.

8. END-TO-END ATTENDANCE

   - Check-in WFO inside radius.
   - WFO outside radius.
   - WFH/WFA.
   - Duplicate check-in.
   - Checkout without check-in.
   - Camera denied/fallback upload.
   - Invalid/oversize file.
   - HR live feed/report/export.
   - Top Management report read-only.
   - Karyawan/Atasan global report forbidden.
   - Photo retention worker retry/repeat.

9. END-TO-END APPROVAL

   Jalur:
   - Karyawan dengan Atasan -> Atasan -> HR.
   - Karyawan tanpa Atasan -> HR.
   - Atasan -> HR.
   - HR -> Top Management.

   Variasi:
   - Approve.
   - Reject dengan note.
   - Reject tanpa note.
   - Delegation.
   - Auto-escalation setelah 2x24 jam.
   - Decision concurrency.
   - Decision vs delegation.
   - Decision vs escalation.
   - Insufficient balance/quota.
   - Supporting document required/optional.
   - Leave balance deducted exactly once.
   - Overtime duration, tanpa compensation.

10. END-TO-END NOTIFICATION

    - Submission -> approver notification.
    - Decision -> applicant + next approver.
    - Delegation/escalation.
    - Unread count.
    - Mark read.
    - Deep link.
    - Dismiss.
    - Retry event after dismiss tidak muncul.
    - Concurrent/repeat writer tidak duplicate.
    - Contract H-30 recipients.
    - HR without supervisor tidak self-notify secara salah.

11. ACCESS DAN AUDIT

    - HR reads/updates permissions.
    - Top Management reads but cannot update.
    - Karyawan/Atasan forbidden.
    - Permission update applies immediately.
    - Audit filters/pagination.
    - Audit coverage write/download/decision/auth.
    - Audit redaction.
    - Database rejects Audit Log UPDATE/DELETE.

12. AUTHORIZATION MATRIX

    Buat machine-readable test matrix untuk seluruh 42 endpoint:
    - Public.
    - Karyawan owner/non-owner.
    - Atasan owner/direct-report/non-report.
    - HR.
    - Top Management read/mutation/HR approval.
    - Soft-deleted resource.

    Setiap forbidden case harus membuktikan tidak ada DB/storage/notification/audit side
    effect yang salah.

13. SECURITY REVIEW

    Review:
    - SQL injection.
    - XSS/unsafe HTML.
    - CSRF relevance sesuai token strategy.
    - Authorization bypass/IDOR.
    - JWT algorithm/expiry/session.
    - Rate-limit bypass.
    - File signature/path traversal/polyglot risk.
    - Spreadsheet formula injection.
    - Secret/PII logging.
    - CORS dan security headers.
    - Dependency vulnerabilities menggunakan project-local tooling.
    - Docker user/network/secrets.

    Jangan melakukan destructive exploit pada environment non-test.
    Tidak boleh ada unresolved critical/high issue sebelum release-ready.

14. ACCESSIBILITY DAN RESPONSIVE

    Critical UI:
    - Login.
    - Navigation.
    - Employee form/table/detail.
    - Attendance camera/location.
    - Leave/overtime forms.
    - Approval dialogs/timeline.
    - Notifications.
    - Access/audit.
    - Dashboard charts.

    Verify:
    - Keyboard only.
    - Focus order/return.
    - Labels/errors/live regions.
    - Contrast.
    - Non-color status.
    - Reduced motion.
    - Mobile/tablet/desktop.
    - Zoom 200%.

15. PERFORMANCE DAN RELIABILITY

    - Backend query count/N+1.
    - Index/query plan list/report/audit/notification.
    - Bounded pagination/export.
    - Frontend bundle and code splitting.
    - Duplicate requests/cache.
    - Memory/blob/MediaStream leak.
    - Worker batching/retry/idempotency.
    - API/worker graceful shutdown.
    - Nextcloud timeout/failure cleanup.
    - PostgreSQL/Redis pool behavior.

16. OBSERVABILITY

    - Request ID end-to-end.
    - Structured HTTP logs.
    - Scheduler run ID/counters/duration.
    - Startup/shutdown logs.
    - Health returns only safe DB/Redis status.
    - No secret/PII in logs.
    - Document operational checks and failure symptoms.

17. BACKUP, RESTORE, DAN RUNBOOK

    Buat/review documentation:
    - Environment setup.
    - Migration deploy/rollback.
    - PostgreSQL backup/restore.
    - Nextcloud data considerations.
    - Redis treated as ephemeral session/cache.
    - Account lock/reset contract limitation.
    - Worker operations.
    - Log/health troubleshooting.
    - Known limitations and Coming Soon.

    Jangan menjalankan production backup/restore tanpa izin.

18. RELEASE EVIDENCE

    Buat `docs/release/readiness-report.md` berisi:
    - Scope dan commit SHA.
    - OpenAPI operation count/lint result.
    - Migration result.
    - Commands dan hasil test/build.
    - E2E matrix result.
    - Security findings.
    - Accessibility result.
    - Performance observations.
    - Screenshots critical UI tanpa PII.
    - Known issues/limitations.
    - Go/no-go recommendation berdasarkan evidence.
    - Items not run beserta alasan.

Quality gates:
1. Backend format/tidy/vet/lint/unit/integration/concurrency/build.
2. Frontend format/lint/unit/component/E2E/build.
3. OpenAPI lint, 42 operations, refs resolved.
4. Migration clean/up/down-one/re-up.
5. Docker Compose config + all image build.
6. Authorization matrix.
7. Scheduler repeat/concurrency tests.
8. Secret/PII/dependency/security scan.
9. Accessibility critical routes.
10. Release report complete.

Git/release actions hanya bila diotorisasi:
- Branch `chore/integration-release`.
- Commit contoh:
  - `test(e2e): add GSNpeeps critical workflows`
  - `test(security): add full authorization matrix`
  - `fix(integration): align frontend and API contracts`
  - `docs(release): add readiness report and runbook`
- PR: `chore(release): verify GSNpeeps release readiness`
- Jangan push, merge, tag, deploy, atau menjalankan production migration tanpa izin.

Aturan akhir:
- Jangan menambah fitur.
- Jangan melemahkan test/security agar build hijau.
- Jangan menyatakan test yang tidak dijalankan sebagai lulus.
- Jangan memakai data personal nyata.
- Jangan deploy otomatis.
- Jika blocked, laporkan evidence dan exact blocker.
```

## Acceptance Criteria

- [ ] Tepat 42 endpoint selaras antara OpenAPI, backend, frontend, dan mocks.
- [ ] Migration 26 tabel dari clean database serta rollback test lulus.
- [ ] Seluruh service build dan topology Docker sesuai System Design.
- [ ] E2E auth, employee, attendance, approval, notification, access, dan audit lulus.
- [ ] Authorization matrix empat role dan row-level scope lulus.
- [ ] Scheduler idempotency/concurrency lulus.
- [ ] Tidak ada unresolved critical/high security issue.
- [ ] Critical UI accessible dan responsive.
- [ ] Tidak ada secret, PII fixture, atau sensitive log.
- [ ] Release readiness report lengkap dengan evidence dan known limitations.
- [ ] Tidak ada push/tag/deploy/production mutation tanpa otorisasi.

## Files yang Akan Dibuat atau Disesuaikan

```text
backend/tests/
├── contract/
├── integration/
├── authorization/
└── e2e/

frontend/
├── src/**/*.test.*
└── e2e/

docs/
├── operations/
│   ├── deployment.md
│   ├── migration-rollback.md
│   ├── backup-restore.md
│   └── troubleshooting.md
└── release/
    └── readiness-report.md

docker-compose.yml
```

Perubahan production configuration harus tetap melalui review dan otorisasi.
