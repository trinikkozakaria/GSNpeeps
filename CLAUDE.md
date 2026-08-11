# CLAUDE.md — GSNpeeps

> Panduan untuk Claude Code saat mengerjakan project ini.
> **Scope fase ini**: Employee Database, Dashboard HR, Absensi, Approval, Notifikasi, User Management, dan RBAC.

---

## 1. Konteks Project

**Nama produk**: GSNpeeps  
**Jenis**: Aplikasi dashboard internal berbasis web yang menyerupai HRIS untuk pengelolaan data karyawan, metrik HR, absensi, dan approval.  
**Target dokumen**: 28 Agustus 2026.  
**Bahasa komunikasi**: **Bahasa Indonesia** untuk penjelasan, komentar domain, pesan validasi, dan pesan end-user. Identifier kode, nama tabel, endpoint, dan commit tetap menggunakan bahasa Inggris.

### Tujuan

- Menjadi single source of truth data karyawan.
- Menyediakan dashboard metrik HR untuk pengambilan keputusan.
- Menyediakan absensi foto dengan mode WFO, WFH, dan WFA.
- Mengotomasi pengajuan dan approval Ketidakhadiran serta Lembur.
- Membatasi data dan aksi berdasarkan role dan hubungan organisasi.
- Menyediakan notifikasi in-app yang idempotent.

### Struktur Organisasi

```text
Perusahaan
├── Departemen (N)
│   ├── Jabatan/Posisi (N)
│   └── Karyawan (N)
└── Karyawan
    └── Atasan langsung (self-reference, nullable)
```

`employees.atasan_id` menentukan bawahan langsung dan routing approval tahap Atasan.

### Roles Sistem

- `karyawan` — melihat data sendiri, metrik personal, absensi, dan pengajuan sendiri.
- `atasan` — seluruh kemampuan Karyawan ditambah approval bawahan langsung.
- `hr` — CRUD data karyawan, dashboard, monitoring, final approval, master izin, Role, Permission, dan Audit Log.
- `top_management` — monitoring read-only seluruh menu termasuk AKSES, ditambah final approval pengajuan milik HR. Hanya satu user dan tidak memiliki Metrik Personal.

---

## 2. Sumber Kebenaran

Gunakan urutan berikut ketika ada konflik:

1. Instruksi terbaru pengguna.
2. PRD v1.2.
3. User Story v1.2.
4. API Contract v1.1 untuk endpoint, payload, response, dan HTTP status.
5. Database Schema v1.1 untuk tabel, tipe, constraint, dan index.
6. Sequence Diagram v1.1 untuk urutan interaksi.
7. System Design v1.0 untuk arsitektur dan deployment.
8. Ringkasan `.claude/specs/`.
9. Kode dan test yang sudah ada.

Lokasi dan versi dokumen tercatat di `.claude/specs/document-index.md`.

Jika sumber bertentangan, jangan memilih diam-diam. Catat konflik, ikuti sumber dengan otoritas tertinggi, dan minta keputusan bila hasilnya mengubah kontrak produk.

Nama produk repository ini selalu **GSNpeeps**. `janjikupadamu.id` adalah kandidat domain deployment dari dokumen, bukan nama produk.

---

## 3. Tech Stack

### Backend

- **Language**: Go.
- **HTTP**: standard library `net/http` + `github.com/gorilla/mux`.
- **Database access**: `github.com/jackc/pgx/v5` (termasuk `pgxpool`).
- **Migration**: `github.com/pressly/goose/v3`.
- **Validation**: `github.com/go-playground/validator/v10`.
- **JWT**: `github.com/golang-jwt/jwt/v5`.
- **Redis client**: `github.com/redis/go-redis/v9`.
- **Logging**: standard library `log/slog`.
- **Test assertion/mock helper**: `github.com/stretchr/testify`.
- **API style**: REST + JSON, tanpa web framework.
- **Authentication**: JWT dengan masa berlaku 8 jam.
- **Database**: PostgreSQL 16.
- **Cache/session/rate limit**: Redis 7.
- **File integration**: Nextcloud self-hosted melalui WebDAV.
- **Worker**: binary/container terpisah dengan codebase yang sama seperti API.

### Frontend

