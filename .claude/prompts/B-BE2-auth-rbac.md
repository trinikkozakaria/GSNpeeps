# Prompt — Epic B-BE.2: Authentication & RBAC

**Agent**: Backend  
**Branch**: `feat/be-auth-rbac`  
**Estimasi**: 1.5–2 hari  
**Prerequisite**: Epic B-BE.1 selesai dan `docs/openapi.yaml` sudah disetujui

## Prompt untuk Claude Code

```text
Implementasikan Authentication dan fondasi RBAC untuk backend GSNpeeps.

Sebelum mulai:
- Baca `CLAUDE.md`.
- Baca `docs/openapi.yaml` sebagai sumber kebenaran kontrak HTTP.
- Baca `.claude/skills/3.TASK_AUTH_RBAC.md`.
- Baca API Contract v1.1 bagian Authentication, rate limiting, dan header umum.
- Baca Database Schema v1.1 untuk roles, employees, users, permissions, dan audit_logs.
- Baca PRD v1.2 dan `.claude/specs/access-matrix.md` untuk empat role.
- Gunakan skill `.claude/skills/hris-backend/SKILL.md` beserta references terkait.
- Gunakan `.claude/skills/hris-git/SKILL.md` hanya pada tahap Git.
- Pertahankan `net/http` + `gorilla/mux`, pgx, Goose, `slog`,
  go-playground/validator, golang-jwt/jwt/v5, go-redis/v9, dan Testify.

Konteks:
- Nama produk: GSNpeeps.
- Role: `karyawan`, `atasan`, `hr`, `top_management`.
- Hanya ada satu user Top Management.
- JWT berlaku 8 jam.
- JWT minimal berisi `user_id`, `role`, dan `exp`.
- Status token aktif di-cross-check melalui Redis key `session:<user_id>`.
- Logout atau lockout harus langsung membuat token lama tidak berlaku.
- Lima kegagalan login berturut-turut mengunci akun.
- Akun yang terkunci wajib melalui reset password.
- Password disimpan sebagai secure password hash, tidak pernah plaintext.
- Rate limit endpoint selain login: 120 request/menit per user.
- Authorization wajib memeriksa role dan resource scope.

SCOPE ENDPOINT SESUAI OPENAPI:

- POST `/api/v1/auth/login`
- POST `/api/v1/auth/logout`
- GET `/api/v1/auth/me`
- PATCH `/api/v1/auth/me/password`
- POST `/api/v1/auth/reset-password` (public, rate-limited)

NON-SCOPE:

- Refresh token.
- Refresh cookie.
- `/auth/refresh`.
- CRUD user.
- Halaman Profil Saya.

Catatan penting:
- Ikuti self-reset OpenAPI 0.4.0: email + password saat ini + password baru + konfirmasi.
- Self-reset boleh membuka account locked hanya setelah password saat ini terverifikasi.
- Terapkan rate limit gabungan per akun dan IP; kegagalan verifikasi masuk counter yang sama
  dengan login dan selalu memakai error generik.
- Setelah berhasil, reset counter/lock, cabut seluruh session, dan wajibkan login ulang.
- Password tidak pernah tampil kepada HR, response, log, atau audit.
- Forgot-password via email/OTP belum termasuk scope.

Kerjakan dengan urutan:

1. REVIEW KONTRAK DAN SCHEMA

   - Pastikan kelima operasi auth/password contract tersedia dan tidak ada refresh endpoint.
   - Pastikan tidak ada RefreshCookie atau refresh-token schema.
   - Verifikasi request login:

     {
       "email": "user@example.test",
       "password": "secret"
     }

   - Verifikasi response login sukses:

     {
       "success": true,
       "data": {
         "token": "<jwt>",
         "expires_in": 28800,
         "role": "karyawan",
         "user": {
           "id": "<uuid>",
           "nama": "Nama Sintetis"
         }
       },
       "message": "Login berhasil"
     }

   - Jika OpenAPI berbeda dari API Contract v1.1, perbaiki OpenAPI dan validasi ulang
     sebelum menulis handler.

2. MIGRATION CORE AUTH

   Buat migration sesuai Database Schema v1.1 untuk tabel minimum yang dibutuhkan:

   - `departments`
   - `positions`
   - `roles`
   - `employees`
   - `users`
   - `permissions`
   - `audit_logs`

   Gunakan tabel existing jika sudah dibuat pada epic sebelumnya. Jangan menduplikasi migration.

   Wajib:
   - UUID primary key memakai `gen_random_uuid()`.
   - `roles.nama` unik dan dibatasi ke:
     - `karyawan`
     - `atasan`
     - `hr`
     - `top_management`
   - `employees.nip` unik.
   - `employees.atasan_id` self-reference nullable.
   - `employees.deleted_at` nullable untuk soft-delete.
   - `users.employee_id` unik dan NOT NULL.
   - `users.email` unik dan NOT NULL.
   - `users.password_hash` NOT NULL.
   - `users.failed_login_count` default 0.
   - `users.account_locked` default false.
   - `users.last_login_at` nullable.
   - `permissions` unik per `(role_id, modul, aksi)`.
   - `audit_logs` tidak memiliki updated_at/deleted_at.
   - Audit Log append-only.

   Jangan membuat tabel refresh token.
   Jangan membuat schema auto-migration.

3. SEED EMPAT ROLE DAN DATA TEST SINTETIS

   Seed wajib:
   - Empat role fixed sesuai schema.
   - Permission minimum yang diturunkan dari access matrix dan operation OpenAPI.
   - Data organisasi minimum sintetis untuk menghubungkan employee dan user.
   - Satu akun sintetis untuk masing-masing role pada environment development/test.

   Aturan:
   - Jangan memakai data personal nyata.
   - Password seed dibaca dari environment atau generated fixture test.
   - Jangan commit plaintext production password.
   - Seeder harus idempotent.
   - Pastikan hanya satu akun Top Management dibuat.
   - Jangan menebak permission bisnis di luar access matrix/OpenAPI.

4. DOMAIN MODEL DAN DTO

   Buat atau lengkapi model:
   - Role.
   - Employee identity minimum.
   - User.
   - Permission.
   - AuditLog.

   DTO:
   - LoginRequest.
   - LoginResponse.
   - LoginUserResponse.
   - AuthenticatedIdentity internal.

   Aturan:
   - Domain entity tidak bergantung pada HTTP.
   - DTO response tidak pernah memiliki `password_hash`.
   - DTO tidak mengembalikan failed login count atau internal session value.
   - JSON field memakai snake_case.
   - Gunakan UUID type/pola yang telah disetujui.

5. REPOSITORY CONTRACT

   Definisikan interface minimum dari sisi service:

   User repository:
   - FindForLogin(ctx, normalizedEmail).
   - FindIdentityByID(ctx, userID).
   - RecordSuccessfulLogin(ctx, userID, timestamp).
   - RecordFailedLogin(ctx, userID, threshold) -> updated state.
   - ResetFailedLogin(ctx, userID).
   - LockAccount(ctx, userID) bila tidak digabung dalam RecordFailedLogin.

   Permission repository:
   - HasPermission(ctx, roleID/roleName, module, action).
   - ListForRole(ctx, roleID/roleName) bila dibutuhkan authorization service.

   Audit repository:
   - Append(ctx, entry).

   Aturan concurrency:
   - Perubahan failed login count dan account lock harus atomic.
   - Gunakan transaction + row lock atau single conditional UPDATE yang setara.
   - Dua login gagal bersamaan tidak boleh kehilangan increment.
   - Login sukses tidak boleh membuka akun yang sudah locked.
   - Query harus mengecualikan employee yang soft-deleted/nonaktif sesuai kontrak.
   - Gunakan explicit column list, bukan `SELECT *`.

6. PASSWORD HASHING

   Buat abstraction:
   - Hash(password).
   - Compare(hash, password).

   Gunakan bcrypt atau Argon2 sesuai architecture decision yang telah disetujui.

   Aturan:
   - Jangan memilih algorithm/cost diam-diam jika belum diputuskan.
   - Jangan log password.
   - Jangan membedakan pesan user untuk email tidak ditemukan dan password salah.
   - Gunakan safe maximum input length untuk mencegah abuse.
   - Test hash dan compare tanpa snapshot hash yang nondeterministik.

7. JWT SERVICE

   Buat atau lengkapi JWT service:
   - Sign(userID, role, expiry).
   - ParseAndValidate(token).
   - Return identity yang typed.

   Wajib:
   - TTL 8 jam atau 28.800 detik.
   - Validasi signature, expiry, required claims, dan algorithm.
   - Tolak algorithm yang tidak diizinkan.
   - Gunakan current time/clock abstraction agar test deterministik.
   - Jangan memasukkan password, permission list, NIP, atau PII yang tidak diperlukan.
   - Jangan log token.

8. REDIS SESSION STORE

   Buat abstraction:
   - Create/ReplaceSession(ctx, userID, tokenFingerprint, ttl).
   - ValidateSession(ctx, userID, tokenFingerprint).
   - DeleteSession(ctx, userID).

   Aturan:
   - Key: `session:<user_id>`.
   - TTL tidak boleh lebih panjang dari JWT.
   - Jangan simpan plaintext password.
   - Simpan fingerprint/hash token atau value aman sesuai architecture decision.
   - Compare value secara aman.
   - Login baru mengganti session aktif untuk key user tersebut, sesuai single-key design
     pada System Design.
   - Redis failure pada login/auth check harus fail-closed.
   - Logout idempotent: session yang sudah tidak ada tetap menghasilkan response aman
     sesuai OpenAPI, selama token request awal valid.

9. LOGIN RATE LIMIT DAN ACCOUNT LOCKOUT

   Terapkan dua mekanisme yang berbeda:

   A. Failed login/account lock:
   - Normalisasi email.
   - User tidak ditemukan atau password salah -> `401 INVALID_CREDENTIALS`.
   - Untuk user yang ditemukan, increment `failed_login_count` secara atomic.
   - Kegagalan ke-1 sampai ke-4 -> `401 INVALID_CREDENTIALS`.
   - Kegagalan ke-5 -> set `account_locked=true`, hapus Redis session,
     return `429 ACCOUNT_LOCKED`.
   - Login berikutnya saat locked -> `429 ACCOUNT_LOCKED`.
   - Login sukses -> reset failed count, update last_login_at, create Redis session.
   - Jangan membuka locked account hanya karena password yang benar diberikan.

   B. Transport/rate-limit protection:
   - Gunakan Redis counter untuk melindungi endpoint login dari abuse.
   - Key tidak boleh membocorkan raw email; hash identifier bila digunakan.
   - Pertimbangkan kombinasi account identifier dan IP sesuai decision.
   - Jangan mengubah kebijakan lockout produk menjadi sekadar per-IP rate limit.

   Hindari account enumeration:
   - Response email tidak ditemukan dan password salah harus sama.
   - Jangan bocorkan status employee atau role sebelum autentikasi berhasil.
   - Pertimbangkan dummy password hash comparison untuk email tidak ditemukan.

10. AUTH SERVICE

    Definisikan:
    - Login(ctx, input, requestMetadata).
    - Logout(ctx, identity, tokenFingerprint, requestMetadata).

    Login flow:
    1. Normalize dan validate input.
    2. Apply rate-limit check.
    3. Find user for login.
    4. Return generic invalid credential untuk identifier/password salah.
    5. Reject locked account dengan `ACCOUNT_LOCKED`.
    6. Reject employee/user nonaktif sesuai mapping error contract.
    7. Compare password.
    8. Catat failure secara atomic bila salah.
    9. Sign JWT 8 jam.
    10. Store Redis session dengan TTL yang sama.
    11. Reset failed count dan update last_login_at.
    12. Append audit record.
    13. Return token, expires_in, role, ID, dan nama.

    Gunakan transaction untuk perubahan PostgreSQL yang harus konsisten.
    Perhatikan boundary PostgreSQL dan Redis:
    - Jangan return token sukses bila Redis session gagal dibuat.
    - Lakukan kompensasi/delete session jika langkah setelah Redis gagal dan hasilnya
      membuat login tidak boleh dianggap berhasil.
    - Dokumentasikan urutan dan failure behavior.

    Logout flow:
    1. Identity sudah divalidasi middleware.
    2. Delete Redis session.
    3. Append audit logout.
    4. Return response sukses sesuai OpenAPI.

11. AUTH MIDDLEWARE

    Implementasikan middleware fail-closed:
    1. Ambil `Authorization: Bearer <token>`.
    2. Tolak header hilang/malformed dengan `401 UNAUTHORIZED`.
    3. Parse dan validasi JWT.
    4. Bedakan invalid/expired hanya sesuai error code OpenAPI yang disetujui.
    5. Hitung token fingerprint.
    6. Cross-check `session:<user_id>` di Redis.
    7. Resolve user masih aktif/tidak soft-deleted bila policy memerlukannya.
    8. Inject typed identity ke `context.Context`.

    Jangan:
    - Percaya role dari request body/header custom.
    - Melanjutkan request saat Redis tidak tersedia.
    - Menggunakan string context key yang dapat collision.
    - Log Authorization header.

12. RBAC DAN RESOURCE SCOPE

    Implementasikan abstraction/middleware:
    - RequireRole(roles...).
    - RequirePermission(module, action).
    - RequireAnyPermission(...), hanya bila ada use case nyata.
    - IdentityFromContext.

    Siapkan authorization service untuk check dinamis dari database/cache.
    Jangan menyimpan permission list di JWT karena permission dapat diubah HR dan harus
    berlaku tanpa menunggu JWT 8 jam berakhir.

    Resource scope bukan middleware role sederhana:
    - `self`: data user/employee sendiri.
    - `direct_report`: bawahan langsung berdasarkan `employees.atasan_id`.
    - `hr_full`: akses HR sesuai operation.
    - `top_management_read_only`: semua read operation yang diizinkan.
    - `top_management_hr_approval`: hanya decision request milik HR.

    Epic ini membuat helper/service contract dan test dasarnya.
    Enforcement pada endpoint bisnis ditambahkan bersama masing-masing endpoint.

13. AUTH HANDLER DAN ROUTES

    POST `/api/v1/auth/login`:
    - Public.
    - Parse JSON dengan body size limit.
    - Tolak unknown/invalid body sesuai OpenAPI.
    - Validate field.
    - Panggil service.
    - Return token response tanpa cookie.

    POST `/api/v1/auth/logout`:
    - Wajib BearerAuth.
    - Lewati auth middleware lengkap termasuk Redis cross-check.
    - Panggil service logout.
    - Return:

      {
        "success": true,
        "data": null,
        "message": "Logout berhasil"
      }

    Jangan daftarkan:
    - `/auth/refresh`

    POST `/api/v1/auth/reset-password`:
    - Public dan rate-limited.
    - Verifikasi body dan password saat ini tanpa membocorkan keberadaan email.
    - Ganti hash secara atomik, buka lock, reset failed count, cabut seluruh session.
    - Return success aman tanpa password; pengguna login ulang.

14. AUDIT LOG

    Catat:
    - Login sukses.
    - Login gagal untuk account yang dapat diidentifikasi, tanpa password.
    - Account locked.
    - Logout.

    Gunakan action/modul/detail yang konsisten dengan schema.
    Jika taxonomy login failure belum ditetapkan:
    - Gunakan `aksi=LOGIN` dengan outcome aman di JSONB detail, atau
    - Catat decision yang dipilih.

    Jangan simpan:
    - Password.
    - JWT.
    - Redis session value.
    - Full Authorization header.
    - Detail yang membantu account enumeration.

    Audit append harus melalui repository insert-only.
    Tidak boleh ada update/delete Audit Log.

15. ERROR MAPPING

    Gunakan sentinel/typed error dan map ke OpenAPI:

    - Invalid JSON/field -> `400 VALIDATION_ERROR` atau code persis OpenAPI.
    - Credential salah -> `401 INVALID_CREDENTIALS`.
    - Bearer tidak ada/tidak valid/session invalid -> `401 UNAUTHORIZED`.
    - Akun terkunci -> `429 ACCOUNT_LOCKED`.
    - Rate limit umum -> `429 TOO_MANY_REQUESTS`.
    - Error internal -> `500 INTERNAL_ERROR`.

    Jika `USER_INACTIVE` dibutuhkan tetapi tidak ada pada API Contract v1.1:
    - Jangan menambah code diam-diam.
    - Gunakan code kontrak yang disetujui atau catat gap di OpenAPI decision log.

    Jangan expose database/Redis/hash/JWT errors ke client.

16. COMPOSITION ROOT

    Update `cmd/api/main.go`:
    - Wire user, permission, dan audit repositories.
    - Wire password hasher.
    - Wire JWT service.
    - Wire Redis session store dan login limiter.
    - Wire auth service.
    - Wire auth handler.
    - Wire auth dan RBAC middleware.
    - Register dua route auth.

    Hindari service locator/global mutable singleton.

17. UNIT TEST

    Minimal Auth Service:
    - Login valid -> token 8 jam, Redis session dibuat, counter reset, audit appended.
    - Email tidak ditemukan -> generic `INVALID_CREDENTIALS`.
    - Password salah ke-1 -> counter 1 dan `INVALID_CREDENTIALS`.
    - Password salah ke-4 -> belum locked.
    - Password salah ke-5 -> locked, existing session deleted, `ACCOUNT_LOCKED`.
    - Password benar pada account locked -> tetap `ACCOUNT_LOCKED`.
    - User/employee nonaktif -> ditolak tanpa session.
    - Redis CreateSession gagal -> login tidak return sukses.
    - Logout valid -> session deleted dan audit appended.
    - Logout idempotent sesuai contract.

    Minimal Middleware:
    - Header Bearer hilang.
    - Header malformed.
    - Signature invalid.
    - Token expired.
    - Redis session tidak ada.
    - Fingerprint tidak cocok.
    - Redis unavailable -> fail-closed.
    - Token valid + session valid -> identity typed di context.

    Minimal RBAC:
    - Role diizinkan.
    - Role ditolak.
    - Permission diizinkan.
    - Permission ditolak.
    - Top Management ditolak untuk mutation biasa.
    - Identity context hilang -> unauthorized/forbidden, tidak panic.

18. INTEGRATION TEST

    Gunakan PostgreSQL dan Redis test sesuai stack B-BE.1:
    - Migration up berhasil.
    - Seed empat role idempotent.
    - Login sukses menghasilkan response sesuai OpenAPI.
    - Empat kegagalan masih invalid credential.
    - Kegagalan kelima menghasilkan 429 dan account_locked di DB.
    - Token login dapat melewati protected test endpoint.
    - Logout membuat token lama gagal pada request berikutnya.
    - Restart/Redis session missing membuat token tidak aktif.
    - Audit rows tercatat dan tidak mengandung secret.
    - Dua login gagal concurrent tidak kehilangan increment.
    - Query Audit Log update/delete gagal untuk DB user aplikasi bila privilege
      enforcement sudah menjadi scope migration.

19. MANUAL SMOKE TEST

    Jangan hardcode credential di prompt, source code, atau command history.
    Gunakan environment lokal:

    - `GSNPEEPS_TEST_EMAIL`
    - `GSNPEEPS_TEST_PASSWORD`

    Skenario:
    - Login valid -> 200, token, expires_in 28800, role, user.
    - Logout dengan Bearer token -> 200.
    - Gunakan token lama setelah logout -> 401.
    - Login password salah lima kali pada akun test disposable -> 429 ACCOUNT_LOCKED.
    - Login password benar pada akun locked -> tetap 429.
    - Jangan membuka akun tersebut manual lewat SQL sebagai bagian smoke test.

Quality gates:

1. Format seluruh kode Go.
2. Jalankan `go mod tidy`.
3. Jalankan `go vet ./...`.
4. Jalankan `go test ./...`.
5. Jalankan integration test PostgreSQL + Redis.
6. Jalankan linter project.
7. Build API dan worker.
8. Jalankan OpenAPI lint.
9. Jalankan `docker compose config`.
10. Scan diff untuk secret, credential, token, fixture PII, dan debug endpoint.
11. Verifikasi response login/logout terhadap contoh OpenAPI.
12. Verifikasi tidak ada refresh-token table, cookie, endpoint, atau dependency flow.

Jika tool/dependency test belum tersedia:
- Jangan install global dependency tanpa izin.
- Gunakan project-local tooling.
- Laporkan gate yang tidak dijalankan.
- Jangan menyatakan gate lulus tanpa evidence.

Tahap Git hanya dilakukan jika pengguna meminta commit/push atau task aktif memberi
otorisasi eksplisit.

Branch:
- `feat/be-auth-rbac`

Commit yang disarankan:
- `feat(auth): add core auth migrations and role seeds`
- `feat(auth): add password hashing and jwt service`
- `feat(auth): add redis-backed session validation`
- `feat(auth): implement login lockout and logout`
- `feat(rbac): add role permission and scope guards`
- `test(auth): cover lockout session and middleware flows`

Jangan membuat commit kosong hanya untuk mengikuti daftar.
Jangan push branch atau membuka PR tanpa otorisasi eksplisit.

Jika diotorisasi, judul PR:
- `feat(auth): implement GSNpeeps authentication and RBAC`

Aturan akhir:
- Jangan menambah endpoint yang tidak ada di OpenAPI.
- Jangan membuat refresh-token flow atau refresh cookie.
- Jangan membuat reset oleh HR atau forgot-password email/OTP di luar self-reset OpenAPI.
- Jangan menyimpan permission list di JWT.
- Jangan membuat auth/RBAC middleware yang fail-open.
- Jangan menggunakan AutoMigrate.
- Jangan log password, JWT, session value, atau Authorization header.
- Jangan mengembalikan password_hash atau internal account state.
- Jangan memakai data personal nyata untuk seed/test.
- Identifier code menggunakan English.
- Pesan end-user menggunakan Bahasa Indonesia.
```

