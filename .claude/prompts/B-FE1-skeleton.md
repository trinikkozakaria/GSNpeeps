# Prompt — Epic B-FE.1: Frontend Skeleton & Design Foundation

**Agent**: Frontend  
**Branch**: `feat/fe-skeleton`  
**Estimasi**: 1.5–2 hari  
**Prerequisite**: Fase A selesai dan `docs/openapi.yaml` sudah disetujui

## Prompt untuk Claude Code

```text
Buat skeleton, architecture foundation, dan design foundation frontend GSNpeeps.

Sebelum mulai:
- Baca `CLAUDE.md`.
- Baca `docs/openapi.yaml`.
- Baca `.claude/skills/2.TASK_FOUNDATION.md`.
- Gunakan skill `.claude/skills/hris-frontend/SKILL.md` beserta references.
- Gunakan skill Git hanya jika tahap Git diotorisasi.
- Periksa repository dan pertahankan keputusan frontend yang sudah ada.

Konteks:
- Nama produk: GSNpeeps.
- Frontend: React JavaScript/JSX + Vite + React Router + Tailwind CSS.
- Package manager: pnpm.
- HTTP/server-state/form: Axios + TanStack Query + React Hook Form + Zod.
- Test: Vitest + Testing Library + Playwright.
- Production assets disajikan sebagai static assets melalui Nginx.
- Backend base path: `/api/v1`.
- UI dan pesan end-user: Bahasa Indonesia.
- Identifier code: English.
- Empat role: Karyawan, Atasan, HR, Top Management.
- Tidak boleh membuat mock contract yang berbeda dari OpenAPI.

STACK FINAL:

Jangan membuat proposal ulang atau menambahkan Next.js, TypeScript, npm/yarn, Jest, atau
HTTP/query/form/router alternatif. Client-state global, table/chart/icon, network mock,
component workbench, formatter, dan linter tambahan hanya dipilih jika task nyata
membutuhkannya.

1. PROJECT SCAFFOLD

   - Inisialisasi `frontend/` memakai Vite React JavaScript.
   - Gunakan pnpm dan commit hanya `pnpm-lock.yaml`.
   - Set target browser yang masuk akal untuk camera/geolocation.
   - Buat production build untuk static delivery.
   - Jangan mengaktifkan telemetry yang tidak diperlukan.
   - Jangan memasukkan endpoint/secret production ke bundle.

2. FOLDER STRUCTURE

   Target:
   - `src/app` untuk provider/bootstrap composition.
   - `src/routes` untuk React Router, guards, dan navigation.
   - `src/modules` untuk modul bisnis. Setiap modul memiliki folder sendiri.
   - Di dalam modul, pisahkan `pages`, `components`, `hooks`, `api`, `schemas`, `utils`,
     dan `tests` sesuai kebutuhan; jangan membuat folder kosong.
   - `src/components/ui` untuk primitives.
   - `src/components/layout` untuk shell.
   - `src/lib/api` untuk transport client, response envelope, dan error mapping.
   - `src/hooks`, `src/stores`, `src/schemas`, `src/mocks`, `src/styles`, `src/test`.
   - Jangan membuat `.storybook` sebelum component workbench dipilih khusus.

   Jangan membuat barrel file besar atau folder duplikat tanpa kebutuhan. Jangan commit
   `dist`, `node_modules`, coverage, Playwright output, cache, atau debug log.

3. ENVIRONMENT CONFIG

   Definisikan hanya public runtime/build config:
   - App name: GSNpeeps.
   - API base URL.
   - Environment marker bila diperlukan.

   Aturan:
   - Gunakan prefix public `VITE_`.
   - Validate config pada startup/build.
   - `.env.example` hanya placeholder.
   - Jangan expose JWT secret, DB, Redis, atau Nextcloud credential.
   - Jangan hardcode `janjikupadamu.id`; production URL berasal dari env.

4. DESIGN TOKENS

   Buat token untuk:
   - Color semantic: background, surface, text, muted, border, primary, danger,
     warning, success, info.
   - Typography scale.
   - Spacing.
   - Radius.
   - Shadow.
   - Focus ring.
   - Z-index.
   - Breakpoints.

   Aturan:
   - Contrast sesuai WCAG AA.
   - Status tidak hanya dibedakan warna.
   - Light theme wajib; dark theme hanya jika diminta.
   - Jangan menebak logo/brand visual yang belum diberikan.
   - Gunakan teks GSNpeeps, bukan membuat logo baru.

5. GLOBAL STYLES DAN APP BOOTSTRAP

   - Reset/base styles.
   - Font stack yang aman; jangan menambah external font tanpa keputusan.
   - Focus-visible.
   - Reduced motion.
   - Root providers hanya yang dibutuhkan.
   - Strict mode bila compatible.
   - Global error boundary.
   - Suspense/loading boundary sesuai stack.

6. API CLIENT FOUNDATION

   Buat:
   - Base URL config.
   - JSON request helper.
   - Multipart helper.
   - File response helper.
   - Bearer injection boundary.
   - AbortSignal/cancellation.
   - Safe timeout behavior.
   - Standard success/error envelope parser.
   - Mapping `error.fields` ke form.

   Aturan:
   - Jangan mengubah snake_case contract.
   - Jangan membuat endpoint string tersebar di component.
   - Jangan otomatis retry mutation.
   - Jangan retry 401/403/409/422 tanpa policy.
   - Jangan log token, payload PII, file, atau response sensitif.
   - Belum implementasikan persistent auth storage sebelum FE.2 decision.

7. SERVER STATE FOUNDATION

   - Configure TanStack Query client.
   - Default retry hanya untuk safe transient read.
   - Central query key factory.
   - Cache tidak boleh shared lintas user setelah logout.
   - Sediakan invalidate/reset mechanism.
   - Jangan prefetch data sensitif sebelum role diketahui.

8. ROUTER FOUNDATION

   Routes minimum:
   - `/login`.
   - Protected application shell placeholder.
   - `/403`.
   - `/404`.
   - Generic error route.

   Belum buat halaman bisnis.
   Guard belum boleh fail-open; protected shell menunggu auth state FE.2.

9. LAYOUT FOUNDATION

   Buat reusable:
   - AppShell.
   - Sidebar.
   - Topbar.
   - Mobile navigation/drawer.
   - PageContainer.
   - PageHeader.
   - Notification slot placeholder tanpa fake unread count.
   - User menu placeholder tanpa fake identity.

   Requirements:
   - Nama GSNpeeps konsisten.
   - Responsive mulai mobile.
   - Sidebar keyboard accessible dan focus-managed.
   - Skip link ke main content.
   - No horizontal overflow pada viewport kecil.

10. UI PRIMITIVES

    Minimum:
    - Button.
    - Input, Select, Textarea, Checkbox.
    - Field/Label/ErrorMessage.
    - Card.
    - Badge/StatusBadge.
    - Alert.
    - Dialog/ConfirmDialog.
    - Dropdown menu.
    - Tabs.
    - Skeleton/Spinner.
    - EmptyState.
    - ErrorState.
    - Pagination shell.
    - Table shell.

    Requirements:
    - Semantic HTML.
    - Keyboard and screen-reader behavior.
    - Disabled/loading state.
    - Tidak membuat abstraction satu kali pakai.

11. ERROR EXPERIENCE

    Sediakan treatment untuk:
    - Offline/network failure.
    - 400 validation.
    - 401 unauthenticated.
    - 403 forbidden.
    - 404 not found.
    - 409 conflict.
    - 413/415 upload.
    - 422 business rule.
    - 429 rate limit/account locked.
    - 500 internal error.

    Pesan tidak boleh menampilkan stack trace atau raw backend response.

12. TEST FOUNDATION

    Configure:
    - Vitest + Testing Library.
    - DOM matchers.
    - User-event.
    - API mocking yang mengikuti OpenAPI.
    - Accessibility smoke testing bila stack mendukung.
    - Playwright E2E skeleton tanpa production credential.

    Minimal test:
    - App renders GSNpeeps.
    - API envelope success/error mapping.
    - Validation field mapping.
    - Button loading prevents duplicate action.
    - Dialog keyboard close/focus return.
    - AppShell mobile navigation.
    - 403/404 rendering.

13. NGINX DAN CONTAINER

    - Frontend Dockerfile multi-stage.
    - Production image menyajikan static assets melalui Nginx sesuai System Design,
      atau terintegrasi dengan root Nginx decision yang disetujui.
    - Non-root bila image memungkinkan.
    - SPA route fallback React Router ke `index.html`.
    - Cache hashed assets; jangan cache HTML shell secara berlebihan.
    - Security headers tidak merusak camera/geolocation/API.
    - Update Compose tanpa expose service internal ke public selain entry Nginx.

14. COMMANDS DAN QUALITY GATES

    Sediakan scripts:
    - dev.
    - build.
    - lint.
    - format/check.
    - test.
    - test coverage bila dipilih.
    - test e2e.

    Jalankan:
    1. `pnpm install --frozen-lockfile`.
    2. Format check.
    3. Lint.
    4. Unit/component tests.
    5. Production build.
    6. Docker build.
    7. `docker compose config`.
    8. Bundle inspection dasar.
    9. Secret scan.

    Jika tool belum tersedia, jangan install global tanpa izin; laporkan gate.

Git hanya bila diotorisasi:
- Branch `feat/fe-skeleton`.
- Commit contoh:
  - `chore(frontend): initialize React application`
  - `feat(ui): add GSNpeeps design foundation`
  - `feat(api): add OpenAPI-aligned client foundation`
  - `feat(layout): add responsive application shell`
  - `test(frontend): add component test foundation`
  - `chore(docker): add frontend production image`
- PR: `feat(frontend): add GSNpeeps application foundation`
- Jangan push/open PR tanpa izin eksplisit.

Aturan akhir:
- Jangan membuat halaman bisnis palsu.
- Jangan membuat data dashboard palsu.
- Jangan membuat kontrak selain OpenAPI.
- Jangan menambahkan dependency di luar baseline tanpa kebutuhan dan extension decision.
- Jangan menyimpan token sebelum strategy FE.2 disetujui.
- Jangan menggunakan nama produk selain GSNpeeps.
```