- **Framework/library**: React.
- **Language**: JavaScript dengan file `.js`/`.jsx`; jangan menambahkan TypeScript.
- **Bundler/dev server**: Vite.
- **Package manager**: pnpm; `pnpm-lock.yaml` adalah lockfile tunggal.
- **Router**: React Router.
- **Styling**: Tailwind CSS.
- **HTTP client**: Axios.
- **Server state**: TanStack Query.
- **Form**: React Hook Form.
- **Schema/validation**: Zod.
- **Unit/component test**: Vitest + Testing Library.
- **End-to-end test**: Playwright.
- **Delivery**: production static assets disajikan melalui Nginx.

### Infrastructure

- **Reverse proxy**: Nginx.
- **TLS**: Let's Encrypt/Certbot.
- **Container**: Docker + Docker Compose.
- **Services**: `nginx`, `frontend`, `backend-api`, `cron-worker`, `postgres`, `redis`, `nextcloud`.
- Hanya Nginx yang boleh mengekspos port publik. Service lain berada di Docker internal network.

### Batas Pemilihan Stack

- Stack di atas adalah baseline final keputusan produk tanggal 29 Juli 2026.
- Jangan menambahkan Next.js, TypeScript, Gin, Echo, GORM, sqlc, npm/yarn lockfile, Jest,
  atau HTTP/query/form/router alternatif.
- Library yang belum dipilih dan bukan bagian baseline (misalnya chart, data-table, icon,
  mocking network, client-state global, scheduler, export, dan linter tambahan) dipilih hanya
  saat kebutuhan nyata muncul dan dicatat tanpa mengganti baseline.

---

## 4. Struktur Monorepo

Gunakan struktur target berikut sesuai stack final.

```text
GSNpeeps/
├── backend/
│   ├── cmd/
│   │   ├── api/main.go
│   │   ├── worker/main.go
│   │   └── seed/main.go
│   ├── internal/
│   │   ├── config/
│   │   ├── domain/
│   │   ├── dto/
│   │   ├── validation/
│   │   ├── repository/
│   │   ├── service/
│   │   ├── handler/
│   │   ├── middleware/
│   │   ├── router/
│   │   ├── worker/
│   │   ├── platform/
│   │   │   ├── postgres/
│   │   │   ├── redis/
│   │   │   ├── nextcloud/
│   │   │   ├── password/
│   │   │   ├── jwt/
│   │   │   ├── export/
│   │   │   └── logger/
│   │   └── pkg/
│   │       └── response/
│   ├── migrations/
│   ├── seeds/
│   ├── tests/
│   ├── .env.example
│   ├── Dockerfile
│   ├── Makefile
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/
│   │   ├── app/
│   │   ├── routes/
│   │   ├── modules/
│   │   ├── components/
│   │   ├── lib/
│   │   │   └── api/
│   │   ├── hooks/             # hook lintas modul saja
│   │   ├── stores/            # hanya jika global client-state disetujui
│   │   ├── schemas/           # schema shared; resource schema tetap di modul
│   │   ├── mocks/             # test/dev only, tidak masuk production
│   │   └── styles/
│   ├── public/
│   ├── .env.example
│   ├── Dockerfile
│   ├── package.json
│   ├── pnpm-lock.yaml
│   └── vite.config.js
├── docs/
├── .claude/
├── docker-compose.yml
└── CLAUDE.md
```

Jangan membuat folder ad-hoc yang menduplikasi fungsi folder di atas.

---

## 5. Konvensi Kode

### Backend Go

- Gunakan dependency direction: `handler -> service/use-case -> repository/integration`.
- Handler menangani decode, validasi request, pemanggilan use-case, dan mapping response.
- Service/use-case menangani aturan bisnis dan batas transaksi.
- Repository menangani akses PostgreSQL.
- Integrasi Redis dan Nextcloud harus berada di balik interface.
- Setiap method service dan repository menerima `context.Context`.
- Gunakan parameterized query. Jangan gabungkan input user ke SQL.
- Jangan memakai `SELECT *`; sebut kolom secara eksplisit.
- Gunakan explicit transaction untuk perubahan lintas tabel.
- Wrap error dengan konteks tanpa membocorkan SQL, credential, atau internal stack ke response.
- Jangan memakai panic sebagai kontrol alur.
- Gunakan structured logging dan request ID.
- Jangan mencatat password, token, foto, dokumen, atau data personal sensitif ke log.
- Inject clock, ID generator, dan integration adapter untuk test yang deterministik.

### Standard API Response

```json
{
  "success": true,
  "data": {},
  "message": "Deskripsi singkat hasil"
}
```

