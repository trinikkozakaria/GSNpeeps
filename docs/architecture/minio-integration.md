currently object storage are served using Nextcloud. I want you to abstract the upload and download of object storage, and add minio as the default object storage. make me the action list of what needs to change in the backend and the whole system in ../../docker-compose.yml. append the list here

---

## Action List — MinIO Object Storage Integration

**Keputusan** (2026-08-20): pertahankan Nextcloud sebagai adapter kompatibilitas mundur,
tambahkan interface `filestore` dengan adapter **MinIO** sebagai default. Driver aktif dipilih
lewat `STORAGE_DRIVER`. Baseline `CLAUDE.md` §3/§12 sudah diperbarui untuk mencatat keputusan
ini.

### Kondisi kode saat ini (baseline sebelum perubahan)

- `backend/internal/platform/webdav/client.go` — implementasi konkret WebDAV (`Upload`,
  `Delete`, `Download`, `Health`), sekaligus mendeklarasikan interface `Storage` yang tidak
  dipakai konsumen mana pun.
- Konsumer nyata (`internal/service/employee_service.go`, `internal/handler/uat_handler.go`,
  worker) sudah mendefinisikan interface sempit sendiri (`Upload(...)`, `Download(...)`) yang
  dipenuhi `*webdav.Client` secara struktural — bagian abstraksi ini sudah idiomatis dan
  **tidak perlu diubah**.
- `cmd/api/main.go` dan `cmd/worker/main.go` memanggil `webdav.New(cfg.Nextcloud)` secara
  langsung dan meneruskan `*webdav.Client` konkret — inilah titik yang harus diganti jadi
  pemilihan driver.
- `internal/config/config.go` hanya punya struct `Nextcloud` dan mewajibkan
  `NEXTCLOUD_WEBDAV_URL`/`NEXTCLOUD_USERNAME`/`NEXTCLOUD_APP_PASSWORD` di `validate()`.
- `docker-compose.yml` menjalankan `nextcloud` sebagai service wajib (`backend-api` dan
  `migrate` depend on it via healthcheck) — tidak ada MinIO.
- `docs/architecture` (`.claude/skills/hris-backend/references/architecture.md`) sudah
  menyebut port `FileStore` dan direktori `platform/nextcloud/`, tapi implementasi nyata ada di
  `platform/webdav/` — dokumen perlu diselaraskan ke struktur final di bawah.

### Fase 1 — Port dan adapter backend

1. Buat `backend/internal/platform/filestore/filestore.go` berisi port bersama:
   ```go
   type Store interface {
       Upload(ctx context.Context, objectPath string, body io.Reader, contentType string) (string, error)
       Download(ctx context.Context, storedPath string) (io.ReadCloser, string, error)
       Delete(ctx context.Context, objectPath string) error
       Health(ctx context.Context) error
   }
   ```
   Signature harus identik dengan interface sempit yang sudah dipakai `employee_service.go`
   dan `uat_handler.go` supaya kedua adapter bisa saling ditukar tanpa menyentuh service/handler.
2. Pindahkan helper `safePath` (path traversal guard) dari `platform/webdav/client.go` ke
   `platform/filestore/path.go` agar dipakai bersama oleh adapter WebDAV maupun MinIO — hindari
   duplikasi logic keamanan yang sama persis di dua tempat.
