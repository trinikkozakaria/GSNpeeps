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

### D-017 — Export format enum is `xlsx` atau `pdf`

`ExportFormatParam` revisi 0.4.0 memakai enum `xlsx|csv`, sedangkan kedua operation export
(`GET /api/v1/karyawan/export` dan `GET /api/v1/laporan/kehadiran/export`) hanya mendeklarasikan
response `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` dan
`application/pdf`. API Contract v1.1 §5.6 menetapkan `format=xlsx|pdf`.

Karena API Contract memiliki otoritas lebih tinggi dan `csv` tidak memiliki response media type
di kontrak manapun, enum diselaraskan menjadi `xlsx|pdf` dengan default `xlsx`. Nilai `format`
di luar enum menghasilkan `400 VALIDATION_ERROR`. Tidak ada format export lain yang ditambahkan.

### D-018 — Kolom detail karyawan yang dibutuhkan OpenAPI 0.4.0

Database Schema v1.1 §3 mendefinisikan tabel detail karyawan lebih ringkas daripada schema
`EmployeeDetail`, `CurrentSalary`, `PositionHistory`, dan `EmployeeDocument` pada OpenAPI 0.4.0.
Selisihnya:

| Schema OpenAPI | Field | Status di Database Schema v1.1 |
|---|---|---|
| `CurrentSalary` | `tunjangan`, `potongan`, `take_home_pay` | tidak ada |
| `PositionHistory` | `departemen`, `jabatan` sebagai objek `Department`/`Position` | hanya `jabatan VARCHAR(100)` |
| `EmployeeDocument` | `nama_file`, `created_at` | hanya `file_url`, `uploaded_at` |
| `EmployeeNPWP` | `status_ptkp` | tidak ada |
| `EducationHistory` | `jurusan` | tidak ada |
| `EmergencyContact` | `is_primary` | tidak ada |

Keputusan:

1. `take_home_pay` wajib ada karena formula payroll D-015 memakainya. Migration menambahkan
   `tunjangan`, `potongan`, dan `take_home_pay` pada `employee_salaries`. `take_home_pay`
   adalah kolom generated `gaji_pokok + tunjangan - potongan` sehingga tidak dapat menyimpang.
2. `employee_position_history` menambahkan `department_id` dan `position_id` nullable agar
   objek `Department`/`Position` dapat dibentuk tanpa menebak. Kolom `jabatan VARCHAR(100)`
   dari schema dipertahankan sebagai label historis.
3. `employee_documents` menambahkan `nama_file`. Kolom `uploaded_at` dari schema dipertahankan
   dan dipetakan ke field response `created_at`; tidak ada kolom kedua yang dibuat.
4. `status_ptkp`, `jurusan`, dan `is_primary` tidak termasuk `required` pada schema manapun dan
   belum memiliki sumber data. Field tersebut tidak dikirim sampai sumbernya ditetapkan;
   kolom database tidak ditambahkan hanya untuk mengisi field opsional.
5. Nama tabel mengikuti Database Schema v1.1: `employee_emergency_contacts`,
   `employee_education`, `employee_position_history`, `employee_salaries`, dan
   `employee_documents`.

### D-019 — Status karyawan pada agregasi dashboard

D-015 menyebut kelompok aktif mencakup `aktif`/`cuti` dan kelompok nonaktif mencakup
`nonaktif`/`resign`. `employees.status` pada Database Schema v1.1 dan enum `EmployeeStatus`
pada OpenAPI hanya memiliki `aktif` dan `nonaktif`; `cuti` dipetakan ke `aktif` dan `resign`
dipetakan ke `nonaktif` pada boundary HTTP maupun database.

Implementasi memakai pemetaan tersebut: `karyawan_aktif` menghitung `status='aktif'`,
`karyawan_nonaktif` menghitung `status='nonaktif'`, dan `resign` menghitung karyawan dengan
`deleted_at` di dalam rentang periode. Kolom status terpisah untuk `cuti`/`resign` tidak
ditambahkan tanpa keputusan produk.

### D-020 — Metrik yang bergantung pada modul Attendance dan Ketidakhadiran

`PersonalMetrics` seluruhnya, serta `hadir_valid`, `terlambat`, `hari_izin_disetujui`, dan
`pengajuan_menunggu` pada `DashboardMetrics`, membaca tabel `attendances`, `leave_requests`,
dan `overtime_requests` yang belum dibuat pada epic ini.

