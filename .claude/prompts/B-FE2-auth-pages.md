# Prompt — Epic B-FE.2: Authentication, Session UX & Role Navigation

**Agent**: Frontend  
**Branch**: `feat/fe-auth-rbac`  
**Estimasi**: 1.5–2 hari  
**Prerequisite**: B-FE.1 dan B-BE.2 selesai; lima operasi auth/password OpenAPI tersedia

## Prompt untuk Claude Code

```text
Implementasikan login UI, auth state, logout, protected routes, dan role-aware
navigation GSNpeeps.

Sebelum mulai:
- Baca `CLAUDE.md`, `docs/openapi.yaml`, dan openapi decision log.
- Baca `.claude/skills/3.TASK_AUTH_RBAC.md`.
- Baca access matrix dan frontend skill references.
- Pastikan lima operasi auth/password OpenAPI 0.4.0 tersedia; jangan menambah refresh.
- Pertahankan stack dan patterns B-FE.1.
- Gunakan React JavaScript/JSX, React Router, Axios, TanStack Query, React Hook Form, Zod,
  Tailwind CSS, Vitest/Testing Library, Playwright, dan pnpm.

Contract:
- Login body: email + password.
- Login response: token, expires_in=28800, role, user{id,nama}.
- Logout memakai Bearer token.
- Tidak ada refresh token/cookie.
- `/auth/me` memulihkan identity dan role setelah reload.
- `/auth/me/password` menangani perubahan password pengguna yang sedang login.
- `/auth/reset-password` menangani self-reset dengan email, password saat ini, password baru,
  dan konfirmasi; tidak ada password sementara dari HR.

1. AUTH STORAGE DECISION

   Periksa decision tentang penyimpanan token dan pemulihan session.
   Jika belum ada:
   - Buat `docs/architecture/frontend-auth-session.md`.
   - Bandingkan in-memory only, sessionStorage, dan opsi backend cookie yang akan
     memerlukan perubahan kontrak.
   - Jelaskan XSS, reload UX, multi-tab, expiry, dan logout behavior.
   - Jangan memakai localStorage/sessionStorage/cookie diam-diam.
   - Minta persetujuan sebelum implementasi persistence.

   Gunakan `/auth/me` untuk memulihkan identity nama/role setelah reload tanpa mempercayai
   claim atau state browser sebagai enforcement security.
   Backend tetap satu-satunya enforcement.

2. LOGIN PAGE

   - Route `/login`.
   - Branding GSNpeeps.
   - Field email dan password dengan visible labels.
   - Password show/hide accessible.
   - Client validation mirror OpenAPI.
   - Disable duplicate submit.
   - Autofill attributes yang benar.
   - Enter submit.
   - Loading state.
   - Error mapping aman.
   - Redirect ke landing page role setelah sukses.
   - Jangan menampilkan password di log/devtools custom output.

3. ERROR UX

   - INVALID_CREDENTIALS: pesan generik, tidak membedakan email/password.
   - ACCOUNT_LOCKED: arahkan ke halaman self-reset resmi.
   - TOO_MANY_REQUESTS: tampilkan retry guidance tanpa countdown palsu.
   - Network/500: retry manual.
   - Validation: inline field error.
   - Jangan expose raw error/stack.

4. AUTH STATE

   State minimum:
   - status: initializing/authenticated/unauthenticated.
   - token handle sesuai storage decision.
   - expiresAt.
   - role.
   - user id/nama.

   Rules:
   - Jangan gunakan role frontend sebagai security.
   - Expiry dihitung dari server response/claim yang tervalidasi secara format.
   - Clear seluruh auth + server cache saat logout/expiry.
   - Jangan persist permission list.
   - Hindari flash protected content ketika initializing.

5. API CLIENT AUTH INTEGRATION

   - Inject Bearer token.
   - Satu handler untuk 401: clear session/cache dan arahkan login.
   - 403 menuju forbidden experience, bukan logout otomatis.
   - Jangan retry 401.
   - Cegah redirect loop pada login.
   - Support AbortSignal.
   - Redact Authorization dari debug logger.

6. LOGOUT

   - User action dari user menu.
   - Panggil POST `/auth/logout`.
   - Clear client state/cache meski network failure, sambil memberi feedback aman.
   - Jangan menganggap token tetap dapat dipakai.
   - Redirect login dan cegah back navigation menampilkan cached sensitive page.

7. ROUTE GUARDS

   - PublicOnlyRoute untuk login.
   - AuthenticatedRoute untuk application shell.
   - Role/capability guard untuk UX.
   - Dedicated 403.
   - Preserve intended destination hanya bila aman.
   - Direct URL unauthorized tidak boleh memicu fetch sensitif.
   - Unknown role -> fail-closed.

8. ROLE NAVIGATION

   Karyawan:
   - Profil Saya.
   - Metrik Personal.
   - Absensi: Kehadiran, Ketidakhadiran, Lembur.
   - Notifikasi.

   Atasan:
   - Seluruh menu Karyawan.
   - Approval bawahan langsung.

   HR:
   - Employee Database.
   - Dashboard HR.
   - Profil/Metrik Personal.
   - Absensi pribadi.
   - Live Feed dan Laporan.
   - Approval dan rekap.
   - Master Jenis Izin.
   - AKSES.
   - Notifikasi.

   Top Management:
   - Dashboard/data/live feed/laporan read-only.
   - Approval pengajuan HR.
   - AKSES read-only.
   - Notifikasi.
   - Tidak ada Metrik Personal.

   Coming Soon:
   - Hiring Progress.
   - Recruitment Cost.
   - Benefit.

   Jangan tampilkan menu AKSES pada Karyawan/Atasan.
   Gunakan satu navigation config, bukan condition tersebar.

9. USER MENU DAN SESSION EXPIRY

   - Tampilkan nama dan role label.
   - Logout action keyboard accessible.
   - Tangani expiry 8 jam.
   - Jangan membuat silent refresh karena endpoint tidak ada.
   - Optional warning sebelum expiry hanya jika disetujui; bukan reminder absensi.

10. TEST

    Unit/component:
    - Login validation dan submit.
    - INVALID_CREDENTIALS, ACCOUNT_LOCKED, 429, 500.
    - Duplicate submit.
    - Token injection/redaction.
    - 401 clear state/cache.
    - 403 tidak logout.
    - Logout success/failure.
    - Session expiry.

    Matrix:
    - Menu setiap empat role.
    - Top Management tidak melihat Metrik Personal/mutation.
    - Karyawan/Atasan tidak melihat AKSES/dashboard HR.
    - Unknown role fail-closed.
    - Direct forbidden route tidak fetch API.

    E2E:
    - Login -> role landing.
    - Logout -> old token unusable.
    - Reload behavior sesuai storage decision.
    - Locked account UX.

Quality gates:
1. Format, lint, unit/component/E2E tests.
2. Production build.
3. OpenAPI contract/mocks validation.
4. Accessibility scan login/nav/dialog.
5. Responsive test mobile/desktop.
6. Secret/token log scan.
7. No refresh endpoint/cookie implementation.

Git hanya bila diotorisasi:
- Branch `feat/fe-auth-rbac`.
- Commit contoh:
  - `feat(auth): add GSNpeeps login flow`
  - `feat(auth): add session lifecycle and logout`
  - `feat(rbac): add role-aware navigation and route guards`
  - `test(auth): cover login lockout and role matrix`
- PR: `feat(auth): add frontend authentication and role navigation`
- Jangan push/open PR tanpa izin eksplisit.

Aturan akhir:
- Jangan membuat refresh flow.
- Jangan membuat reset/change password UI tanpa endpoint.
- Jangan menyimpan token sebelum strategy disetujui.
- Jangan mengandalkan client role untuk security.
- Jangan tampilkan protected content saat auth initializing.
```

