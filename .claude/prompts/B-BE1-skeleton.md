# Prompt — Epic B-BE.1: Backend Skeleton & Infrastruktur

**Agent**: Backend  
**Branch**: `feat/be-skeleton`  
**Estimasi**: 1–2 hari  
**Prerequisite**: Fase A selesai dan `docs/openapi.yaml` sudah disetujui

## Prompt untuk Claude Code

```text
Tolong buatkan skeleton dan infrastruktur backend GSNpeeps.

Sebelum mulai:
- Baca `CLAUDE.md`.
- Baca `docs/openapi.yaml` sebagai sumber kebenaran kontrak HTTP.
- Baca `.claude/skills/2.TASK_FOUNDATION.md`.
- Gunakan skill `.claude/skills/hris-backend/SKILL.md`.
- Baca seluruh reference yang ditunjuk oleh skill tersebut.
- Gunakan `.claude/skills/hris-git/SKILL.md` hanya pada tahap Git.
- Periksa repository dan pertahankan keputusan arsitektur yang sudah ada.

Konteks:
- Nama produk: GSNpeeps.
- Backend: Go REST API.
- Database: PostgreSQL 16.
- Cache, session, dan rate-limit counter: Redis 7.
- Storage file: Nextcloud self-hosted melalui WebDAV.
- API dan worker menggunakan image/codebase yang sama tetapi entrypoint berbeda.
- JWT berlaku 8 jam; implementasi autentikasi lengkap dikerjakan pada epic berikutnya.
- Base path API: `/api/v1`.
- Health endpoint public: `GET /health`.
- Format response mengikuti `docs/openapi.yaml`.
- Semua service selain Nginx berada di internal Docker network.

ARCHITECTURE DECISION GATE:

Sebelum menulis kode, periksa apakah repository sudah menetapkan:
- HTTP router.
- PostgreSQL driver dan pola data access.
- Migration tool.
- Configuration loader.
- Structured logger.
- Validator.
- Redis client.
- Test/assertion libraries.
- Linter.

Jika keputusan belum tersedia:
1. Buat `docs/architecture/backend-stack-proposal.md`.
2. Untuk tiap kategori, usulkan satu pilihan utama dan maksimal satu alternatif.
3. Jelaskan alasan, trade-off, maintenance status, dan dampaknya pada struktur kode.
4. Pastikan pilihan cocok dengan Go, PostgreSQL 16, Redis 7, WebDAV, Docker,
   OpenAPI 3.1, dan modular monolith.
5. Jangan menambah dependency utama atau membuat skeleton yang bergantung padanya
   sebelum pilihan disetujui pengguna.
6. Berhenti dan laporkan keputusan yang dibutuhkan.

Jika keputusan sudah tersedia dan disetujui, lanjutkan pekerjaan berikut.

Kerjakan dengan urutan:

1. GO MODULE DAN STRUKTUR DASAR

   - Pastikan `backend/go.mod` memakai module path yang disepakati.
   - Buat entrypoint:
     - `cmd/api/main.go`
     - `cmd/worker/main.go`
   - Buat struktur package modular:
     - `internal/config`
     - `internal/domain`
     - `internal/repository`
     - `internal/service`
     - `internal/handler`
     - `internal/middleware`
     - `internal/dto`
     - `internal/validation`
     - `internal/platform/postgres`
     - `internal/platform/redis`
     - `internal/platform/nextcloud`
     - `internal/platform/logger`
     - `internal/http/router`
     - `internal/http/response`
     - `internal/worker`
   - Jangan membuat package bisnis kosong hanya agar tree terlihat lengkap.
     Tambahkan file hanya bila memiliki kontrak atau implementasi nyata.

2. CONFIGURATION

   Buat config terstruktur untuk:

   - App:
     - environment
     - port
     - base URL
     - log level
     - shutdown timeout
   - HTTP:
     - read timeout
     - read header timeout
     - write timeout
     - idle timeout
     - max request body
   - PostgreSQL:
     - connection URL
     - max open connection
     - max idle connection
     - connection lifetime
   - Redis:
     - URL
     - pool size
     - dial/read/write timeout
   - JWT:
     - secret
     - TTL default 8 jam
   - Nextcloud:
     - WebDAV base URL
     - technical username
     - app password
     - root folder
   - CORS:
     - allowed origin
   - Rate limit:
     - login failure limit default 5
     - general request limit default 120 per menit

   Aturan:
   - Env-first.
   - Validasi seluruh variable wajib pada startup.
   - Kembalikan error yang menjelaskan nama konfigurasi yang hilang tanpa mencetak secret.
   - Jangan menyediakan insecure production fallback.
   - Buat `.env.example` dengan placeholder, bukan credential nyata.

3. STRUCTURED LOGGER

   - Inisialisasi berdasarkan environment dan level.
   - Dukungan `request_id`, method, path, status, duration, remote IP, dan user ID
     jika nanti tersedia.
   - Helper untuk mengambil logger dari `context.Context`.
   - Jangan gunakan `fmt.Println` atau logger standar ad-hoc di production code.
   - Redact Authorization, cookie, password, JWT secret, Nextcloud credential,
     dan data personal.

4. POSTGRESQL CONNECTION

   - Buka connection pool memakai driver yang telah disetujui.
   - Terapkan pool config dan ping dengan timeout.
   - Sediakan health check yang menerima context.
   - Wrap error dengan konteks.
   - Jangan menjalankan schema auto-migration.
   - Seluruh perubahan schema harus melalui migration tool yang disetujui.

5. REDIS CONNECTION

   - Buat client memakai library yang telah disetujui.
   - Terapkan timeout dan pool configuration.
   - Sediakan ping/health check dengan context.
   - Definisikan interface minimum agar session dan rate limit dapat diuji dengan mock/fake.
   - Belum perlu mengimplementasikan login session pada epic ini.

6. NEXTCLOUD WEBDAV ADAPTER

   Definisikan interface storage minimum:

   - Upload(ctx, path, reader, contentType) -> stored reference
   - Delete(ctx, path)
   - Health(ctx) bila WebDAV menyediakan check yang aman

   Implementasi awal:
   - Menggunakan technical account dan app password dari env.
   - Tidak mengekspos credential ke handler atau frontend.
   - Membatasi path di bawah root folder GSNpeeps.
   - Menolak path traversal.
   - Memakai timeout HTTP.
   - Tidak mengimplementasikan modul upload bisnis pada epic ini.

7. STANDARD HTTP RESPONSE

   Buat helper yang mengikuti `docs/openapi.yaml`:

   - Success.
   - Success dengan PaginationMeta.
   - Error.
   - Validation error dengan `error.fields`.
   - Method untuk menulis JSON sekali dan menetapkan Content-Type.

   Aturan:
   - Jangan expose internal error.
   - Jangan menulis header/body dua kali.
   - Jangan membuat response shape alternatif.
   - Tambahkan unit test untuk success, pagination, validation error, dan internal error.

8. VALIDATION

   - Gunakan validator yang telah disetujui.
   - Buat wrapper kecil agar handler tidak terikat ke detail library.
   - Format error field memakai nama JSON `snake_case`.
   - Pesan end-user menggunakan Bahasa Indonesia.
   - Jangan mengimplementasikan validation rule bisnis yang belum digunakan.

9. HTTP MIDDLEWARE

   Implementasikan middleware dasar:

   - Recovery:
     - Tangkap panic.
     - Log internal detail dengan request ID.
     - Return `500 INTERNAL_ERROR`.
   - Request ID:
     - Terima request ID yang valid atau buat UUID baru.
     - Kembalikan melalui response header.
   - Access logger:
     - Log setelah response selesai.
   - CORS:
     - Hanya origin dari config.
     - Jangan pakai wildcard pada production.
   - Body limit:
     - Batas global yang masuk akal.
     - Upload endpoint nanti tetap menerapkan batas file 5 MB.
   - Rate limit abstraction:
     - Siapkan interface/middleware boundary.
     - Implementasi policy login dan per-user dilakukan bersama endpoint terkait.

   Authentication dan RBAC:
   - Definisikan context type dan helper identity secara fail-closed.
   - Jangan membuat dummy middleware yang mengizinkan protected endpoint.
   - Protected business routes belum didaftarkan sampai Auth epic selesai.

10. ROUTER DAN HEALTH ENDPOINT

    - Buat router menggunakan pilihan yang telah disetujui.
    - Pasang middleware global dengan urutan yang terdokumentasi.
    - Sediakan route group `/api/v1` tanpa business endpoint kosong.
    - Implementasikan public `GET /health`.
    - Response sukses harus mengikuti OpenAPI:

      {
        "success": true,
        "data": {
          "status": "ok",
          "db": "connected",
          "redis": "connected"
        },
        "message": "Service healthy"
      }

    - Jika PostgreSQL atau Redis gagal, return `503 SERVICE_UNAVAILABLE`.
    - Jangan sertakan credential, DSN, hostname sensitif, atau stack trace.

11. API ENTRYPOINT

    `cmd/api/main.go` wajib:

    - Load dan validasi config.
    - Init logger.
    - Connect PostgreSQL.
    - Connect Redis.
    - Init Nextcloud adapter tanpa melakukan upload.
    - Wire dependencies secara eksplisit.
    - Start `http.Server` dengan seluruh timeout.
    - Tangani SIGINT/SIGTERM.
    - Graceful shutdown dengan timeout.
    - Tutup resource dalam urutan aman.
    - Return non-zero exit code bila startup gagal.

12. WORKER ENTRYPOINT

    `cmd/worker/main.go` wajib:

    - Load config dan init logger.
    - Connect resource yang sama melalui composition root terpisah.
    - Sediakan runner lifecycle dan graceful shutdown.
    - Belum menjalankan contract notification, auto-escalation, atau photo-retention job.
    - Jangan membuat loop kosong yang busy-wait.

13. MIGRATION INFRASTRUCTURE

    - Konfigurasikan migration tool yang telah disetujui.
    - Buat folder `backend/migrations`.
    - Tambahkan migration awal hanya jika dibutuhkan untuk extension PostgreSQL
      seperti `pgcrypto`, dan hanya bila sesuai Database Schema.
    - Jangan membuat 26 tabel bisnis pada prompt skeleton ini kecuali task migration
      terpisah sudah disetujui.
    - Sediakan command:
      - migrate-up
      - migrate-down-one
      - migrate-status
      - migrate-new
    - Pastikan rollback tidak menggunakan reset/drop database.

14. DOCKER DAN LOCAL DEVELOPMENT

    - Buat `backend/Dockerfile` multi-stage.
    - Jalankan binary sebagai non-root user.
    - Gunakan healthcheck API.
    - Tambahkan atau perbarui `docker-compose.yml` untuk:
      - backend-api
      - cron-worker
      - postgres:16
      - redis:7
      - nextcloud
      - internal network
    - Jangan mengekspos PostgreSQL, Redis, backend, worker, atau Nextcloud ke publik.
      Port lokal development hanya boleh ditambahkan bila memang diperlukan dan
      dijelaskan.
    - Jangan memasukkan secret ke image atau Compose file.
    - Gunakan env file/variables dan volume bernama.
    - Nginx/frontend boleh tetap placeholder bila belum menjadi scope epic ini.

15. COMMAND INTERFACE

    Buat Makefile atau task runner yang telah disetujui dengan target setara:

    - dev-api
    - dev-worker
    - build
    - test
    - vet
    - lint
    - fmt
    - migrate-up
    - migrate-down-one
    - migrate-status
    - migrate-new
    - docker-config

    Command harus bekerja dari lokasi yang terdokumentasi dan gagal dengan exit code
    non-zero saat terjadi error.

16. TEST

    Minimal test:

    - Config valid berhasil dimuat.
    - Config wajib yang hilang menghasilkan error tanpa membocorkan secret.
    - Standard success response.
    - Paginated response.
    - Validation error response.
    - Recovery middleware menghasilkan 500 dan request ID.
    - Request ID diteruskan ke response.
    - Health handler menghasilkan 200 ketika DB/Redis sehat.
    - Health handler menghasilkan 503 ketika salah satu dependency gagal.
    - Nextcloud adapter menolak path traversal.
    - Graceful shutdown atau lifecycle component dapat diuji tanpa sleep panjang.

Quality gates setelah implementasi:

1. Format seluruh kode Go.
2. Jalankan `go mod tidy`.
3. Jalankan `go vet ./...`.
4. Jalankan `go test ./...`.
5. Jalankan linter project.
6. Build API binary.
7. Build worker binary.
8. Jalankan migration status/up/down-one pada database test bila infrastructure tersedia.
9. Jalankan `docker compose config`.
10. Build container yang berubah.
11. Jalankan stack lokal bila aman.
12. Verifikasi `GET /health` menghasilkan 200 saat PostgreSQL dan Redis sehat.
13. Verifikasi endpoint menghasilkan 503 saat dependency yang wajib tidak sehat.
14. Scan diff untuk secret, credential, debug code, dan data personal.

Jika sebuah tool belum tersedia:
- Jangan install global dependency tanpa izin.
- Gunakan tool project-local yang sudah ada.
- Laporkan gate yang tidak dapat dijalankan beserta alasannya.
- Jangan menyatakan gate tersebut lulus.

Tahap Git hanya dilakukan jika pengguna meminta commit/push atau task aktif memberi
otorisasi eksplisit.

Branch:
- `feat/be-skeleton`

Commit yang disarankan:
- `chore(backend): initialize Go module structure`
- `feat(config): add validated environment configuration`
- `feat(platform): add postgres redis and nextcloud adapters`
- `feat(http): add response helpers and base middleware`
- `feat(health): add dependency health endpoint`
- `feat(worker): add worker lifecycle skeleton`
- `chore(docker): add backend development infrastructure`

Jangan membuat commit kosong hanya untuk mengikuti daftar.
Jangan push branch atau membuka PR tanpa otorisasi eksplisit.

Jika diotorisasi, judul PR:
- `feat(backend): add GSNpeeps skeleton and infrastructure`

Aturan akhir:
- Jangan mengimplementasikan endpoint bisnis pada epic ini.
- Jangan menambah refresh-token flow; API Contract hanya menetapkan JWT 8 jam
  dengan Redis session cross-check.
- Jangan menggunakan schema auto-migration.
- Jangan membuat authentication middleware yang fail-open.
- Jangan membuat library wrapper yang tidak memberi abstraction value.
- Jangan menyalin dependency atau credential dari project MBG.
- Identifier kode menggunakan English.
- Pesan end-user menggunakan Bahasa Indonesia.
- Semua perbedaan terhadap OpenAPI harus diperbaiki di OpenAPI lebih dulu atau
  dicatat sebagai keputusan kontrak; jangan membuat kontrak kedua di kode.
```

