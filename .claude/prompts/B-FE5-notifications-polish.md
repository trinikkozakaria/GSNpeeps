# Prompt — Epic B-FE.5: Notifications, Access Admin & UI Polish

**Agent**: Frontend  
**Branch**: `feat/fe-notifications-access`  
**Estimasi**: 2.5–3 hari  
**Prerequisite**: B-FE.4 dan B-BE.5 selesai

## Prompt untuk Claude Code

```text
Implementasikan Notification Center, AKSES, Audit Log, dan final frontend polish
GSNpeeps.

Sebelum mulai:
- Baca CLAUDE, OpenAPI, TASK_NOTIFICATIONS_ACCESS, dan access matrix.
- Gunakan hris-frontend skill.
- Pertahankan patterns existing dan jangan membuat notification contract baru.

1. NOTIFICATION BELL

   - Tampil untuk seluruh role.
   - Fetch `/notifikasi/unread-count` hanya saat authenticated.
   - Badge accessible; angka besar memiliki treatment aman.
   - Loading tidak menampilkan fake count.
   - Polling hanya jika disetujui dan dengan interval/backoff wajar.
   - Refetch saat window focus bila query strategy mendukung.

2. NOTIFICATION LIST

   - List terbaru, filter unread, pagination.
   - Loading/empty/error.
   - Tipe/status memiliki icon + text, bukan warna saja.
   - Timestamp locale Indonesia.
   - Message diperlakukan sebagai text, bukan unsafe HTML.
   - Hanya data recipient dari API; jangan filter security client-side.

3. READ, DEEP LINK, DAN DISMISS

   - Mark read sesuai endpoint.
   - Update count/list cache konsisten.
   - Deep link berdasarkan `referensi_tipe` dan `referensi_id` melalui mapping terbatas,
     bukan URL mentah dari server.
   - Jika target forbidden/deleted, tampilkan state aman.
   - Dismiss memerlukan affordance/confirmation yang proporsional.
   - Setelah dismiss hilang dari UI dan count.
   - Jangan membuat undo yang memanggil endpoint restore karena tidak ada kontrak.

4. AKSES NAVIGATION

   - Hanya HR dan Top Management.
   - HR melihat mutation permission.
   - Top Management read-only; tidak render/tidak enable mutation controls.
   - Karyawan/Atasan tidak melihat menu dan direct route tidak fetch.

5. ROLE OVERVIEW

   - Tampilkan empat role dan jumlah user.
   - Loading/empty/error.
   - Tidak membuat role create/delete karena endpoint tidak ada.

6. PERMISSION MATRIX

   - Role selector/query sesuai OpenAPI.
   - Tampilkan module/action/readable labels.
   - HR dapat toggle/update satu permission sesuai request contract.
   - Top Management view-only.
   - Disable duplicate mutation.
   - Tangani 409/403/validation.
   - Invalidate navigation/capability query setelah update.
   - Jangan mengubah current client security behavior sebelum backend confirm;
     backend tetap authoritative.

7. AUDIT LOG

   - HR dan Top Management read-only.
   - Filter date range dan user.
   - Pagination URL-addressable.
   - Table responsive.
   - Detail JSON aman diformat tanpa unsafe HTML.
   - Jangan membuat edit/delete/export bila endpoint tidak ada.
   - Jangan menampilkan secret/PII raw walau backend keliru; redact defense-in-depth.

8. NOTIFICATION CACHE

   - Query keys user-scoped.
   - Login/logout mengganti/clear cache.
   - Read/dismiss update list dan count atomically atau invalidate.
   - Race polling vs mutation tidak menghidupkan kembali dismissed item di UI.
   - Jangan optimistic dismiss bila rollback UX tidak aman.

9. GLOBAL UI POLISH

   Review seluruh aplikasi:
   - Nama GSNpeeps.
   - Sidebar/nav consistency.
   - Loading, empty, error, forbidden.
   - Page title dan breadcrumb.
   - Toast/live region.
   - Confirmation dialog.
   - Form server errors.
   - Status labels.
   - Mobile/table behavior.
   - Keyboard/focus.
   - Reduced motion.
   - Color contrast.
   - Indonesian copy consistency.
   - Coming Soon hanya tiga modul.

10. PERFORMANCE

    - Route code splitting bila stack mendukung.
    - Lazy load chart/camera-heavy modules.
    - Avoid waterfall and duplicate requests.
    - Memoization hanya berdasarkan evidence.
    - Images lazy-loaded dengan dimensions.
    - Review bundle; jangan menambah library besar untuk fungsi kecil.

11. SECURITY REVIEW

    - Tidak render server message sebagai HTML.
    - Tidak log token/PII/file.
    - External links safe.
    - Blob URLs revoked.
    - Protected cache cleared on logout.
    - Direct forbidden route no fetch.
    - Permission UI bukan security.
    - CSP-compatible patterns; tidak memakai eval/inline unsafe workaround.

12. TEST

    Component/integration:
    - Bell count/loading/error.
    - Read updates count.
    - Dismiss removes and stays removed after refetch.
    - Deep link mapping and forbidden target.
    - Notification message XSS payload rendered as text.
    - HR permission mutation.
    - Top Management read-only.
    - Karyawan/Atasan no AKSES fetch.
    - Audit filters/pagination/redaction.

    E2E:
    - Approval event -> approver notification -> deep link.
    - Decision -> applicant notification.
    - Read/dismiss lifecycle.
    - HR changes permission.
    - Top Management views access/audit but cannot mutate.
    - Responsive/keyboard critical paths.

Quality gates:
1. Format/lint/unit/component/E2E/build.
2. OpenAPI mock alignment.
3. Accessibility scan semua critical routes.
4. Responsive review common breakpoints.
5. Bundle/performance inspection.
6. XSS/secret/PII/token scan.
7. Zero unresolved high/critical UI security issue.

Git hanya bila diotorisasi:
- Branch `feat/fe-notifications-access`.
- Commit contoh:
  - `feat(notification): add notification center`
  - `feat(access): add role and permission views`
  - `feat(audit): add audit log viewer`
  - `fix(ui): improve responsive and accessible states`
  - `test(frontend): add notification and access flows`
- PR: `feat(frontend): add notifications access and final polish`
- Jangan push/open PR tanpa izin.

Aturan akhir:
- Tidak ada create-notification UI.
- Tidak ada restore-dismiss endpoint.
- Tidak ada Role CRUD.
- Tidak ada Audit mutation.
- Top Management selalu read-only pada AKSES.
- Jangan melemahkan route/backend security untuk membuat UI test lulus.
```

## Acceptance Criteria

- [ ] Notification bell/list/count/read/deep-link/dismiss sesuai OpenAPI.
- [ ] Dismissed notification tidak muncul kembali setelah refetch.
- [ ] Message dirender sebagai text dan aman dari XSS.
- [ ] HR dapat update permission; Top Management view-only.
- [ ] Audit Log read-only, terfilter, responsif, dan aman.
- [ ] Cache lifecycle aman lintas login/logout dan mutation.
- [ ] Seluruh aplikasi konsisten, accessible, dan responsive.
- [ ] Test, bundle review, lint, dan production build lulus.

## Files yang Akan Dibuat atau Disesuaikan

```text
frontend/src/modules/
├── notifications/{api,components,hooks,pages,tests}/
├── access/{api,components,hooks,pages,tests}/
└── audit/{api,components,hooks,pages,tests}/

frontend/src/
├── components/layout/NotificationBell.*
├── routes/navigation.*
└── styles/
```

Gunakan React JavaScript/JSX, React Router, Axios, TanStack Query, React Hook Form, Zod,
Tailwind CSS, Vitest/Testing Library, Playwright, dan pnpm.