3. Tambah `backend/internal/platform/minio/client.go`:
   - Gunakan `github.com/minio/minio-go/v7` (`go get` lalu `go mod tidy`; ini dependency baru
     yang inheren pada keputusan MinIO, bukan pelanggaran baseline).
   - `New(cfg config.MinIO) (*Client, error)` — inisialisasi `minio.New(...)`, lalu pastikan
     bucket ada (`BucketExists` + `MakeBucket` idempotent) sekali saat startup, bukan per
     request (beda dengan `ensureCollections` WebDAV yang jalan per-object karena WebDAV butuh
     parent collection; object storage S3-compatible tidak butuh itu — key dengan `/` cukup).
   - `Upload` → `PutObject`, kembalikan locator relatif konsisten dengan pola
     `employees/<employee_id>/<document_type>/<generated_id>-<safe_name>` yang sudah dipakai.
   - `Download` → `GetObject`, kembalikan `io.ReadCloser` + content-type (backend tetap jadi
     proxy; **jangan** beralih ke presigned URL langsung ke browser, supaya authorization
     check per-request di handler tidak bisa dilewati — konsisten dengan §8 File Security).
   - `Delete` → `RemoveObject`.
   - `Health` → `BucketExists` atau `ListBuckets` ringan dengan timeout.
   - Pakai `safePath` yang sama dari `platform/filestore` untuk menolak traversal sebelum jadi
     object key.
4. Pastikan `*webdav.Client` dan `*minio.Client` sama-sama memenuhi `filestore.Store` (compile-
   time assertion `var _ filestore.Store = (*Client)(nil)` di masing-masing adapter).
5. Hapus interface `Storage` yang tidak terpakai di `platform/webdav/client.go` (duplikat dari
   `filestore.Store`) — jangan disimpan sebagai dead code.

### Fase 2 — Konfigurasi

6. `internal/config/config.go`:
   - Tambah `Storage.Driver` (`STORAGE_DRIVER`, default `"minio"`, hanya menerima
     `minio`/`nextcloud`, tolak nilai lain di `validate()`).
   - Tambah struct `MinIO{Endpoint, AccessKey, SecretKey, Bucket, UseSSL, Region, Timeout}`
     dari `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET`,
     `MINIO_USE_SSL`, `MINIO_HTTP_TIMEOUT`.
   - Ubah `validate()` supaya hanya mewajibkan env var driver yang aktif (MinIO_* saat
     `minio`, `NEXTCLOUD_*` saat `nextcloud`) — jangan mewajibkan keduanya sekaligus.
   - Pertahankan struct `Nextcloud` apa adanya untuk adapter kompatibilitas mundur.
7. `backend/.env.example`: tambah `STORAGE_DRIVER=minio` dan blok `MINIO_*`; beri komentar
   bahwa `NEXTCLOUD_*` hanya dipakai saat `STORAGE_DRIVER=nextcloud`. (Sudah dicerminkan di
   `CLAUDE.md` §12; sinkronkan nilai placeholder-nya.)

### Fase 3 — Wiring `cmd/`

8. Tambah factory kecil, misalnya `filestore.New(cfg config.Config) (filestore.Store, error)`
   yang switch pada `cfg.Storage.Driver` dan memanggil `minio.New` atau `webdav.New`.
9. `cmd/api/main.go` dan `cmd/worker/main.go`: ganti pemanggilan langsung `webdav.New(cfg.Nextcloud)`
   dengan `filestore.New(cfg)`; tipe variabel (`documentStore`, `storage`) cukup dideklarasikan
   sebagai `filestore.Store` — tidak ada perubahan lain di kedua file karena parameter yang
   diteruskan ke service/handler/worker sudah berupa interface sempit yang identik.

### Fase 4 — `docker-compose.yml`

10. Tambah service `minio`:
    - Image `minio/minio:RELEASE.<pin-versi-terbaru>` (pin versi eksplisit, jangan `latest`).
    - `command: server /data --console-address ":9001"`.
    - Env: `MINIO_ROOT_USER`, `MINIO_ROOT_PASSWORD` dari `.env` (map ke `MINIO_ACCESS_KEY`/
      `MINIO_SECRET_KEY` yang dipakai backend, atau samakan nama variabel).
    - Volume `minio-data:/data` (tambahkan ke blok `volumes:` root).
    - `expose: ["9000", "9001"]` — **jangan** publish port ke host/Nginx; console admin hanya
      diakses lewat `docker compose exec`/port-forward manual, bukan lewat gateway publik,
      konsisten dengan aturan "hanya Nginx boleh expose port publik".
    - `healthcheck`: `curl -f http://localhost:9000/minio/health/live` (image MinIO sudah
      punya `curl` built-in) dengan interval/retries serupa pola `postgres`/`redis`.
    - `networks: [internal]`.
