# GSNpeeps OpenAPI decision log

Dokumen ini mencatat keputusan dan gap ketika sumber GSNpeeps diterjemahkan menjadi
OpenAPI 3.1. PDF sumber tetap dipertahankan sebagai rekam acuan awal.

## Source precedence

Urutan sumber awal:

1. API Contract v1.1 untuk path, method, payload, response, dan HTTP status.
2. Database Schema v1.1 untuk tipe data, enum, relasi, dan constraint.
3. PRD v1.2 dan User Story v1.2 untuk role, row-level authorization, dan acceptance criteria.
4. Sequence Diagram v1.1 untuk urutan side effect dan transaksi.
5. System Design v1.0 untuk deployment dan boundary integrasi.
6. Keputusan produk tertulis tanggal 28 Juli 2026 untuk revisi setelah PDF diterbitkan.

Keputusan produk terbaru mengungguli bagian PDF yang secara eksplisit direvisi di bawah.

## Decisions applied through OpenAPI 0.4.0

### D-001 — 46 approved operations

API Contract v1.1 memuat 42 operasi. Keputusan produk tanggal 28 Juli 2026 menambahkan:

- `GET /api/v1/auth/me`;
- `PATCH /api/v1/auth/me/password`;
- `POST /api/v1/auth/reset-password`;
- `GET /api/v1/master/lokasi-kantor`.

OpenAPI 0.4.0 karena itu memiliki tepat 46 operasi. Tidak ada refresh-token endpoint.

### D-002 — Check-in and checkout share one operation

Check-in dan checkout menggunakan `POST /api/v1/absensi/checkin`. Aksi dibedakan melalui
field multipart `tipe` dengan enum `check_in` atau `check_out`. Tidak dibuat route
`/checkout`.

### D-003 — Employee PUT is a partial update

`PUT /api/v1/karyawan/{id}` mengikuti API Contract: seluruh field
`UpdateEmployeeRequest` opsional.

### D-004 — Top Management access uses the latest contract

Top Management mendapat akses read-only ke daftar/detail/dokumen karyawan, dashboard,
live feed, laporan, role, permission, dan audit log. Pernyataan lama US-23 dianggap
digantikan oleh sumber versi lebih baru.

### D-005 — Status values at the HTTP boundary

Database, query, dan response API memakai enum machine-readable lowercase/underscore yang
sama: `menunggu_atasan`, `menunggu_hr`, `menunggu_top_management`, `disetujui`, `ditolak`,
dan `dibatalkan`. Frontend bertanggung jawab memetakan enum tersebut ke label Bahasa
Indonesia. API tidak lagi mengirim label tampilan sebagai nilai status.

### D-006 — File URL remains an opaque API value

Schema mempertahankan `file_url`, `foto_url`, `dokumen_url`, dan `slip_url`. Nilainya harus
berupa URL/path akses-terkontrol tanpa credential Nextcloud. Mekanisme delivery final masih
menjadi gap G-007.

### D-007 — Health endpoint is outside `/api/v1`

`/health` tetap berada di root, sedangkan operasi bisnis memakai prefix `/api/v1`.

### D-008 — Synthetic identifiers are UUID

Database Schema menetapkan ID sebagai UUID. OpenAPI menggunakan `format: uuid` dan contoh
sintetis.

### D-009 — Login is email-only

Login hanya menerima `email` dan `password`. Istilah username pada narasi sumber tidak
berlaku untuk implementasi GSNpeeps.

### D-010 — Current-user session restoration

`GET /api/v1/auth/me` menjadi sumber identitas dan role setelah browser reload. Endpoint
memerlukan JWT serta session Redis aktif, dan tidak memperpanjang umur token delapan jam.

### D-011 — Employee self-service password reset

Karyawan mereset password sendiri melalui `POST /api/v1/auth/reset-password`, termasuk ketika
akun terkunci. Karena layanan email/OTP belum disetujui, request wajib memverifikasi `email`
dan password saat ini sebelum menerima password baru. Sistem:

- membuka lockout dan mereset counter kegagalan;
- mencabut seluruh session aktif;
- memperbarui password hash dan status akun yang ditampilkan ke HR;
- mencatat audit tanpa nilai password.

Kegagalan memakai error generik, rate limit gabungan akun+IP, dan counter yang sama dengan
login agar endpoint tidak menjadi bypass brute-force. HR tidak melihat atau mengganti
password. Password yang terlupa tanpa mengetahui password saat ini tetap di luar scope sampai
kanal verifikasi recovery disetujui.

### D-012 — Coordinates required for every work mode

`gps_lat` dan `gps_long` wajib untuk WFO, WFH, dan WFA. Backend menghitung radius kantor
hanya untuk WFO terhadap `office_location_id` aktif yang dipilih karyawan dan menolak jarak
lebih dari 100 meter dengan `422 OUT_OF_RADIUS`. WFH dan WFA menyimpan koordinat tetapi tidak
menjalankan validasi radius kantor.

### D-013 — The approved schema contains 26 tables

