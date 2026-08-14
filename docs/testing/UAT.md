# Status verifikasi terbaru

Terakhir diverifikasi: **14 Agustus 2026**.

Penanda status:

- `[x]` = implementasi sudah selesai dan tercakup verifikasi teknis.
- `[ ]` = belum selesai atau masih memerlukan tindakan/verifikasi manual.

Status saat ini: **seluruh implementasi yang memiliki requirement sudah selesai dan automated
verification lulus; sign-off UAT manual oleh pengguna tetap perlu dilakukan**. Rincian
pengujian manual dicatat di
[`browser-verification-checklist.md`](./browser-verification-checklist.md); jangan menganggap
item yang belum dicentang sebagai sudah lolos.

Yang masih belum selesai:

- [ ] Sign-off seluruh item UAT manual pada `browser-verification-checklist.md`.
- [ ] Unggah ulang foto profil lama yang sebelumnya gagal tersimpan ketika Nextcloud belum aktif.
- [ ] Aktivasi kartu Hiring Progress, Recruitment Cost, dan Benefit setelah requirement serta
  sumber datanya tersedia; untuk saat ini ketiganya tetap **Coming Soon** dan bukan bug.

Hasil verifikasi teknis terakhir:

- [x] `http://localhost:8080/` merespons HTTP `200`.
- [x] `http://localhost:8080/health` merespons HTTP `200`; PostgreSQL dan Redis berstatus `ok`.
- [x] Seluruh test backend (`go test ./...`) lulus.
- [x] Seluruh unit test frontend lulus: **208/208** dari **33** test file.
- [x] Production build frontend berhasil.
- [x] Playwright build smoke: **44 lulus, 50 dilewati sesuai flag/project, 0 gagal**.
- [x] Playwright UAT terisolasi: **6/6 lulus**, mencakup approval, absensi, laporan,
  lifecycle/bulk karyawan, notifikasi/akses/audit, dan kegagalan Nextcloud.
- [x] Migrasi database perbaikan skema UAT `00011_repair_uat_schema.sql` sudah diterapkan.
- [x] Seed SOP jenis izin pada migrasi `00012` dan `00013` sudah diterapkan.

Gap UI yang ditutup pada verifikasi ini:

- [x] Sidebar sudah dikelompokkan per modul dengan child menu yang mengikuti role.
- [x] Upload dokumen karyawan hanya menerima jenis dokumen aktif dari master data.
- [x] Foto profil dapat diganti langsung dari halaman Profil Saya dan navbar ikut diperbarui.
- [x] Koreksi absensi tersedia sebagai antrean di Persetujuan untuk Atasan dan HR.
- [x] Ringkasan beranda memiliki loading, error, retry, empty, dan data state yang jelas.
- [x] Company Feed dan Kalender memiliki loading, error, empty/success, retry, serta pencegahan
  submit berulang.
- [x] Master Jenis Izin dapat diubah HR; kategori dan kode dikunci agar histori tetap konsisten.
- [x] Form izin menampilkan batas maksimal setelah jenis dipilih, mengunci tanggal akhir maksimal,
  dan tetap mengizinkan durasi yang lebih pendek.
- [x] Dokumen gambar memiliki thumbnail dan dokumen PDF dapat dipratinjau melalui endpoint media
  terproteksi; format lain tetap dapat diunduh dengan aman.
- [x] Bulk upload karyawan menampilkan hasil per baris dan email perusahaan yang dibuat otomatis.

Jika `localhost:8080` kembali menampilkan `ERR_CONNECTION_REFUSED`, pastikan stack Compose
masih aktif. Pada verifikasi terakhir, pembuatan ulang container `nginx` memulihkan publikasi
port tanpa menghapus data PostgreSQL, Redis, atau Nextcloud.

---

## Checklist requirement UAT

- [x] HR dapat membuat jenis dokumen yang seragam untuk seluruh karyawan, misalnya Foto KTP,
  lalu setiap karyawan dapat mengunggah dokumen sesuai jenis tersebut.
- [x] Absensi dihitung hadir hanya ketika clock-in dan clock-out tersedia pada tanggal yang sama.
- [x] HR dapat menerbitkan Company Feed berformat WYSIWYG ke timeline seluruh karyawan.
- [x] Beranda menampilkan Company Feed, saldo cuti per jenis, pengajuan yang perlu disetujui,
  dan pengajuan ketidakhadiran pribadi.
- [x] Ketidakhadiran membedakan izin tanpa saldo dan cuti yang mengurangi saldo. Batas maksimal
  izin dapat diatur melalui Master Jenis Izin dan diterapkan otomatis pada tanggal pengajuan.
- [x] Komposisi departemen menggunakan pie chart berisi headcount tiap departemen.
- [x] Rasio gender menggunakan ikon gender.
- [x] Kalender menampilkan hari libur dan mendukung bulk insert/bulk update tanggal.
- [x] Profil Saya menampilkan usia berdasarkan tanggal lahir.
- [x] Ajukan Ketidakhadiran dan Ajukan Lembur menjadi submenu dari Pengajuan.
- [x] Sidebar dapat di-collapse; menu ringkas tetap dapat diklik untuk membuka submenu. Sesuai
  instruksi UI terbaru, sidebar terbuka ditutup dengan klik di luar sidebar, bukan hover.
- [x] Foto profil dapat ditambahkan, dimuat melalui media terproteksi, dan tampil di navbar.
- [x] Ukuran tombol utama menggunakan ukuran medium.
- [x] Top Management tidak melihat Employee Database, Dashboard HR, Live Feed Absensi,
  Laporan Kehadiran, Akses, dan Audit Log.
- [x] Status terlambat dan pulang cepat menggunakan tampilan danger, bukan success.
- [x] Tombol Tambah Karyawan sejajar dengan kontrol export pada Employee Database.
- [x] Form pendidikan memiliki tahun masuk dan tahun lulus opsional untuk status masih belajar.
- [x] Waktu ditampilkan dalam format 24 jam zona Asia/Jakarta; penyimpanan sistem memakai UTC.
- [x] Nilai uang menggunakan format Rupiah dengan delimiter.
- [x] Database karyawan mendukung bulk upload.
- [x] Bulk upload otomatis membuat akun email perusahaan dan menangani nama yang duplikat.
- [x] Gambar yang diunggah memiliki preview dan PDF dapat ditampilkan inline.
- [x] Koreksi clock-in/clock-out menggunakan approval berjenjang Karyawan → Atasan → HR.