11. `backend-api` dan `cron-worker`: tambah `STORAGE_DRIVER`, `MINIO_ENDPOINT: "minio:9000"`,
    `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET`, `MINIO_USE_SSL: "false"` ke blok
    `environment: &backend-environment`; ganti `depends_on.nextcloud` → `depends_on.minio`
    (`condition: service_healthy`).
12. Jadikan service `nextcloud` opsional lewat Compose `profiles: ["nextcloud"]`, supaya
    `docker compose up` default tidak lagi menjalankan Nextcloud kecuali eksplisit
    `--profile nextcloud` atau `STORAGE_DRIVER=nextcloud` dipilih untuk suatu environment.
    Pertahankan definisi service dan `nextcloud-data` volume apa adanya untuk kompatibilitas
    mundur.
13. `.env.example` (root): tambah `MINIO_ROOT_USER`, `MINIO_ROOT_PASSWORD`,
    `STORAGE_DRIVER=minio`; catat bahwa `NEXTCLOUD_ADMIN_USER`/`NEXTCLOUD_ADMIN_PASSWORD`
    hanya perlu diisi bila menjalankan profile `nextcloud`.
14. Jalankan `docker compose config` setelah perubahan untuk memastikan file tetap valid
    (Definition of Done §15 mewajibkan ini untuk area yang berubah).

### Fase 5 — Migrasi data (kondisional)

15. Cek apakah ada environment (dev/staging) yang sudah memiliki file nyata di Nextcloud. Bila
    ada, tulis skrip migrasi sekali-jalan (mis. `backend/cmd/migrate-storage/main.go`) yang
    membaca seluruh object di bawah `NEXTCLOUD_ROOT_FOLDER` lewat adapter WebDAV dan menulis
    ulang ke bucket MinIO dengan path yang sama persis, lalu verifikasi checksum sebelum
    memutus akses Nextcloud. **Jangan hapus data Nextcloud lama sampai verifikasi migrasi
    selesai.**
    Bila belum ada data nyata (kemungkinan besar untuk project ini, karena masih pre-launch
    menuju target 28 Agustus 2026), lewati fase ini dan cukup catat sebagai known limitation.

### Fase 6 — Test

16. Tambah `backend/internal/platform/minio/client_test.go` setara `webdav/client_test.go`:
    path-traversal rejection, upload/download/delete round-trip, health check, dan penolakan
    saat bucket tidak bisa dibuat — gunakan MinIO real via container test (mis.
    `testcontainers-go` module MinIO) supaya konsisten dengan strategi "repository/integration
    test memakai service nyata" di CLAUDE.md §14, bukan mock jaringan.
17. Tambah contract test table-driven di `platform/filestore` yang menjalankan skenario yang
    sama (upload lalu download mengembalikan isi identik, delete membuat objek tak terbaca,
    traversal ditolak) terhadap kedua adapter, supaya keduanya dijamin punya perilaku setara
    di balik interface `Store`.
18. Perbarui test yang sudah meng-hardcode ekspektasi Nextcloud-only bila ada (mis. skenario di
    `internal/worker/worker_test.go`, `internal/service/employee_service_test.go`) untuk jalan
    terhadap `filestore.Store` (bisa pakai fake in-memory yang mengimplementasikan interface
    tsb untuk unit test service/handler yang tidak butuh I/O nyata).

### Fase 7 — Dokumentasi

19. `.claude/skills/hris-backend/references/architecture.md`: perbarui diagram system context,
    daftar direktori `platform/` (ganti `nextcloud/` jadi `webdav/` + `minio/`, tambah
    `filestore/`), dan bagian "Nextcloud and file architecture" jadi "Object storage and file
    architecture" yang menjelaskan port `filestore.Store` + kedua adapter.