## Acceptance Criteria

- [ ] Auth storage/session strategy disetujui dan terdokumentasi.
- [ ] Login/logout/current-user/change-password sesuai OpenAPI.
- [ ] JWT expiry 8 jam dan 401 lifecycle ditangani.
- [ ] Account lockout mengarahkan user ke self-reset resmi tanpa membocorkan keberadaan email.
- [ ] Navigation matrix keempat role benar.
- [ ] Direct forbidden route fail-closed dan tidak fetch data sensitif.
- [ ] Logout membersihkan auth dan seluruh user-scoped cache.
- [ ] Change-password dan self-reset memiliki validasi, state sukses/gagal, serta login ulang
  setelah seluruh session dicabut.
- [ ] Tidak ada refresh-token flow.
- [ ] Test, accessibility, responsive, lint, dan build lulus.

## Files yang Akan Dibuat atau Disesuaikan

```text
frontend/src/
├── modules/auth/
│   ├── api/
│   ├── components/
│   ├── hooks/
│   ├── pages/
│   ├── schemas/
│   ├── store/
│   └── tests/
├── routes/                     # React Router route tree dan guards
├── components/layout/{Sidebar,Topbar,UserMenu}.*
└── lib/api/client.*

docs/architecture/frontend-auth-session.md
```

Gunakan JavaScript/JSX dan konvensi stack final.
