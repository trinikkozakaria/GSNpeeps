# GSNpeeps

GSNpeeps adalah fondasi HRIS untuk employee database, attendance, leave, overtime,
approval, notification, role/permission, dan audit. Kontrak HTTP berada di
`docs/openapi.yaml`.

## Stack

- Frontend: React/JSX, Vite, React Router, Tailwind CSS, Axios, TanStack Query,
  React Hook Form, Zod, Vitest, Testing Library, Playwright, pnpm.
- Backend: Go, `net/http`, Gorilla Mux, pgx, Goose, validator, JWT v5, Redis,
  `slog`, dan Testify.
- Runtime: Nginx, PostgreSQL 16, Redis 7, dan Nextcloud.

## Menjalankan frontend

```powershell
Set-Location frontend
pnpm.cmd install
pnpm.cmd dev
```

Vite berjalan pada `http://localhost:5173`. Untuk koneksi API lokal, jalankan
backend pada port `8080`.

## Menjalankan backend

Salin `backend/.env.example` menjadi `backend/.env`, isi secret lokal, lalu ekspor
nilainya ke environment shell sebelum menjalankan:

```powershell
Set-Location backend
go mod download
go run ./cmd/api
```

Endpoint fondasi:

```text
GET http://localhost:8080/health
```

Health bernilai `200` hanya ketika PostgreSQL dan Redis dapat diakses.

Migration Auth/RBAC dijalankan dengan:

```powershell
Set-Location backend
make migrate-up
make seed
```

Seeder hanya menerima `APP_ENV=development` atau `test` dan membaca password sintetis
dari `SEED_PASSWORD`.

## Menjalankan seluruh stack

Salin `.env.example` menjadi `.env`, ganti semua placeholder secret, lalu:

```powershell
docker compose config
docker compose up --build
```

Hanya Nginx dipublikasikan ke host pada `http://localhost:8080`. Backend,
PostgreSQL, Redis, worker, frontend container, dan Nextcloud berada di network
internal.

Migration Goose dijalankan otomatis sebelum API/worker. Untuk membuat empat akun sintetis
development secara idempotent:

```powershell
docker compose --profile tools run --rm seed
```

Email fixture memakai domain `example.test`; passwordnya adalah nilai `SEED_PASSWORD`
lokal dan tidak ditulis ke repository.

## Session frontend

Bearer token disimpan di cookie `gsnpeeps_session` yang ditulis dari browser (backend belum
mengirim `Set-Cookie`), dengan `SameSite=Strict`, `Secure` saat HTTPS, dan `Max-Age`
mengikuti sisa masa berlaku token. Setelah reload, token dipulihkan dari cookie lalu
diverifikasi ulang ke `GET /auth/me` sebelum sesi dianggap valid. Token belum bisa
`HttpOnly` karena masih dibaca JavaScript untuk header `Authorization`. Keputusan dan
trade-off lengkap ada di `docs/architecture/frontend-auth-session.md`.

Nextcloud pada Compose foundation menggunakan penyimpanan database bawaannya agar
stack lokal dapat diinisialisasi tanpa database kedua. Sebelum production,
konfigurasikan database Nextcloud terpisah dan app password technical account.

## Quality gate

```powershell
Set-Location backend
gofmt -w .
go vet ./...
go test ./...
go build ./...

Set-Location ../frontend
pnpm.cmd test
pnpm.cmd build
pnpm.cmd test:e2e
```

Browser Playwright perlu dipasang satu kali dengan
`pnpm.cmd exec playwright install chromium`.