20. `.claude/skills/hris-backend/references/nextcloud-worker.md`: tambah bagian MinIO (retensi
    foto 3 bulan tetap lewat `Delete` di interface yang sama, tidak ada perubahan logic worker)
    atau ganti judul jadi mencakup kedua backend.
21. `docs/operations/runbook.md`: tambah prosedur operasional MinIO (backup bucket, rotasi
    `MINIO_SECRET_KEY`, cara mengakses console admin lewat port-forward internal).
22. Tandai keputusan ini di `.claude/specs/document-index.md` bila dokumen tsb melacak
    keputusan arsitektur bertanggal.

### Fase 8 — Keamanan (checklist sebelum merge)

- MinIO root/access credential tidak pernah dikirim ke frontend (sama seperti aturan Nextcloud
  app password di CLAUDE.md §8).
- Bucket bersifat private (bukan `public-read`); tidak ada anonymous policy yang membuat
  object bisa diakses tanpa lewat backend.
- Backend tetap jadi proxy authorization untuk `Download` — tidak ada presigned URL yang
  dikirim langsung ke browser tanpa pengecekan role/ownership di handler.
- Validasi ukuran (maks 5 MB), ekstensi, MIME, dan file signature tetap dilakukan di backend
  sebelum `Upload`, tidak berubah oleh penggantian backend storage.
- `MINIO_SECRET_KEY` tidak pernah masuk log (sejalan dengan larangan mencatat credential di
  CLAUDE.md §5).

### Urutan pengerjaan yang disarankan

Fase 1 → 2 → 3 (backend selesai dan lulus test lokal dengan `STORAGE_DRIVER=minio`) → Fase 4
(compose) → Fase 6 (test lengkap termasuk contract test dua adapter) → Fase 5 (migrasi data,
hanya bila relevan) → Fase 7 (dokumentasi) → Fase 8 (checklist keamanan sebagai gate sebelum
merge). CLAUDE.md §3/§12 sudah diperbarui di awal task ini sebagai keputusan baseline resmi.
currently object storage are served using Nextcloud. I want you to abstract the upload and download of object storage, and add minio as the default object storage. make me the action list of what needs to change in the backend and the whole system in ../../docker-compose.yml. append the list here

---

## Action List — MinIO Object Storage Integration

**Keputusan** (2026-08-20): pertahankan Nextcloud sebagai adapter kompatibilitas mundur,
tambahkan interface `filestore` dengan adapter **MinIO** sebagai default. Driver aktif dipilih
lewat `STORAGE_DRIVER`. Baseline `CLAUDE.md` §3/§12 sudah diperbarui untuk mencatat keputusan
ini.

### Kondisi kode saat ini (baseline sebelum perubahan)

- `backend/internal/platform/webdav/client.go` — implementasi konkret WebDAV (`Upload`,
  `Delete`, `Download`, `Health`), sekaligus mendeklarasikan interface `Storage` yang tidak
  dipakai konsumen mana pun.
- Konsumer nyata (`internal/service/employee_service.go`, `internal/handler/uat_handler.go`,
  worker) sudah mendefinisikan interface sempit sendiri (`Upload(...)`, `Download(...)`) yang
  dipenuhi `*webdav.Client` secara struktural — bagian abstraksi ini sudah idiomatis dan
  **tidak perlu diubah**.
- `cmd/api/main.go` dan `cmd/worker/main.go` memanggil `webdav.New(cfg.Nextcloud)` secara
  langsung dan meneruskan `*webdav.Client` konkret — inilah titik yang harus diganti jadi
  pemilihan driver.
- `internal/config/config.go` hanya punya struct `Nextcloud` dan mewajibkan
  `NEXTCLOUD_WEBDAV_URL`/`NEXTCLOUD_USERNAME`/`NEXTCLOUD_APP_PASSWORD` di `validate()`.
- `docker-compose.yml` menjalankan `nextcloud` sebagai service wajib (`backend-api` dan
  `migrate` depend on it via healthcheck) — tidak ada MinIO.