## Acceptance Criteria

- [ ] Architecture decision gate dipenuhi sebelum dependency utama ditambahkan.
- [ ] API dan worker binary dapat di-build.
- [ ] Configuration loader memvalidasi env wajib tanpa membocorkan secret.
- [ ] PostgreSQL dan Redis connection memiliki timeout, pool, dan health check.
- [ ] Nextcloud adapter memakai WebDAV, technical account, timeout, dan path protection.
- [ ] Standard response helper sesuai `docs/openapi.yaml`.
- [ ] Middleware recovery, request ID, logger, CORS, dan body limit tersedia.
- [ ] Protected route tidak pernah didaftarkan secara fail-open.
- [ ] `GET /health` menghasilkan 200 saat PostgreSQL/Redis sehat.
- [ ] `GET /health` menghasilkan 503 saat dependency wajib gagal.
- [ ] API dan worker mendukung graceful shutdown.
- [ ] Migration memakai tool eksplisit, bukan auto-migrate.
- [ ] Dockerfile menjalankan binary sebagai non-root.
- [ ] Docker Compose valid dan service non-public berada di internal network.
- [ ] `go vet ./...`, `go test ./...`, linter, dan build lulus.
- [ ] Unit test response, middleware, health, config, dan path traversal lulus.
- [ ] Tidak ada secret, credential, data personal, atau debug code dalam diff.

## Files yang Akan Dibuat atau Disesuaikan

Struktur final mengikuti library yang disetujui, dengan target minimum:

```text
backend/
├── cmd/
│   ├── api/main.go
│   └── worker/main.go
├── internal/
│   ├── config/
│   │   ├── config.go
│   │   └── config_test.go
│   ├── middleware/
│   │   ├── recovery.go
│   │   ├── request_id.go
│   │   ├── access_log.go
│   │   ├── cors.go
│   │   ├── body_limit.go
│   │   └── identity.go
│   ├── platform/
│   │   ├── logger/
│   │   ├── postgres/
│   │   ├── redis/
│   │   └── nextcloud/
│   ├── http/
│   │   ├── router/
│   │   └── response/
│   ├── validation/
│   └── worker/
├── migrations/
├── tests/
├── .env.example
├── Dockerfile
├── Makefile
└── go.mod

docs/
└── architecture/
    └── backend-stack-proposal.md  # hanya jika keputusan stack belum ada

docker-compose.yml
```

Jangan membuat file kosong yang tidak memiliki kontrak atau implementasi nyata.