Endpoint list menambahkan:

```json
{
  "meta": {
    "page": 1,
    "limit": 20,
    "total_data": 134,
    "total_page": 7
  }
}
```

Error:

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Data belum valid",
    "fields": {
      "email": "Format email tidak valid"
    }
  }
}
```

`error.fields` hanya digunakan untuk validation error yang membutuhkan detail per field.

Nilai status pada database, query, dan response API wajib memakai enum
lowercase/underscore yang sama. Label Bahasa Indonesia hanya dibuat di frontend; jangan
mengirim label tampilan sebagai nilai status API.

### Frontend React

- Organisasikan kode berdasarkan modul bisnis di `src/modules`, dengan satu folder untuk
  setiap modul seperti `auth`, `dashboard`, `employees`, `profile`, `attendance`, `leave`,
  `overtime`, `notifications`, dan `access`.
- Di dalam setiap modul, pisahkan `pages`, `components`, `hooks`, `api`, `schemas`, `utils`,
  dan `tests` sesuai kebutuhan. Jangan membuat folder kosong hanya untuk memenuhi template.
- Simpan komponen yang hanya dipakai satu modul di dalam modul tersebut. `src/components`
  hanya untuk primitive, form adapter, layout, dan feedback yang benar-benar lintas modul.
- Simpan transport HTTP, normalisasi error, response envelope, dan konfigurasi query bersama
  di `src/lib`; endpoint suatu resource tetap berada di `src/modules/<module>/api`.
- Gunakan React Router di `src/routes`; `src/app` hanya untuk bootstrap/provider composition,
  bukan file-based routing. Jangan membuat struktur Next.js `src/app/(group)`.
- Jika tampilan berbeda menurut role, letakkan variasinya di dalam modul pemilik, misalnya
  `modules/dashboard/components/hr` dan `modules/dashboard/components/employee`; jangan
  menduplikasi seluruh struktur aplikasi per role.
- `dist`, `node_modules`, output Playwright, coverage, dan log debug adalah generated artifacts
  dan tidak boleh di-commit.
- Gunakan Axios melalui satu instance di `src/lib/api`, TanStack Query untuk server state,
  React Hook Form + Zod untuk form, serta Vitest/Testing Library dan Playwright untuk test.
- Centralize API base URL, Bearer token, response envelope, error mapping, dan multipart upload.
- Jangan mendefinisikan request/response shape berdasarkan tebakan komponen; ikuti API Contract.
- Server state tidak boleh diduplikasi tanpa alasan ke global client state.
- Frontend route guard dan hidden menu hanya UX; backend tetap sumber otorisasi.
- Gunakan label yang terlihat, semantic controls, focus state, dan status yang tidak bergantung pada warna saja.
- Sediakan loading, empty, validation, success, forbidden, conflict, rate-limit, dan server-error state.
- Cegah double submit ketika mutation berlangsung.
- Buat tabel responsif dan pertahankan filter/pagination di URL bila memungkinkan.
- Jangan render atau cache data sensitif sebelum hak akses dipastikan.

### Naming

- Package Go: lowercase dan ringkas.
- Struct/type public: PascalCase.
- Variable/function private: camelCase.
- Komponen React: PascalCase.
- Hook: prefix `use`.
- Database/API identifier: English snake_case sesuai kontrak.
- Teks antarmuka: Bahasa Indonesia.

### Git Commit

Gunakan Conventional Commits:

```text
feat(auth): add redis-backed session validation
feat(employee): add soft-delete endpoint
feat(attendance): enforce WFO office radius
fix(approval): prevent duplicate final decisions
docs(claude): align project instructions
```

---

## 6. Schema Database

### Konvensi

- Seluruh primary key memakai UUID dari `gen_random_uuid()`.
- Foreign key memakai `ON DELETE RESTRICT` secara default.
- Tabel histori menggunakan soft-delete bila ditentukan schema.
- File fisik tidak disimpan di PostgreSQL; hanya URL/path Nextcloud.
- Hampir semua tabel menggunakan `created_at` dan `updated_at`.
- Status/tipe dibatasi dengan PostgreSQL enum atau `CHECK`.
- Migration harus eksplisit dan dapat direview. Jangan mengandalkan auto-migrate di production.

### 26 Tables

#### Organisasi dan akun

- `office_locations`
- `departments`
- `positions`
- `roles`
- `employees`
- `users`

#### Detail data karyawan

- `employee_addresses`
- `employee_ktp`
- `employee_contracts`
- `employee_bpjs`
- `employee_npwp`
- `employee_emergency_contacts`
- `employee_education`
- `employee_position_history`
- `employee_salaries`
- `employee_documents`

#### Kehadiran dan ketidakhadiran

- `attendances`
- `leave_types`
- `leave_balances`
- `leave_requests`
- `leave_approvals`

#### Lembur

- `overtime_requests`
- `overtime_approvals`

#### Akses dan audit

- `permissions`
- `audit_logs`

#### Notifikasi

- `notifications`

### Constraint Penting

- `employees.nip`, `users.email`, nomor identitas relevan, dan nomor kontrak harus unik sesuai schema.
- `office_locations.code` unik; koordinat menjadi sumber tepercaya untuk validasi WFO.
- Karyawan bebas memilih lokasi kantor aktif saat WFO; tidak ada assignment kantor permanen.
- `employees.atasan_id` adalah self-reference nullable.
- Employee delete adalah soft-delete: set `status='nonaktif'` dan `deleted_at`.
- `employee_salaries` unik per `(employee_id, periode)`.
- `leave_balances` unik per `(user_id, tahun)`.
- `notifications` unik per `(recipient_user_id, event_key)`.
- `audit_logs` append-only; DB user aplikasi tidak boleh memiliki privilege UPDATE/DELETE pada tabel ini.

Baca Database Schema v1.1 sebelum menulis tipe kolom, constraint, index, dan migration lengkap.

---

## 7. API Endpoints

Base URL dokumen:

```text
https://api.janjikupadamu.id/api/v1
```

Base path aplikasi:

```text
/api/v1
```

Total: **47 endpoint dalam 13 modul**.

### Sistem

| Method | Path | Akses |
|---|---|---|
| GET | `/health` | Public |

### Authentication

| Method | Path | Akses |
|---|---|---|
| POST | `/auth/login` | Public |
| POST | `/auth/reset-password` | Public, rate-limited, verifikasi password saat ini |
| POST | `/auth/logout` | Semua role terautentikasi |
| GET | `/auth/me` | Semua role terautentikasi |
| PATCH | `/auth/me/password` | Semua role terautentikasi |

### Master Organisasi

| Method | Path | Akses |
|---|---|---|
| GET | `/master/departemen` | Semua role |
| GET | `/master/jabatan` | Semua role |
| GET | `/master/lokasi-kantor` | Semua role |

### Data Karyawan

| Method | Path | Akses |
|---|---|---|
| GET | `/karyawan` | HR, Top Management read-only |
| GET | `/karyawan/{id}` | HR, Top Management read-only |
| POST | `/karyawan` | HR |
| PUT | `/karyawan/{id}` | HR |
| DELETE | `/karyawan/{id}` | HR, soft-delete |
| GET | `/karyawan/export` | HR |
| POST | `/karyawan/{id}/dokumen` | HR |
| GET | `/karyawan/{id}/dokumen` | HR, Top Management read-only bila kontrak mengizinkan |

### Profil dan Metrik Personal

| Method | Path | Akses |
|---|---|---|
| GET | `/profil/saya` | Karyawan, Atasan, HR |
| GET | `/profil/saya/metrik` | Karyawan, Atasan, HR |

### Dashboard

| Method | Path | Akses |
|---|---|---|
| GET | `/dashboard/metrik` | HR, Top Management read-only |

### Kehadiran

| Method | Path | Akses |
|---|---|---|
| POST | `/absensi/checkin` | Karyawan, Atasan, HR |
| GET | `/absensi/livefeed` | HR, Top Management read-only |

### Laporan Kehadiran

| Method | Path | Akses |
|---|---|---|
| GET | `/laporan/kehadiran` | HR, Top Management read-only |
| GET | `/laporan/kehadiran/export` | HR |

### Ketidakhadiran

| Method | Path | Akses |
|---|---|---|
| POST | `/ketidakhadiran` | Karyawan, Atasan, HR |
| GET | `/ketidakhadiran` | Approver terkait |
| GET | `/ketidakhadiran/{id}` | Pemohon/approver terkait |
| GET | `/ketidakhadiran/saya` | Karyawan, Atasan, HR |
| PUT | `/ketidakhadiran/{id}/decision` | Approver pada tahap aktif |
| PUT | `/ketidakhadiran/{id}/delegate` | Atasan terkait |

### Master Jenis Izin

| Method | Path | Akses |
|---|---|---|
| GET | `/master/jenis-izin` | Semua role; non-HR hanya jenis izin aktif |
| POST | `/master/jenis-izin` | HR |
| PUT | `/master/jenis-izin/{id}` | HR |

### Lembur

| Method | Path | Akses |
|---|---|---|
| POST | `/lembur` | Karyawan, Atasan, HR |
| GET | `/lembur` | Approver terkait |
| GET | `/lembur/saya` | Karyawan, Atasan, HR |
| GET | `/lembur/{id}` | Pemohon/approver terkait |
| PUT | `/lembur/{id}/decision` | Approver pada tahap aktif |
| GET | `/lembur/rekap` | HR, Top Management read-only |

### Akses

| Method | Path | Akses |
|---|---|---|
| GET | `/akses/role` | HR, Top Management read-only |
| GET | `/akses/permission` | HR, Top Management read-only |
| PUT | `/akses/permission` | HR |
| GET | `/akses/audit-log` | HR, Top Management read-only |

### Notifikasi

| Method | Path | Akses |
|---|---|---|
| GET | `/notifikasi` | Recipient sendiri |
| GET | `/notifikasi/unread-count` | Recipient sendiri |
| PUT | `/notifikasi/{id}/read` | Recipient sendiri |
| DELETE | `/notifikasi/{id}` | Recipient sendiri, soft-dismiss |

Kontrak aktif 0.5.0 mempertahankan current-user dan change-password, serta menyediakan
self-reset yang memverifikasi password saat ini. Revisi 0.5.0 menambahkan `GET
/lembur/saya` (D-035) agar histori lembur milik sendiri tidak lagi memakai endpoint inbox
approval. Jangan menambahkan refresh token atau mekanisme forgot-password berbasis
email/OTP tanpa keputusan produk.

---

## 8. Authentication, RBAC, dan Security

### JWT dan Session

- JWT berlaku 8 jam.
- Claim minimal: `user_id`, `role`, dan `exp`.
- Token aktif di-cross-check melalui Redis key `session:<user_id>`.
- Logout atau lockout harus langsung menginvalidasi session Redis.
- Password disimpan sebagai bcrypt/Argon2 hash, tidak pernah plaintext.

### Login Lockout

- Hitung kegagalan per akun.
- Lima kegagalan berturut-turut mengunci akun.
- Login sukses atau reset password yang valid mereset counter.
- Self-reset menerima email, password saat ini, password baru, dan konfirmasi. Setelah berhasil,
  akun dibuka, seluruh session dicabut, dan pengguna wajib login ulang.
- Self-reset memakai error generik, rate limit gabungan akun+IP, dan counter kegagalan yang sama
  dengan login agar tidak menjadi bypass brute-force.
- HR hanya melihat status akun yang sudah diperbarui; password lama maupun baru tidak pernah
  dikirim atau ditampilkan kepada HR.
- Lupa password tanpa mengetahui password saat ini belum termasuk scope sampai kanal verifikasi
  email/OTP disetujui.
- Kondisi terkunci menggunakan `429 ACCOUNT_LOCKED` sesuai API Contract.

### Rate Limit

- Login: batas lima kegagalan per akun dalam rolling window.
- Endpoint lain: 120 request/menit per user.
- Counter disimpan di Redis.

### Authorization

- Role diambil dari identitas server-side, bukan request body.
- Selalu cek role, ownership, hubungan bawahan langsung, dan tahap approval.
- Atasan hanya boleh melihat/memutus pengajuan bawahan langsung.
- Pemohon hanya boleh melihat pengajuan sendiri kecuali juga menjadi approver.
- Top Management read-only kecuali decision pengajuan milik HR.
- Frontend permission check hanya UX. Backend tetap sumber kebenaran.

### File Security

- Browser mengunggah file ke Backend API.
- Backend memvalidasi ukuran, ekstensi, MIME type, dan file signature.
- Maksimum 5 MB per file.
- Backend mengunggah ke Nextcloud melalui akun teknis/app password.
- Credential Nextcloud tidak pernah dikirim ke frontend.
- PostgreSQL hanya menyimpan URL/path.

### Audit

- Catat login, logout, create, update, delete, approve, reject, download, dan permission change.
- Sertakan actor, waktu, modul, data ID, perubahan relevan, dan IP bila tersedia.
- Jangan mencatat secret atau isi dokumen.
- Audit Log tidak dapat diedit atau dihapus.

---

## 9. Workflow Bisnis

### Approval Ketidakhadiran dan Lembur

```text
Karyawan dengan Atasan -> Atasan -> HR
Karyawan tanpa Atasan  -> HR
Atasan mengajukan      -> HR
HR mengajukan          -> Top Management
```

- Reject mengakhiri alur dan mewajibkan catatan.
- Approve Atasan memindahkan pengajuan ke HR.
- Atasan dapat mendelegasikan keputusan ke HR.
- SLA 2x24 jam dan auto-escalation hanya berlaku dari tahap Atasan ke HR.
- Tidak ada SLA auto-escalation dari HR ke Top Management.
- Gunakan transaksi dan row lock/conditional update untuk mencegah dua keputusan.
- Request yang sudah diputus menghasilkan `409 ALREADY_DECIDED`.

### Ketidakhadiran

- Mencakup Cuti, Izin, dan Perjalanan Dinas.
- Dokumen pendukung wajib untuk semua jenis.
- Perjalanan Dinas juga mewajibkan lokasi tujuan dan keperluan tugas.
- Approval fase ini hanya terkait kehadiran; budget menunggu Benefit.

### Lembur

- Dokumen pendukung opsional.
- Durasi tersimpan, tetapi kompensasi/uang lembur dihitung manual di luar sistem.

### Kehadiran

- Mode kerja: WFO, WFH, WFA.
- Radius 100 meter hanya untuk WFO.
- WFH dan WFA tidak dibatasi radius kantor.
- Koordinat wajib untuk semua mode.
- Sistem memakai master `office_locations`. Saat WFO, karyawan memilih kantor aktif dan backend
  mengambil koordinat tepercaya; tidak ada assignment kantor permanen.
- Alamat dan koordinat resmi akan di-seed kemudian. Jangan membuat lokasi fiktif.
- Waktu server/network adalah sumber kebenaran.
- Waktu lokal hanya digunakan pada watermark.
- Hari kerja reguler Senin-Jumat, 09.00-18.00 WIB (`Asia/Jakarta`).
- Check-in tepat 09.00 tidak terlambat; setelah 09.00 berstatus `terlambat`.
- Checkout sebelum 18.00 tetap valid dan dicatat `pulang_cepat`.
- Cegah check-in ganda dan checkout tanpa check-in.
- Sediakan fallback upload foto berwatermark bila kamera live gagal.
- Tidak ada reminder absensi.

### Dashboard Metrics

- Filter memakai periode harian, mingguan, bulanan, atau tahunan dengan tanggal acuan.
- Seluruh boundary kalender dan jam kerja menggunakan Asia/Jakarta; minggu adalah Senin-Minggu.
- Kehadiran hanya berasal dari check-in valid.
- Jam masuk adalah 09:00:00 WIB; hanya waktu setelah 09:00:00 yang terlambat.
- Hitungan dan komposisi departemen karyawan nonaktif dipisahkan dari karyawan aktif.
- Gender kosong masuk kategori `belum_diisi`, bukan laki-laki atau perempuan.

### Notifikasi

- Dibuat server-side saat event terjadi, bukan melalui endpoint create publik.
- Setiap recipient/event memiliki `event_key` deterministik.
- `UNIQUE(recipient_user_id, event_key)` menegakkan idempotensi.
- Dismiss mengisi `dismissed_at`, bukan hard-delete.
- Event yang sudah di-dismiss tidak boleh dibuat ulang.
- Pengajuan baru dikirim ke approver aktif.
- Perubahan status dikirim ke pemohon dan approver tahap berikutnya.
- Kontrak H-30 berarti tepat 30 hari kalender sebelum tanggal berakhir berdasarkan
  timezone Asia/Jakarta.
- Penerima HR adalah semua HR aktif selain karyawan yang kontraknya akan habis.
- Jika tidak ada HR aktif lain, kirim ke satu-satunya Top Management aktif.
- Deduplikasi penerima berdasarkan user ID; jangan pernah self-notify.

---

## 10. Scheduler Jobs

| Job | Frekuensi | Hasil |
|---|---|---|
| Notifikasi kontrak H-30 | Harian | Insert idempotent untuk atasan aktif + semua HR aktif selain subjek; fallback satu Top Management |
| Auto-escalation approval | Beberapa menit/cron | Pindahkan request `menunggu_atasan` lebih dari 48 jam ke HR |
| Retensi foto absensi | Harian | Hapus file Nextcloud lebih dari 3 bulan dan set `foto_url = NULL` |

Scheduler memakai image/codebase Backend yang sama, tetapi command dan container berbeda.

Setiap job harus aman dijalankan ulang, memiliki structured log, dan tidak menduplikasi efek.

---

## 11. Alur Development

### Prioritas

1. Foundation monorepo, Docker Compose, config, healthcheck, logging, dan migration runner.
2. Database schema organisasi, akun, seed empat role.
3. Authentication, Redis session, lockout, dan RBAC.
4. Employee Database, dokumen, Profil Saya, Metrik Personal, dan Dashboard.
5. Kehadiran, live feed, laporan, export, dan retensi foto.
6. Ketidakhadiran dan Lembur beserta approval, delegasi, dan escalation.
7. Notifikasi, Role/Permission, dan Audit Log.
8. Integrasi frontend-backend, end-to-end test, security review, dan release readiness.

Task detail ada di `.claude/skills/1.TASK.md`.

### Alur Tiap Sesi AI

1. Baca file ini.
2. Baca `.claude/skills/1.TASK.md`.
3. Pilih task pertama yang belum selesai.
4. Muat skill:
   - Backend: `.claude/skills/hris-backend/SKILL.md`
   - Frontend: `.claude/skills/hris-frontend/SKILL.md`
   - Git/review: `.claude/skills/hris-git/SKILL.md`
5. Baca PDF dan spec yang relevan.
6. Tulis scope, non-scope, acceptance criteria, file target, dan test.
7. Implementasikan vertical slice terkecil yang lengkap.
8. Jalankan quality gate.
9. Laporkan file berubah, hasil verifikasi, asumsi, dan pekerjaan tersisa.

### Kapan Minta Klarifikasi

- Ketika field atau endpoint tidak ada di dokumen.
- Ketika ada konflik antar dokumen yang mengubah behavior.
- Ketika harus memilih dependency optional di luar baseline dan dampaknya material.
- Ketika tindakan bersifat destruktif atau menyentuh production.

Jangan meminta klarifikasi untuk pilihan implementasi kecil yang sudah dibatasi jelas oleh stack dan pola repository.

### Yang Tidak Boleh Dilakukan

- Jangan melewati migration dengan auto-migrate production.
- Jangan commit `.env`, credential, token, app password, data karyawan nyata, atau database dump.
- Jangan hardcode connection string atau secret.
- Jangan membuat endpoint tanpa authentication/authorization yang sesuai.
- Jangan mengandalkan hidden menu sebagai security.
- Jangan mengembalikan `password_hash`.
- Jangan hard-delete employee, notification, atau record histori yang ditentukan soft-delete.
- Jangan update/delete Audit Log.
- Jangan membuat reminder absensi.
- Jangan membuat kalkulasi kompensasi lembur.
- Jangan mengimplementasikan Hiring Progress, Recruitment Cost, atau Benefit selain placeholder Coming Soon.
- Jangan menjalankan force push, hard reset, destructive migration, atau operasi production tanpa izin eksplisit.

---

## 12. Environment Variables

### Backend dan Worker

```dotenv
APP_ENV=development
APP_PORT=8080
APP_BASE_URL=http://localhost:8080