## Acceptance Criteria

- [ ] Frontend memakai stack final tanpa dependency paralel.
- [ ] React + Tailwind scaffold dan production build berfungsi.
- [ ] Struktur module/shared jelas tanpa folder duplikat.
- [ ] Design tokens dan primitives memenuhi accessibility dasar.
- [ ] API client mengikuti success/error envelope OpenAPI.
- [ ] AppShell responsif, keyboard accessible, dan memakai nama GSNpeeps.
- [ ] 403, 404, loading, empty, dan error foundation tersedia.
- [ ] Test, lint, format, build, Docker build, dan Compose config lulus.
- [ ] Tidak ada secret, fake business data, atau auth storage yang belum disetujui.

## Files yang Akan Dibuat atau Disesuaikan

```text
frontend/
├── src/
│   ├── app/
│   ├── routes/
│   ├── modules/
│   ├── components/{ui,form,layout,feedback,data-table,charts}/
│   ├── lib/api/
│   ├── hooks/
│   ├── stores/               # hanya bila global client-state dipilih kemudian
│   ├── schemas/              # shared-only
│   ├── mocks/                # dev/test-only
│   ├── styles/
│   └── test/
├── public/
├── .env.example
├── Dockerfile
├── package.json
├── pnpm-lock.yaml
└── vite.config.js

docker-compose.yml
```

Gunakan `.jsx` untuk modul yang merender JSX dan `.js` untuk modul non-JSX.