- `docs/architecture` (`.claude/skills/hris-backend/references/architecture.md`) sudah
  menyebut port `FileStore` dan direktori `platform/nextcloud/`, tapi implementasi nyata ada di
  `platform/webdav/` — dokumen perlu diselaraskan ke struktur final di bawah.

### Fase 1 — Port dan adapter backend

1. Buat `backend/internal/platform/filestore/filestore.go` berisi port bersama:
   ```go
   type Store interface {
       Upload(ctx context.Context, objectPath string, body io.Reader, contentType string) (string, error)
       Download(ctx context.Context, storedPath string) (io.ReadCloser, string, error)
       Delete(ctx context.Context, objectPath string) error
       Health(ctx context.Context) error
   }
   ```
   Signature harus identik dengan interface sempit yang sudah dipakai `employee_service.go`
   dan `uat_handler.go` supaya kedua adapter bisa saling ditukar tanpa menyentuh service/handler.
2. Pindahkan helper `safePath` (path traversal guard) dari `platform/webdav/client.go` ke
   `platform/filestore/path.go` agar dipakai bersama oleh adapter WebDAV maupun MinIO — hindari
   duplikasi logic keamanan yang sama persis di dua tempat.
3. Tambah `backend/internal/platform/minio/client.go`:
   - Gunakan `github.com/minio/minio-go/v7` (`go get` lalu `go mod tidy`; ini dependency baru
     yang inheren pada keputusan MinIO, bukan pelanggaran baseline).
   - `New(cfg config.MinIO) (*Client, error)` — inisialisasi `minio.New(...)`, lalu pastikan
     bucket ada (`BucketExists` + `MakeBucket` idempotent) sekali saat startup, bukan per
     request (beda dengan `ensureCollections` WebDAV yang jalan per-object karena WebDAV butuh
     parent collection; object storage S3-compatible tidak butuh itu — key dengan `/` cukup).
   - `Upload` → `PutObject`, kembalikan locator relatif konsisten dengan pola
     `employees/<employee_id>/<document_type>/<generated_id>-<safe_name>` yang sudah dipakai.
   - `Download` → `GetObject`, kembalikan `io.ReadCloser` + content-type (backend tetap jadi
     proxy; **jangan** beralih ke presigned URL langsung ke browser, supaya authorization
     check per-request di handler tidak bisa dilewati — konsisten dengan §8 File Security).
   - `Delete` → `RemoveObject`.
   - `Health` → `BucketExists` atau `ListBuckets` ringan dengan timeout.
   - Pakai `safePath` yang sama dari `platform/filestore` untuk menolak traversal sebelum jadi
     object key.
4. Pastikan `*webdav.Client` dan `*minio.Client` sama-sama memenuhi `filestore.Store` (compile-
   time assertion `var _ filestore.Store = (*Client)(nil)` di masing-masing adapter).
5. Hapus interface `Storage` yang tidak terpakai di `platform/webdav/client.go` (duplikat dari
   `filestore.Store`) — jangan disimpan sebagai dead code.

### Fase 2 — Konfigurasi

6. `internal/config/config.go`:
   - Tambah `Storage.Driver` (`STORAGE_DRIVER`, default `"minio"`, hanya menerima
     `minio`/`nextcloud`, tolak nilai lain di `validate()`).
   - Tambah struct `MinIO{Endpoint, AccessKey, SecretKey, Bucket, UseSSL, Region, Timeout}`
     dari `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET`,
     `MINIO_USE_SSL`, `MINIO_HTTP_TIMEOUT`.
   - Ubah `validate()` supaya hanya mewajibkan env var driver yang aktif (MinIO_* saat
     `minio`, `NEXTCLOUD_*` saat `nextcloud`) — jangan mewajibkan keduanya sekaligus.
   - Pertahankan struct `Nextcloud` apa adanya untuk adapter kompatibilitas mundur.