Daftar Database Schema v1.1 menyebut 25 nama tabel meskipun ringkasannya menulis 26.
Kebutuhan multi-kantor menetapkan tabel ke-26 bernama `office_locations`. Tabel menyimpan
UUID, kode unik, nama, alamat nullable, latitude, longitude, status aktif, dan timestamp.
Tidak ada foreign key kantor permanen pada employee karena karyawan bebas WFO di kantor aktif
mana pun. Jangan seed lokasi fiktif; data resmi menyusul.

### D-014 — Contract H-30 recipient fallback

H-30 berarti tepat 30 hari kalender sebelum tanggal berakhir kontrak berdasarkan timezone
Asia/Jakarta. Worker harian menentukan penerima dengan urutan berikut:

1. tambahkan atasan langsung yang aktif jika tersedia;
2. tambahkan semua user HR aktif selain karyawan yang kontraknya akan habis;
3. jika tidak ada HR aktif yang memenuhi syarat, tambahkan satu-satunya user Top Management
   aktif;
4. deduplikasi berdasarkan `recipient_user_id` sebelum menulis notifikasi.

Setiap pasangan kontrak/siklus/penerima memakai event key deterministik. Karena invariant
produk menetapkan tepat satu Top Management, fallback tidak memerlukan algoritme pemilihan.
Jika tidak ada Top Management aktif ketika fallback diperlukan, item job gagal secara
terukur dan tidak boleh dianggap sukses atau dikirim kepada subjek sendiri.

### D-015 — Dashboard period, attendance, inactive, and gender rules

Dashboard menerima `periode=harian|mingguan|bulanan|tahunan` dan `tanggal_acuan`. Batas
periode dihitung dalam timezone Asia/Jakarta: harian satu tanggal, mingguan Senin-Minggu,
bulanan bulan kalender, dan tahunan 1 Januari-31 Desember.

- Kehadiran dihitung hanya dari check-in valid dalam rentang periode.
- `hadir_valid` adalah pasangan unik employee/tanggal dengan check-in valid.
- Hari kerja Senin-Jumat, jam kerja 09:00-18:00 WIB.
- Check-in tepat 09:00:00 belum terlambat; setelahnya `terlambat`.
- Checkout sebelum 18:00 boleh dicatat sebagai `pulang_cepat`, bukan ditolak.
- Kelompok aktif mencakup status `aktif` dan `cuti`; kelompok nonaktif mencakup `nonaktif`
  dan `resign`.
- Karyawan baru memakai `tanggal_masuk` dalam periode.
- Resign memakai status `resign` dan `deleted_at` dalam periode.
- Turnover adalah resign dibagi rata-rata headcount awal/akhir dikali 100; denominator nol
  menghasilkan nol.
- Hari izin adalah irisan hari Senin-Jumat dari izin final disetujui dengan periode.
- Estimasi payroll mengalokasikan `take_home_pay` bulanan secara proporsional menurut hari
  Senin-Jumat yang beririsan dengan periode, tanpa kalender libur.
- Organization chart memakai employee aktif/cuti pada akhir periode dan relasi `atasan_id`.
- Gender kosong/tidak lengkap masuk `belum_diisi`, tidak dimasukkan ke `laki_laki` atau
  `perempuan`; gender wajib untuk data baru. Rasio memakai populasi aktif/cuti pada akhir
  periode dan tidak menyertakan resign.

### D-016 — Free choice of active WFO office

Karyawan tidak diikat ke satu kantor. `GET /api/v1/master/lokasi-kantor` menyediakan lokasi
aktif, dan request WFO wajib membawa `office_location_id` pilihan. Backend mengambil
koordinat tepercaya dari `office_locations`, bukan dari client, lalu menghitung radius 100 m.
Master boleh kosong sampai nama/alamat/koordinat resmi tersedia; WFO tidak dapat diproses
tanpa lokasi aktif, tetapi WFH/WFA tetap tidak memakai radius.

## Open contract gaps

### G-007 — Employee document delivery

Kontrak mengembalikan URL tetapi belum menetapkan expiry, signed URL, proxy download, atau
authorization ketika URL dibuka. Keputusan ini ditunda sesuai arahan produk. Implementasi
file delivery belum boleh membuat dokumen public permanen.

### G-010 — Company and public holiday calendar

Hari kerja reguler telah ditetapkan Senin-Jumat, tetapi kalender libur nasional/perusahaan
belum tersedia. Sampai ada keputusan, kontrak hanya memakai weekday dan tidak mengarang
kalender libur. Pengecualian kerja pada akhir pekan/libur memerlukan revisi policy.

## Validation record

Hasil validasi lokal revisi 0.4.0:

- YAML syntax: valid dengan PyYAML 6.0.3.
- OpenAPI root: `3.1.0`; metadata wajib setiap operation tersedia.
- Path count: 38.
- Operation count: 46.
- Operation ID: lengkap dan unik.
- Response coverage: setiap operation memiliki response 2xx dan 4xx.
- `$ref` resolution: seluruh local reference terselesaikan.
- Redocly: baseline 0.1.0 sebelumnya lulus tanpa warning; lint ulang 0.4.0 belum dapat
  dijalankan karena eksekusi paket pihak ketiga di luar sandbox ditolak oleh kebijakan
  keamanan lingkungan. Pemeriksaan aman lokal memakai PyYAML 6.0.3.
