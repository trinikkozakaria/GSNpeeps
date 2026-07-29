# Prompt — Epic B-FE.3: Employee, Profile & HR Dashboard

**Agent**: Frontend  
**Branch**: `feat/fe-employee-dashboard`  
**Estimasi**: 3–4 hari  
**Prerequisite**: B-FE.2 dan B-BE.3 selesai

## Prompt untuk Claude Code

```text
Implementasikan frontend Employee Database, Profil Saya, Metrik Personal, dan
Dashboard HR GSNpeeps.

Sebelum mulai:
- Baca CLAUDE.md, OpenAPI, access matrix, dan TASK_EMPLOYEE_DATA.
- Gunakan skill hris-frontend dan components/forms reference.
- Generate/mirror DTO dan validation dari OpenAPI; jangan menebak field.
- Pertahankan design system, API client, auth, query, dan form patterns existing.

Scope:
- Employee list/detail/create/edit/soft-delete/export/document upload.
- Master department/position dropdown.
- Profil Saya read-only.
- Metrik Personal.
- Dashboard HR dan monitoring Top Management.
- Coming Soon cards: Hiring Progress, Recruitment Cost, Benefit.

1. ROUTES DAN ACCESS

   Target konseptual:
   - `/karyawan`
   - `/karyawan/baru`
   - `/karyawan/:id`
   - `/karyawan/:id/edit`
   - `/profil-saya`
   - `/dashboard`

   Rules:
   - Employee create/edit/delete/export: HR.
   - Employee list/detail: HR dan Top Management; Top Management read-only.
   - Profil/Metrik Personal: Karyawan, Atasan, HR.
   - Dashboard HR: HR dan Top Management read-only.
   - Direct forbidden route tidak fetch resource.

2. REUSABLE DATA COMPONENTS

   Buat/reuse:
   - DataTable.
   - Pagination.
   - Search input dengan debounce/cancel.
   - Filter department/status.
   - Sort hanya jika OpenAPI mendukung.
   - StatusBadge.
   - Detail section.
   - Definition list.
   - File upload field.
   - Export menu.
   - Confirm soft-delete dialog.

   Filter dan page tersimpan di URL. Reset page saat filter berubah.

3. EMPLOYEE LIST

   - Query search, department_id, status, page, limit.
   - Loading skeleton, empty state, error/retry.
   - Responsive table/card treatment.
   - HR melihat create/edit/deactivate/export actions.
   - Top Management tidak melihat mutation controls.
   - Jangan prefetch detail sensitif untuk role lain.
   - Gunakan total/meta dari API, bukan menghitung client-side.

4. EMPLOYEE FORM

   Sections mengikuti OpenAPI:
   - Identity dan employment.
   - Department, position, supervisor.
   - Address.
   - KTP.
   - Contract.
   - BPJS.
   - NPWP.
   - Emergency contacts.
   - Education.
   - Salary.

   Requirements:
   - Client schema mirror Create/Update request.
   - Department change memfilter/reset invalid position.
   - Supervisor tidak boleh employee sendiri pada edit.
   - Dynamic collections accessible.
   - Date inputs dan monetary values locale-safe.
   - PUT edit hanya mengirim intended changed fields sesuai partial-update contract.
   - Server `error.fields` dipetakan ke field/nested field.
   - Unsaved-change warning.
   - Prevent duplicate submit.

5. DETAIL DAN SOFT-DELETE

   - Render Point 1–12 sesuai response.
   - Sensitive sections tidak muncul bila response/role tidak mengizinkan.
   - Document links memiliki label file dan safe external behavior.
   - Deactivate dialog menjelaskan soft-delete, bukan penghapusan permanen.
   - Setelah sukses invalidate list/detail dan redirect aman.
   - Tangani 404/403 tanpa existence leak.

6. DOCUMENT UPLOAD

   - Format: PDF/JPG/PNG/DOC/DOCX/XLS/XLSX/PPT/PPTX.
   - Tolak ZIP/RAR di client untuk UX, tetapi backend tetap authoritative.
   - Maksimum 5 MB.
   - Show selected file, size, replace/remove, progress bila supported.
   - Multipart field persis OpenAPI.
   - Tangani 413 dan 415 secara spesifik.
   - Jangan membaca/menyimpan isi file ke global state.

7. EXPORT

   - HR only.
   - XLSX/PDF.
   - Current filters atau single employee ID.
   - Gunakan authenticated file response.
   - Parse filename dengan aman.
   - Loading dan failure feedback.
   - Jangan membuka blob URL permanen; revoke setelah dipakai.

8. PROFIL SAYA

   - Read-only.
   - Identity berasal dari endpoint `/profil/saya`.
   - Tidak ada tombol edit/request perubahan.
   - Beri petunjuk perubahan melalui HR.
   - Gaji hanya bulan berjalan.
   - Jangan tampilkan dokumen Point 12 bila tidak ada pada response.

9. METRIK PERSONAL

   - Saldo cuti.
   - Lama kerja hari ini.
   - Clock-in/out history.
   - Karyawan/Atasan/HR saja.
   - Top Management tidak mempunyai route/menu.
   - Empty state jujur bila attendance belum ada.
   - Jangan generate angka dummy.

10. DASHBOARD

    Render:
    - Headcount.
    - Join/resign/turnover.
    - Leave.
    - Payroll cost.
    - Gender ratio.
    - Department composition.
    - Org chart.

    Requirements:
    - Selector periode harian/mingguan/bulanan/tahunan dan tanggal acuan.
    - Mingguan ditampilkan Senin-Minggu dalam timezone Asia/Jakarta.
    - Tampilkan karyawan/departemen aktif dan nonaktif pada bagian terpisah.
    - Tampilkan gender `belum_diisi` secara eksplisit jika count lebih dari nol.
    - Label status Bahasa Indonesia dipetakan dari enum API lowercase; jangan mengubah wire value.
    - Chart accessible dengan text summary/table alternative.
    - Currency/percentage/date formatting locale Indonesia.
    - Responsive.
    - HR dan Top Management melihat data sama secara read-only.
    - Coming Soon hanya untuk tiga modul yang disetujui.
    - Jangan membuat interaksi mutation pada Top Management.

11. CACHE DAN MUTATION

    - Query keys include filters/role/user scope.
    - Logout clears all data.
    - Create/update/delete invalidate exact keys.
    - Jangan optimistic update data sensitif kecuali rollback aman.
    - Cancel stale search/detail request.

12. TEST

    Component/integration:
    - List filter/page URL.
    - Role action visibility.
    - Create validation dan server field errors.
    - Partial edit payload.
    - Department-position dependency.
    - Soft-delete confirmation.
    - Upload format/size/413/415.
    - Export blob cleanup.
    - Profile read-only dan current-month salary.
    - Top Management no mutation.
    - Dashboard loading/empty/error/charts accessible.

    E2E:
    - HR create -> view -> edit -> deactivate.
    - HR upload/export.
    - Top Management read-only.
    - Karyawan Profil Saya.
    - Direct forbidden route.

Quality gates:
1. Format, lint, unit/component/E2E tests.
2. Production build.
3. OpenAPI mock/contract alignment.
4. Accessibility dan responsive review.
5. No secret, PII fixture, blob leak, or raw error.

Git hanya bila diotorisasi:
- Branch `feat/fe-employee-dashboard`.
- Commit contoh:
  - `feat(employee): add employee list and detail`
  - `feat(employee): add create edit and document forms`
  - `feat(profile): add read-only self profile`
  - `feat(dashboard): add HR metric dashboard`
  - `test(employee): cover RBAC forms and exports`
- PR: `feat(employee): add employee profile and dashboard UI`
- Jangan push/open PR tanpa izin.

Aturan akhir:
- Tidak ada self-service edit.
- Tidak ada full salary history di Profil Saya.
- Tidak ada business data dummy.
- Tidak ada mutation Top Management.
- Tidak ada kontrak frontend yang berbeda dari OpenAPI.
```

## Acceptance Criteria

- [ ] Employee list/detail/form sesuai OpenAPI dan responsif.
- [ ] HR mutation/export; Top Management read-only.
- [ ] Filter/pagination URL-addressable dan cache benar.
- [ ] Upload format/size/multipart serta error UX benar.
- [ ] Profil Saya read-only dan hanya gaji bulan berjalan.
- [ ] Metrik Personal tidak tersedia bagi Top Management.
- [ ] Dashboard accessible dan Coming Soon hanya tiga modul.
- [ ] Test, accessibility, responsive, lint, dan build lulus.

## Files yang Akan Dibuat atau Disesuaikan

```text
frontend/src/modules/
├── employees/{api,components,hooks,pages,schemas,tests}/
├── profile/{api,components,hooks,pages,tests}/
└── dashboard/
    ├── api/
    ├── components/{hr,employee}/
    ├── hooks/
    ├── pages/
    └── tests/

frontend/src/components/
├── data-table/
├── form/
└── charts/
```

Gunakan React JavaScript/JSX, React Router, Axios, TanStack Query, React Hook Form, Zod,
Tailwind CSS, Vitest/Testing Library, Playwright, dan pnpm.