## Acceptance Criteria

- [ ] Lima operasi auth/password sesuai OpenAPI diimplementasikan.
- [ ] Login valid mengembalikan JWT 8 jam dan response persis kontrak.
- [ ] Redis `session:<user_id>` dibuat saat login dan dicek pada setiap request protected.
- [ ] Logout menghapus session sehingga token lama langsung tidak berlaku.
- [ ] Failed login counter aman terhadap concurrency.
- [ ] Kegagalan pertama sampai keempat menghasilkan `401 INVALID_CREDENTIALS`.
- [ ] Kegagalan kelima dan akun locked menghasilkan `429 ACCOUNT_LOCKED`.
- [ ] Password yang benar tidak membuka akun yang sudah locked.
- [ ] Password memakai secure hash yang disetujui dan tidak pernah dicatat ke log.
- [ ] Empat role dan permission minimum di-seed secara idempotent.
- [ ] Middleware auth dan RBAC bersifat fail-closed.
- [ ] Permission dinamis tidak disimpan di JWT.
- [ ] Resource scope self/direct-report/HR/Top Management memiliki contract dan test.
- [ ] Audit login, login failure/lockout, dan logout tercatat tanpa secret.
- [ ] Tidak ada refresh-token table, endpoint, atau cookie.
- [ ] Self-reset, verifikasi password saat ini, unlock, session revocation, rate limit, dan
  audit tanpa secret teruji.