Endpoint tetap diimplementasikan dengan authorization, boundary periode, dan response shape
sesuai kontrak. Nilai yang bergantung pada modul tersebut dibaca melalui interface
`AttendanceMetricsReader` dan `LeaveMetricsReader` dengan implementasi sementara yang
mengembalikan nol dan koleksi kosong — bukan data buatan. Implementasi nyata dipasang pada
epic Attendance dan Approval tanpa mengubah response schema.

### D-021 — Nama status pengajuan mengikuti OpenAPI

Database Schema v1.1 menuliskan `menunggu_topmanagement` tanpa underscore, sedangkan D-005
dan schema `RequestStatus` memakai `menunggu_top_management`. API Contract memiliki otoritas
lebih tinggi, sehingga database, query, dan response memakai `menunggu_top_management`.

CHECK constraint juga menyertakan `dibatalkan` karena nilai tersebut ada pada `RequestStatus`
meskipun belum ada operasi yang memicunya pada kontrak aktif. Tidak ada endpoint pembatalan
yang ditambahkan.

### D-022 — Kolom absensi mengikuti schema `Attendance`

Schema `Attendance` memuat `office_location_id` dan `distance_meters` yang tidak ada pada
Database Schema v1.1, serta enum status `tepat_waktu|terlambat|pulang_cepat|valid` sedangkan
schema database menuliskan `('tepat_waktu','telat')`. Migration mengikuti OpenAPI:

- `office_location_id` nullable dengan FK ke `office_locations`, terisi hanya untuk WFO;
- `distance_meters` nullable, hasil hitung server untuk WFO;
- CHECK status memakai keempat nilai OpenAPI.

`AttendanceCheckRequest` tidak memiliki field `waktu_local`, sedangkan kolomnya `NOT NULL`.
Karena waktu perangkat tidak tepercaya dan menambah request field memerlukan perubahan
kontrak, backend mengisi `waktu_local` dari waktu server yang dikonversi ke Asia/Jakarta.
Tidak ada timestamp dari client maupun dari metadata gambar yang dipakai sebagai sumber waktu.

### D-023 — Master jenis izin memakai kode dan flag dokumen

Schema `LeaveType` memuat `kode`, `kuota_tahunan`, `memerlukan_dokumen`, dan `is_active`,
sedangkan Database Schema v1.1 hanya memiliki `nama`, `kuota_hari`, dan `aktif`. Migration
mengikuti OpenAPI: `kode` unik, `kuota_tahunan`, `memerlukan_dokumen`, dan `is_active`.

Seed 15 jenis izin belum dibuat karena daftar resmi berada pada User Story US-33 yang tidak
memuat nama dan kuota lengkap pada sumber yang tersedia. Menebak nama atau kuota dilarang,
sehingga seed ditunda dan dicatat sebagai gap G-011.

### D-024 — Dokumen pendukung ketidakhadiran bersifat kondisional

CLAUDE.md menyatakan dokumen pendukung wajib untuk semua jenis, sedangkan `CreateLeaveRequest`
menyatakan dokumen wajib hanya bila master jenis izin mensyaratkannya dan `LeaveType`
menyediakan `memerlukan_dokumen`. Kontrak API lebih spesifik dan dapat dieksekusi, sehingga:

- `leave_requests.dokumen_url` nullable;
- backend menolak request tanpa dokumen ketika `memerlukan_dokumen` bernilai true;
- seluruh jenis izin dapat dikonfigurasi mewajibkan dokumen melalui master.

Field `keperluan_tugas` pada Database Schema dikirim sebagai `keterangan_lokasi` di
`LeaveRequestDetail`. Nama kolom database dipertahankan sesuai schema dan pemetaan dilakukan
pada lapisan response; tidak ada kolom kedua yang dibuat.

### D-025 — Riwayat approval memuat tahap Top Management dan auto-escalation

`ApprovalHistory` revisi 0.4.0 membatasi `tahap` pada `atasan|hr` dan `keputusan` pada
`disetujui|ditolak|didelegasikan`. Alur approval yang ditetapkan PRD mewajibkan tahap
`top_management` untuk pengajuan milik HR dan keputusan sistem `auto_escalate` setelah SLA
2x24 jam. Tanpa kedua nilai tersebut riwayat tidak dapat merepresentasikan alur yang sudah
disetujui.