7. `backend/.env.example`: tambah `STORAGE_DRIVER=minio` dan blok `MINIO_*`; beri komentar
   bahwa `NEXTCLOUD_*` hanya dipakai saat `STORAGE_DRIVER=nextcloud`. (Sudah dicerminkan di
   `CLAUDE.md` §12; sinkronkan nilai placeholder-nya.)

### Fase 3 — Wiring `cmd/`

8. Tambah factory kecil, misalnya `filestore.New(cfg config.Config) (filestore.Store, error)`
   yang switch pada `cfg.Storage.Driver` dan memanggil `minio.New` atau `webdav.New`.
9. `cmd/api/main.go` dan `cmd/worker/main.go`: ganti pemanggilan langsung `webdav.New(cfg.Nextcloud)`
   dengan `filestore.New(cfg)`; tipe variabel (`documentStore`, `storage`) cukup dideklarasikan
   sebagai `filestore.Store` — tidak ada perubahan lain di kedua file karena parameter yang
   diteruskan ke service/handler/worker sudah berupa interface sempit yang identik.

### Fase 4 — `docker-compose.yml`

10. Tambah service `minio`:
    - Image `minio/minio:RELEASE.<pin-versi-terbaru>` (pin versi eksplisit, jangan `latest`).
    - `command: server /data --console-address ":9001"`.
    - Env: `MINIO_ROOT_USER`, `MINIO_ROOT_PASSWORD` dari `.env` (map ke `MINIO_ACCESS_KEY`/
      `MINIO_SECRET_KEY` yang dipakai backend, atau samakan nama variabel).
    - Volume `minio-data:/data` (tambahkan ke blok `volumes:` root).
    - `expose: ["9000", "9001"]` — **jangan** publish port ke host/Nginx; console admin hanya
      diakses lewat `docker compose exec`/port-forward manual, bukan lewat gateway publik,
      konsisten dengan aturan "hanya Nginx boleh expose port publik".
    - `healthcheck`: `curl -f http://localhost:9000/minio/health/live` (image MinIO sudah
      punya `curl` built-in) dengan interval/retries serupa pola `postgres`/`redis`.
    - `networks: [internal]`.
11. `backend-api` dan `cron-worker`: tambah `STORAGE_DRIVER`, `MINIO_ENDPOINT: "minio:9000"`,
    `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET`, `MINIO_USE_SSL: "false"` ke blok
    `environment: &backend-environment`; ganti `depends_on.nextcloud` → `depends_on.minio`
    (`condition: service_healthy`).
12. Jadikan service `nextcloud` opsional lewat Compose `profiles: ["nextcloud"]`, supaya
    `docker compose up` default tidak lagi menjalankan Nextcloud kecuali eksplisit
    `--profile nextcloud` atau `STORAGE_DRIVER=nextcloud` dipilih untuk suatu environment.
    Pertahankan definisi service dan `nextcloud-data` volume apa adanya untuk kompatibilitas
    mundur.
13. `.env.example` (root): tambah `MINIO_ROOT_USER`, `MINIO_ROOT_PASSWORD`,
    `STORAGE_DRIVER=minio`; catat bahwa `NEXTCLOUD_ADMIN_USER`/`NEXTCLOUD_ADMIN_PASSWORD`
    hanya perlu diisi bila menjalankan profile `nextcloud`.
14. Jalankan `docker compose config` setelah perubahan untuk memastikan file tetap valid
    (Definition of Done §15 mewajibkan ini untuk area yang berubah).

### Fase 5 — Migrasi data (kondisional)

15. Cek apakah ada environment (dev/staging) yang sudah memiliki file nyata di Nextcloud. Bila
    ada, tulis skrip migrasi sekali-jalan (mis. `backend/cmd/migrate-storage/main.go`) yang
    membaca seluruh object di bawah `NEXTCLOUD_ROOT_FOLDER` lewat adapter WebDAV dan menulis
    ulang ke bucket MinIO dengan path yang sama persis, lalu verifikasi checksum sebelum
    memutus akses Nextcloud. **Jangan hapus data Nextcloud lama sampai verifikasi migrasi
    selesai.**
    Bila belum ada data nyata (kemungkinan besar untuk project ini, karena masih pre-launch
    menuju target 28 Agustus 2026), lewati fase ini dan cukup catat sebagai known limitation.