- [ ] Unit, integration, concurrency, lint, vet, build, dan OpenAPI validation lulus.

## Test Manual

Contoh di bawah menggunakan environment variable. Jangan mengganti dengan credential
yang di-commit.

```bash
# Login
curl -sS -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"${GSNPEEPS_TEST_EMAIL}\",\"password\":\"${GSNPEEPS_TEST_PASSWORD}\"}"

# Simpan token secara lokal tanpa mencetaknya ke log bersama.
export GSNPEEPS_TEST_TOKEN="<token-dari-response>"

# Logout
curl -sS -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Authorization: Bearer ${GSNPEEPS_TEST_TOKEN}"

# Token lama wajib gagal setelah logout.
curl -sS -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Authorization: Bearer ${GSNPEEPS_TEST_TOKEN}"
```

## Files yang Akan Dibuat atau Disesuaikan

Lokasi detail mengikuti arsitektur yang disetujui pada Epic B-BE.1:

```text
backend/
├── cmd/api/main.go
├── internal/
│   ├── domain/
│   │   ├── employee.go
│   │   ├── user.go
│   │   ├── role.go
│   │   ├── permission.go
│   │   └── audit_log.go
│   ├── repository/
│   │   ├── user_repository.go
│   │   ├── permission_repository.go
│   │   └── audit_repository.go
│   ├── service/
│   │   ├── auth_service.go
│   │   ├── authorization_service.go
│   │   └── errors.go
│   ├── handler/
│   │   └── auth_handler.go
│   ├── dto/
│   │   ├── auth_request.go
│   │   └── auth_response.go
│   ├── middleware/
│   │   ├── auth.go
│   │   ├── identity.go
│   │   └── rbac.go
│   └── platform/
│       ├── password/
│       ├── jwt/
│       └── redis/
│           ├── session_store.go
│           └── login_limiter.go
├── migrations/
│   └── *_create_core_auth_tables.sql
├── seeds/
│   └── roles_and_auth_fixtures.go
└── tests/
    └── auth_integration_test.go

docs/
└── openapi-decisions.md
```

Nama file dapat mengikuti convention repository yang sudah disetujui. Jangan membuat
package paralel yang menduplikasi abstraction dari Epic B-BE.1.