DATABASE_URL=postgres://gsnpeeps:change-me@postgres:5432/gsnpeeps?sslmode=disable
REDIS_URL=redis://redis:6379/0

JWT_SECRET=change-me
JWT_TTL=8h

NEXTCLOUD_BASE_URL=http://nextcloud/remote.php/dav/files/gsnpeeps
NEXTCLOUD_USER=gsnpeeps-service
NEXTCLOUD_APP_PASSWORD=change-me

CORS_ALLOWED_ORIGIN=http://localhost:5173
RATE_LIMIT_LOGIN=5
RATE_LIMIT_DEFAULT=120
LOG_LEVEL=debug
```

### Frontend

```dotenv
APP_NAME=GSNpeeps
API_BASE_URL=http://localhost:8080/api/v1
```

Environment frontend yang boleh masuk bundle memakai prefix `VITE_`. Jangan mengekspos secret
dengan prefix tersebut.

Nilai di atas hanya placeholder development untuk `.env.example`; jangan gunakan credential contoh sebagai production secret.

---

## 13. Perintah Umum

Gunakan command baseline berikut atau script repository yang ekuivalen:

### Backend

```text
dev
build
test
vet
lint
fmt
goose up
goose down
goose create
seed
worker
```

### Frontend

```text
pnpm run dev
pnpm run build
pnpm run lint
pnpm run format
pnpm run test
pnpm run test:e2e
```

### Root

```text
docker compose up
docker compose config
docker compose down
```

Gunakan pnpm untuk seluruh command frontend; jangan membuat npm/yarn lockfile.

---

## 14. Testing Strategy

### Backend

- Unit test untuk validation, service/use-case, authorization, status transition, dan event key.
- Repository/integration test memakai PostgreSQL nyata/container untuk constraint, transaction, pagination, dan soft-delete.
- Handler test untuk success envelope dan seluruh error khusus endpoint.
- Concurrency test untuk dua decision pada request yang sama.
- Scheduler repeat-run test untuk idempotensi.
- Storage failure test untuk upload dan orphan cleanup.
- Pastikan forbidden request tidak menimbulkan database, storage, notification, atau audit side effect.

### Frontend

- Unit test untuk permission helpers, validation schema, utility, dan hooks.
- Component test untuk form, table, notification center, dan approval controls.
- Route test untuk seluruh role dan direct forbidden URL.
- Test loading, empty, validation, 401, 403, 409, 422, 429, dan 500.
- Test camera denied, geolocation denied, fallback upload, WFO out-of-radius, duplicate check-in, dan checkout without check-in.
- Test keyboard navigation, focus handling, label, mobile layout, dan semantic status.

### End-to-End

- Login, logout, dan lockout.
- HR CRUD employee serta soft-delete.
- Profil dan metrik personal.
- Check-in/out WFO, WFH, WFA.
- Seluruh jalur approval berdasarkan role.
- Delegasi dan auto-escalation.
- Notifikasi unread/read/dismiss/deep link.
- Permission administration dan Audit Log read-only.
- Export employee dan attendance.
- Negative authorization untuk keempat role.

---

## 15. Definition of Done

Sebuah fitur dianggap selesai bila:

- Scope dan acceptance criteria task terpenuhi.
- Endpoint mengikuti API Contract dan memiliki validasi serta authorization.
- Migration up dan rollback yang aman tersedia bila schema berubah.
- Constraint, index, transaction, soft-delete, dan audit sesuai Database Schema.
- Unit/integration test relevan tersedia dan lulus.
- Frontend memiliki loading, empty, success, validation, forbidden, conflict, dan server-error state yang relevan.
- UI responsif dan dapat digunakan dengan keyboard.
- Format, lint, vet, test, build, dan `docker compose config` lulus sesuai area perubahan.
- Tidak ada secret, fixture PII, debug route, atau unresolved critical failure.
- Dokumentasi kontrak dan environment diperbarui bila diperlukan.
- Laporan akhir menyebut perubahan, command verifikasi, hasil, asumsi, migration impact, dan known limitations.

Jangan menandai task selesai hanya karena happy path berjalan.

---

## 16. Referensi Internal

- `.claude/specs/document-index.md` — versi dan lokasi enam PDF sumber.
- `.claude/specs/product-requirements.md` — requirement inti.
- `.claude/specs/access-matrix.md` — matriks role.
- `.claude/specs/api-data-summary.md` — ringkasan API dan tabel.
- `.claude/specs/workflows.md` — approval, notification, dan attendance invariants.
- `.claude/skills/` — task dan reusable implementation guidance.
- `.claude/prompts/` — prompt bertahap backend, frontend, dan integration.

---

## 17. Catatan untuk Claude Code

- Selalu baca file ini dan dokumen terkait sebelum memulai task.
- Prioritaskan kebenaran kontrak, security, dan konsistensi data daripada kecepatan.
- Jangan menebak detail bisnis.
- Gunakan nama **GSNpeeps** di seluruh UI dan dokumentasi baru.
- Simpan output pada lokasi yang sesuai struktur monorepo.
- Jangan mengubah file di luar scope tanpa alasan yang dijelaskan.
- Pertahankan perubahan pengguna yang tidak terkait.
- Tulis commit dalam format Conventional Commits.
- File ini adalah living document; perbarui jika keputusan arsitektural baru telah disetujui.
