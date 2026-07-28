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
- Frontend: React + Tailwind CSS.
- Production assets disajikan sebagai static assets melalui Nginx.
- Backend base path: `/api/v1`.
- UI dan pesan end-user: Bahasa Indonesia.
- Identifier code: English.
- Empat role: Karyawan, Atasan, HR, Top Management.
- Tidak boleh membuat mock contract yang berbeda dari OpenAPI.

FRONTEND DECISION GATE:

Periksa apakah repository sudah menetapkan:
- React version dan build tool/framework.
- JavaScript atau TypeScript.
- Package manager.
- Router.
- Server-state/query library.
- Client-state strategy.
- Form + validation library.
- Component primitive/design system.
- Icon library.
- Linter dan formatter.
- Unit/component/E2E test stack.
- Mocking strategy.

Jika belum:
1. Buat `docs/architecture/frontend-stack-proposal.md`.
2. Usulkan satu stack utama dan maksimal satu alternatif per keputusan.
3. Pastikan compatible dengan React, Tailwind, static Nginx delivery, camera,
   geolocation, multipart upload, charts, tables, dan OpenAPI 3.1.
4. Jelaskan bundle impact, accessibility, testing, maintenance, dan learning cost.
5. Jangan menambah dependency utama atau scaffold yang menguncinya sebelum disetujui.
6. Berhenti dan laporkan keputusan yang dibutuhkan.

Jika stack sudah disetujui, kerjakan:

1. PROJECT SCAFFOLD

   - Inisialisasi `frontend/` memakai tool yang disetujui.
   - Gunakan package manager yang disetujui dan commit lockfile.
   - Set target browser yang masuk akal untuk camera/geolocation.
   - Buat production build untuk static delivery.
   - Jangan mengaktifkan telemetry yang tidak diperlukan.
   - Jangan memasukkan endpoint/secret production ke bundle.

2. FOLDER STRUCTURE

   Target:
   - `src/app` untuk bootstrap/providers.
   - `src/routes` untuk route definitions/guards.
   - `src/features` untuk feature modules.
   - `src/components/ui` untuk primitives.
   - `src/components/layout` untuk shell.
   - `src/api` untuk client, contracts, dan error mapping.
   - `src/hooks`, `src/stores`, `src/schemas`, `src/styles`, `src/test`.

   Jangan membuat barrel file besar atau folder duplikat tanpa kebutuhan.

3. ENVIRONMENT CONFIG

   Definisikan hanya public runtime/build config:
   - App name: GSNpeeps.
   - API base URL.
   - Environment marker bila diperlukan.

   Aturan:
   - Gunakan prefix public sesuai build tool.
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

   - Configure query/cache library yang disetujui.
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
    - Unit/component test.
    - DOM matchers.
    - User-event.
    - API mocking yang mengikuti OpenAPI.
    - Accessibility smoke testing bila stack mendukung.
    - E2E skeleton tanpa production credential.

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
    - SPA route fallback aman bila memakai client router.
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
    1. Install dari lockfile secara reproducible.
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
- Jangan memilih dependency utama tanpa decision.
- Jangan menyimpan token sebelum strategy FE.2 disetujui.
- Jangan menggunakan nama produk selain GSNpeeps.
```

## Acceptance Criteria

- [ ] Frontend stack telah disetujui dan terdokumentasi.
- [ ] React + Tailwind scaffold dan production build berfungsi.
- [ ] Struktur feature/shared jelas tanpa folder duplikat.
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
│   ├── features/
│   ├── components/{ui,layout}/
│   ├── api/
│   ├── hooks/
│   ├── stores/
│   ├── schemas/
│   ├── styles/
│   └── test/
├── public/
├── .env.example
├── Dockerfile
├── package.json
└── lockfile

docs/architecture/frontend-stack-proposal.md
docker-compose.yml
```

Nama file mengikuti language/tooling yang disetujui.