Schema diperbarui: `tahap` menerima `top_management` dan `keputusan` menerima `auto_eskalasi`.
`approver_id` dan `approver_nama` menjadi nullable karena keputusan sistem tidak memiliki
approver. Nilai database memakai `approve|reject|delegate|auto_escalate` sesuai Database
Schema dan dipetakan ke nilai response di atas.

### D-026 — Parameter periode laporan kehadiran

`AttendancePeriodParam` bertipe `YearMonth`, bukan enum `harian|mingguan|bulanan|custom`.
Implementasi mengikuti OpenAPI:

- `tanggal_mulai` dan `tanggal_selesai` keduanya terisi menghasilkan rentang custom;
- hanya salah satu terisi menghasilkan `400 VALIDATION_ERROR`;
- tanpa rentang, `periode=YYYY-MM` menghasilkan satu bulan kalender;
- tanpa keduanya, laporan memakai bulan berjalan.

Seluruh boundary dihitung pada Asia/Jakarta.

### D-027 — Tidak ada delegasi lembur

API Contract dan OpenAPI hanya menyediakan `PUT /api/v1/lembur/{id}/decision`. Tidak ada
operasi delegasi lembur, sehingga endpoint tersebut tidak dibuat. Auto-escalation tetap
berlaku untuk lembur karena dipicu worker, bukan endpoint. Bila delegasi lembur dibutuhkan,
kontrak harus direvisi lebih dahulu.

### D-028 — Kepemilikan watermark foto absensi

Watermark adalah tanggung jawab frontend. Backend tidak membuat maupun menormalisasi
watermark, tidak membaca timestamp dari isi gambar, dan tidak menambah field request untuk
metadata watermark. Backend memvalidasi tipe dan ukuran berkas, menyimpan foto melalui
Nextcloud dengan path yang dibentuk server, dan mencatat `waktu_network` dari jam server
sebagai satu-satunya sumber waktu absensi.

### D-029 — Audit log dapat memiliki aktor sistem

Schema `AuditLog` mewajibkan `user_id` bertipe UUID, sedangkan auto-escalation dan job
terjadwal menulis audit tanpa pengguna. Baris tersebut sudah ada sejak BE.4 dan kontrak
lama tidak dapat merepresentasikannya. OpenAPI diubah sehingga `user_id` dan `nama_user`
menerima `null`. Response menandai aktor sistem dengan `user_id: null`; frontend
menampilkannya sebagai "Sistem". Tidak ada aktor palsu yang dibuat hanya agar field terisi.

### D-030 — Kolom `judul` dan `read_at` pada tabel notifications

Database Schema mendaftar `tipe`, `pesan`, `event_key`, `is_read`, dan `dismissed_at`,
tetapi schema response `Notification` mewajibkan `judul` dan menyediakan `read_at`.
Kedua kolom ditambahkan pada migration agar response dapat dipenuhi tanpa mengarang nilai
saat pembacaan. `read_at` dijaga konsisten dengan `is_read` melalui CHECK constraint.
`judul` dan `pesan` dibentuk server dari katalog event, bukan dari input pengguna.

### D-031 — Matriks permission memakai kosakata aksi kontrak

Seed BE.2 memakai aksi seperti `view_own`, `manage`, `approve_direct`, dan `monitor`,
sedangkan `Permission.aksi` pada OpenAPI dibatasi `create`, `read`, `update`, `delete`,
`approve`, dan `export`. Nilai lama tidak dapat dikembalikan maupun diperbarui melalui
endpoint AKSES. Baris permission dipetakan ulang ke kosakata kontrak dengan modul yang
sama dengan menu produk.

Cakupan kepemilikan (`view_own` versus `view`) tidak hilang: pembatasan pemilik, bawahan
langsung, dan tahap approval tetap ditegakkan service dan query repository, bukan oleh
matriks ini. Matriks adalah kontrol kasar per modul/aksi yang dapat diadministrasikan HR.

### D-032 — Notifikasi milik orang lain selalu menghasilkan 403