### Fase 6 — Test

16. Tambah `backend/internal/platform/minio/client_test.go` setara `webdav/client_test.go`:
    path-traversal rejection, upload/download/delete round-trip, health check, dan penolakan
    saat bucket tidak bisa dibuat — gunakan MinIO real via container test (mis.
    `testcontainers-go` module MinIO) supaya konsisten dengan strategi "repository/integration
    test memakai service nyata" di CLAUDE.md §14, bukan mock jaringan.
17. Tambah contract test table-driven di `platform/filestore` yang menjalankan skenario yang
    sama (upload lalu download mengembalikan isi identik, delete membuat objek tak terbaca,
    traversal ditolak) terhadap kedua adapter, supaya keduanya dijamin punya perilaku setara
    di balik interface `Store`.
18. Perbarui test yang sudah meng-hardcode ekspektasi Nextcloud-only bila ada (mis. skenario di
    `internal/worker/worker_test.go`, `internal/service/employee_service_test.go`) untuk jalan
    terhadap `filestore.Store` (bisa pakai fake in-memory yang mengimplementasikan interface
    tsb untuk unit test service/handler yang tidak butuh I/O nyata).

### Fase 7 — Dokumentasi

19. `.claude/skills/hris-backend/references/architecture.md`: perbarui diagram system context,
    daftar direktori `platform/` (ganti `nextcloud/` jadi `webdav/` + `minio/`, tambah
    `filestore/`), dan bagian "Nextcloud and file architecture" jadi "Object storage and file
    architecture" yang menjelaskan port `filestore.Store` + kedua adapter.
20. `.claude/skills/hris-backend/references/nextcloud-worker.md`: tambah bagian MinIO (retensi
    foto 3 bulan tetap lewat `Delete` di interface yang sama, tidak ada perubahan logic worker)
    atau ganti judul jadi mencakup kedua backend.
21. `docs/operations/runbook.md`: tambah prosedur operasional MinIO (backup bucket, rotasi
    `MINIO_SECRET_KEY`, cara mengakses console admin lewat port-forward internal).
22. Tandai keputusan ini di `.claude/specs/document-index.md` bila dokumen tsb melacak
    keputusan arsitektur bertanggal.

### Fase 8 — Keamanan (checklist sebelum merge)

- MinIO root/access credential tidak pernah dikirim ke frontend (sama seperti aturan Nextcloud
  app password di CLAUDE.md §8).
- Bucket bersifat private (bukan `public-read`); tidak ada anonymous policy yang membuat
  object bisa diakses tanpa lewat backend.
- Backend tetap jadi proxy authorization untuk `Download` — tidak ada presigned URL yang
  dikirim langsung ke browser tanpa pengecekan role/ownership di handler.
- Validasi ukuran (maks 5 MB), ekstensi, MIME, dan file signature tetap dilakukan di backend
  sebelum `Upload`, tidak berubah oleh penggantian backend storage.
- `MINIO_SECRET_KEY` tidak pernah masuk log (sejalan dengan larangan mencatat credential di
  CLAUDE.md §5).

### Urutan pengerjaan yang disarankan

Fase 1 → 2 → 3 (backend selesai dan lulus test lokal dengan `STORAGE_DRIVER=minio`) → Fase 4
(compose) → Fase 6 (test lengkap termasuk contract test dua adapter) → Fase 5 (migrasi data,
hanya bila relevan) → Fase 7 (dokumentasi) → Fase 8 (checklist keamanan sebagai gate sebelum
merge). CLAUDE.md §3/§12 sudah diperbarui di awal task ini sebagai keputusan baseline resmi.
