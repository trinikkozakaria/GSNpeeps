# Prompt — Epic B-FE.4: Attendance & Approval Experience

**Agent**: Frontend  
**Branch**: `feat/fe-attendance-approval`  
**Estimasi**: 4–5 hari  
**Prerequisite**: B-FE.3 dan B-BE.4 selesai

## Prompt untuk Claude Code

```text
Implementasikan frontend Absensi tiga tab, Laporan Kehadiran, Ketidakhadiran,
Lembur, dan approval GSNpeeps.

Sebelum mulai:
- Baca CLAUDE, OpenAPI, workflow specs, TASK_ATTENDANCE, TASK_APPROVALS.
- Gunakan hris-frontend skill.
- Pastikan seluruh multipart field/status/error mirror OpenAPI.
- Pertahankan auth, UI, query, form, upload, dan table patterns existing.

1. INFORMATION ARCHITECTURE

   Menu Absensi memiliki tiga tab:
   - Kehadiran.
   - Ketidakhadiran.
   - Lembur.

   Routes tambahan:
   - Histori/pengajuan sendiri.
   - Approval inbox dan detail untuk Atasan/HR/Top Management.
   - Live Feed dan Laporan untuk HR/Top Management.
   - Rekap Lembur untuk HR.
   - Master Jenis Izin untuk HR.

   Navigation dan action mengikuti access matrix; direct URL tetap guarded.

2. KEHADIRAN CAMERA FLOW

   State machine eksplisit:
   - idle.
   - requesting_camera.
   - camera_ready.
   - requesting_location.
   - preview.
   - submitting.
   - success.
   - recoverable_error.

   Requirements:
   - Request camera/location hanya setelah user action.
   - Pilih WFO/WFH/WFA.
   - Capture foto dan preview.
   - Jangan simpan image lebih lama dari kebutuhan.
   - Tampilkan timestamp information tanpa menganggap client time authoritative.
   - Multipart sesuai OpenAPI.
   - Disable duplicate submit.
   - Bersihkan MediaStream tracks saat selesai/unmount.

3. FALLBACK UPLOAD

   - Muncul ketika camera unsupported/denied/error.
   - JPG/PNG maksimal 5 MB.
   - Beri panduan watermark sesuai decision backend/frontend.
   - Jangan memanipulasi waktu untuk menggantikan server timestamp.
   - Preview/remove/replace accessible.

4. GEOLOCATION DAN WFO

   - WFO meminta location dan menjelaskan alasan.
   - WFH/WFA mengikuti field kontrak tetapi tidak menampilkan klaim radius.
   - Tangani permission denied, timeout, unavailable.
   - Backend authoritative untuk radius 100 m.
   - OUT_OF_RADIUS menampilkan pesan actionable tanpa mengungkap konfigurasi sensitif.
   - Jangan polling lokasi terus-menerus.

5. ATTENDANCE RESULT DAN ERRORS

   Tampilkan waktu network, mode kerja, status, alamat, dan photo preview bila response
   mengizinkan.

   Tangani:
   - DUPLICATE_CHECKIN.
   - CHECKOUT_WITHOUT_CHECKIN.
   - OUT_OF_RADIUS.
   - FILE_TOO_LARGE.
   - UNSUPPORTED_FILE_TYPE.
   - Network/500.

   Tidak ada reminder absensi.

6. LIVE FEED DAN LAPORAN

   - HR/Top Management read.
   - Karyawan/Atasan tidak fetch.
   - Date/period/department filters.
   - Loading/empty/error.
   - Photos lazy-loaded dan alt text aman.
   - Report table responsive.
   - Export XLSX/PDF hanya ditampilkan pada HR.
   - Revoke blob URL setelah download.

7. FORM KETIDAKHADIRAN

   Field mirror OpenAPI:
   - Jenis izin.
   - Tanggal mulai/selesai.
   - Alasan.
   - Dokumen wajib PDF/JPG/PNG, maksimal 5 MB.
   - Lokasi tujuan dan keperluan tugas conditional untuk Perjalanan Dinas.

   Requirements:
   - Client validation + server error mapping.
   - Saldo/kuota error UX.
   - No optimistic approved status.
   - Confirmation summary sebelum submit bila design pattern mendukung.

8. FORM LEMBUR

   - Tanggal, jam mulai/selesai, alasan.
   - Dokumen optional.
   - Validate end after start.
   - Durasi dari server menjadi authoritative.
   - Jangan tampilkan kalkulasi kompensasi.

9. HISTORI SENDIRI

   - List/pagination status pengajuan sendiri.
   - Detail dan approval timeline.
   - Status badge + text.
   - Deep link siap untuk notification.
   - Pemohon tidak melihat decision controls.

10. APPROVAL INBOX

    - Atasan: bawahan langsung tahap Atasan.
    - HR: request tahap HR.
    - Top Management: request HR tahap Top Management.
    - Filter/status/page mengikuti API.
    - Jangan menampilkan data di luar response.
    - Role tidak berwenang tidak fetch.

11. DECISION DAN DELEGATION

    - Approve/reject dialog.
    - Reject mewajibkan catatan.
    - Atasan memiliki delegation action untuk Ketidakhadiran sesuai endpoint.
    - Jangan membuat delegation endpoint Lembur bila tidak ada di OpenAPI.
    - Disable controls saat pending.
    - 409 ALREADY_DECIDED -> refresh detail, jelaskan sudah diproses pihak lain.
    - Setelah sukses invalidate inbox, detail, history, metrics, dan notification count
      yang relevan.
    - Jangan optimistic update decision.

12. MASTER JENIS IZIN DAN REKAP

    - HR CRUD sesuai endpoint yang tersedia (GET/POST/PUT, tanpa DELETE).
    - Top Management hanya bila OpenAPI mengizinkan read; jangan menebak.
    - Validate name/quota/active.
    - Rekap Lembur HR dengan period/department/employee/page.

13. ACCESSIBILITY DAN RESPONSIVE

    - Camera controls memiliki accessible name.
    - Permission/error announcements memakai live region secara bijak.
    - Focus kembali setelah dialog.
    - Timeline semantic.
    - Mobile approval actions tetap terlihat dan tidak mudah salah tekan.
    - Status tidak hanya warna.

14. TEST

    Component:
    - Camera success/denied/unavailable/unmount cleanup.
    - WFO location dan WFH/WFA.
    - Fallback validation.
    - Attendance domain errors.
    - Conditional travel fields.
    - Required vs optional document.
    - Overtime time validation.
    - Approval role visibility.
    - Reject note.
    - 409 refresh behavior.
    - Export role visibility/blob cleanup.

    E2E:
    - Karyawan check-in/out.
    - Karyawan submit leave -> Atasan approve -> HR approve.
    - Karyawan tanpa atasan -> HR.
    - Atasan submit -> HR.
    - HR submit -> Top Management.
    - Delegation.
    - Already-decided conflict.
    - Direct forbidden routes.

Quality gates:
1. Format/lint/test/E2E/build.
2. OpenAPI mock alignment.
3. Camera/geolocation tests di supported browser.
4. Accessibility/responsive review.
5. No MediaStream/blob/object URL leak.
6. No PII fixture atau raw file logging.

Git hanya bila diotorisasi:
- Branch `feat/fe-attendance-approval`.
- Commit contoh:
  - `feat(attendance): add camera and location check-in`
  - `feat(leave): add leave request experience`
  - `feat(overtime): add overtime request experience`
  - `feat(approval): add role-scoped approval inbox`
  - `feat(report): add attendance reports and exports`
- PR: `feat(attendance): add attendance and approval UI`
- Jangan push/open PR tanpa izin.

Aturan akhir:
- Tidak ada reminder.
- Tidak ada kompensasi lembur.
- Tidak ada budget approval.
- Tidak ada endpoint client-only.
- Backend tetap authoritative untuk time/radius/authorization/status.
```

## Acceptance Criteria

- [ ] Tiga tab Absensi lengkap dan role-aware.
- [ ] Camera/location/fallback lifecycle aman.
- [ ] WFO/100 m serta domain error UX benar.
- [ ] Dokumen Ketidakhadiran wajib dan Lembur opsional.
- [ ] Seluruh jalur approval dapat digunakan sesuai role.
- [ ] Reject/delegate/409 behavior sesuai kontrak.
- [ ] Live Feed/laporan/export/master/rekap memiliki access yang benar.
- [ ] Test, accessibility, responsive, lint, dan build lulus.

## Files yang Akan Dibuat atau Disesuaikan

```text
frontend/src/modules/
├── attendance/{api,components,hooks,pages,tests}/
├── attendance-reports/{api,components,hooks,pages,tests}/
├── leave/{api,components,hooks,pages,schemas,tests}/
├── overtime/{api,components,hooks,pages,schemas,tests}/
└── approvals/{api,components,hooks,pages,tests}/
```

Gunakan React JavaScript/JSX, React Router, Axios, TanStack Query, React Hook Form, Zod,
Tailwind CSS, Vitest/Testing Library, Playwright, dan pnpm.