`PUT /api/v1/notifikasi/{id}/read` dan `DELETE /api/v1/notifikasi/{id}` mengembalikan 403
`FORBIDDEN` baik ketika notifikasi tidak ada maupun ketika dimiliki pengguna lain. Kode
yang seragam mencegah pemetaan ID milik orang lain melalui perbedaan 403 dan 404. Response
404 tetap tercantum pada kontrak namun tidak dipakai pada dua operasi tersebut.

### D-033 — Notifikasi ditulis dalam transaction yang sama, bukan melalui outbox

Notifikasi berada pada PostgreSQL yang sama dengan pengajuan, sehingga penulisan notifikasi
ikut serta dalam transaction keputusan approval. Kegagalan penulisan membatalkan keputusan
dan tidak menghasilkan event yang hilang diam-diam. Outbox tidak dibuat karena akan
menambah lifecycle pengiriman tanpa memberi jaminan tambahan pada satu database. Bila kanal
eksternal ditambahkan kelak, outbox baru diperlukan dan harus punya migration sendiri.

### D-034 — Matriks permission dikembalikan untuk seluruh role

`GET /api/v1/akses/permission` pada OpenAPI tidak memiliki parameter `role`, sedangkan
uraian task menyebut query role wajib. Kontrak diikuti: satu response memuat seluruh baris
permission empat role beserta `role_id` sehingga frontend dapat mengelompokkannya sendiri.
Jumlah baris terbatas dan diketahui, sehingga tidak diperlukan pagination.

## Open contract gaps

### G-007 — Employee document delivery

Kontrak mengembalikan URL tetapi belum menetapkan expiry, signed URL, proxy download, atau
authorization ketika URL dibuka. Keputusan ini ditunda sesuai arahan produk. Implementasi
file delivery belum boleh membuat dokumen public permanen.

### G-011 — Daftar resmi 15 jenis izin

Database Schema menyebut seed 15 jenis izin sesuai SOP dengan rujukan User Story US-33,
tetapi nama dan kuota lengkap tidak tersedia pada sumber yang ada. Master `leave_types`
dibuat kosong dan HR mengisinya melalui `POST /api/v1/master/jenis-izin`. Jangan menebak
nama maupun kuota jenis izin pada seed atau test.

### G-012 — Jenis izin tidak terbaca oleh pemohon (resolved 2026-08-11)

`GET /api/v1/master/jenis-izin` semula dibatasi HR pada API Contract, sedangkan Karyawan dan
Atasan memerlukan daftar jenis izin untuk mengisi `jenis_izin_id` pada
`POST /api/v1/ketidakhadiran`. Akibatnya pemohon selalu menerima `403` dan tidak dapat
mengajukan ketidakhadiran.

Keputusan produk: read dibuka untuk seluruh role terautentikasi tanpa menambah endpoint di
luar 46 operasi kontrak. Role selain HR selalu dipaksa ke jenis izin aktif dan parameter
`aktif` diabaikan, sehingga master yang sengaja dinonaktifkan HR tidak dapat diajukan.
`POST` dan `PUT` master tetap HR-only.

### G-010 — Company and public holiday calendar

Hari kerja reguler telah ditetapkan Senin-Jumat, tetapi kalender libur nasional/perusahaan
belum tersedia. Sampai ada keputusan, kontrak hanya memakai weekday dan tidak mengarang
kalender libur. Pengecualian kerja pada akhir pekan/libur memerlukan revisi policy.

## Validation record

Hasil validasi lokal setelah penerapan D-017 (enum `ExportFormatParam`):

- YAML syntax: valid dengan PyYAML.
- OpenAPI root: `3.1.0`; metadata wajib setiap operation tersedia.
- Path count: 38 (tidak berubah).
- Operation count: 46 (tidak berubah; tidak ada endpoint baru di luar kontrak).
- Operation ID: lengkap dan unik.
- Response coverage: setiap operation memiliki response 2xx dan 4xx.
- `$ref` resolution: seluruh local reference terselesaikan.
- `ExportFormatParam.enum`: `xlsx`, `pdf`.
- Redocly: baseline 0.1.0 sebelumnya lulus tanpa warning; lint ulang 0.4.0 belum dapat
  dijalankan karena eksekusi paket pihak ketiga di luar sandbox ditolak oleh kebijakan
  keamanan lingkungan. Pemeriksaan aman lokal memakai PyYAML 6.0.3.
