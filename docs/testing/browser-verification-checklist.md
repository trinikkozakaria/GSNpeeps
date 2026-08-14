# Browser Verification Checklist — GSNpeeps

> Status otomatis 14 Agustus 2026: backend test lulus, frontend **207/207** lulus,
> production build berhasil, build smoke **44 lulus / 50 dilewati / 0 gagal**, dan
> UAT terisolasi **6/6 lulus**.
> Checkbox di bawah tetap merupakan bukti verifikasi browser/manual; hasil otomatis tidak
> otomatis mencentang journey yang belum diperiksa manusia.

### Akun fixture

| Email | Role | Catatan |
|---|---|---|
| `karyawan@example.test` | karyawan | punya atasan (`SYN-ATASAN-001`) |
| `karyawan.tanpa.atasan@example.test` | karyawan | tanpa atasan, approval langsung ke HR |
| `atasan@example.test` | atasan | approver bawahan langsung |
| `hr@example.test` | hr | akses penuh |
| `top.management@example.test` | top_management | beranda ringkas, notifikasi, dan final approval milik HR |

Password seluruh akun adalah nilai `SEED_PASSWORD` di `.env` lokal (minimal 12 karakter).
Nilainya tidak pernah ditulis ke repository.

### Master data sintetis

- Kantor `OFFICE-SYN-001` pada koordinat `-6.2000000, 106.8000000`. Koordinat ini fiktif
  dan hanya untuk menguji aturan radius WFO.
- Jenis izin `IZIN-SYN-E2E` (tanpa kuota, dokumen tidak wajib).
- Jenis izin `CUTI-SYN-E2E` (kuota 12, dokumen wajib).

Untuk menguji WFO, gunakan geolocation override DevTools ke koordinat kantor sintetis.

---

## 1. Authentication dan session

- [x] `/login` berhasil untuk kelima akun; redirect ke `/app` dan landing sesuai role.
- [ ] Lima kegagalan login berturut-turut mengunci akun dan menampilkan `429 ACCOUNT_LOCKED`.
- [ ] Login sukses mereset counter kegagalan.
- [ ] `/reset-password` menerima email, password saat ini, password baru, dan konfirmasi.
- [ ] Setelah self-reset berhasil, akun terbuka, seluruh session dicabut, dan pengguna
      wajib login ulang.
- [ ] Self-reset dengan password saat ini yang salah menghasilkan error generik, bukan
      pesan yang membedakan akun ada atau tidak.
- [ ] `/app/keamanan` dapat mengganti password sendiri.
- [ ] Reload browser mengakhiri sesi lokal dan mengarahkan ke `/login`. Ini perilaku yang
      disengaja; token hanya disimpan di memori tab. Lihat
      `docs/architecture/frontend-auth-session.md`.
- [x] Setelah logout, membuka kembali URL `/app/...` ditolak.

## 2. Guard dan negative authorization

Uji dengan mengetik URL langsung, bukan lewat menu. Hidden menu hanya UX; backend tetap
sumber otorisasi.

- [x] Karyawan membuka `/app/karyawan`, `/app/dashboard`, `/app/akses`, `/app/audit` →
      `/forbidden`.
- [x] Top Management membuka `/app/profil`, `/app/metrik-personal`, `/app/absensi` →
      forbidden. Top Management tidak memiliki Metrik Personal.
- [x] Top Management membuka `/app/karyawan/baru`, `/app/karyawan/:id/edit`,
      `/app/master/jenis-izin`, `/app/lembur/rekap` → forbidden. Route tersebut HR-only.
- [x] Atasan membuka `/app/karyawan` dan `/app/live-feed` → forbidden.
- [ ] URL tidak dikenal di dalam `/app` menampilkan NotFound, bukan crash.
- [ ] Request yang ditolak tidak menimbulkan side effect di database, storage,
      notification, maupun audit.

## 3. Karyawan dan Atasan — journey personal

Route personal terbuka untuk `karyawan`, `atasan`, dan `hr`.

- [x] `/app/profil` menampilkan data pribadi. Response tidak pernah memuat `password_hash`.
- [x] `/app/metrik-personal` menampilkan metrik personal. 


### Kehadiran — `/app/absensi`

Area dengan cabang terbanyak; verifikasi seluruh kombinasi.

- [x] Mode WFO dengan koordinat di dalam radius 100 m dari kantor terpilih → sukses.
- [x] Mode WFO di luar radius → ditolak dengan pesan yang jelas.
- [x] Mode WFH tidak dibatasi radius kantor, koordinat tetap wajib.
- [x] Mode WFA tidak dibatasi radius kantor, koordinat tetap wajib.
- [x] Karyawan dapat memilih kantor aktif saat WFO; tidak ada assignment kantor permanen.
- [x] Izin kamera ditolak → fallback upload foto berwatermark tersedia.
- [x] Izin geolocation ditolak → error state yang dapat dipahami, submit dicegah.
- [x] Check-in ganda pada hari yang sama ditolak.
- [x] Checkout tanpa check-in ditolak.
- [x] Check-in tepat 09:00:00 WIB tidak berstatus terlambat.
- [x] Check-in setelah 09:00:00 WIB berstatus `terlambat`.
- [x] Checkout sebelum 18:00 WIB tetap valid dan tercatat `pulang_cepat`.
- [x] Tidak ada reminder absensi. Fitur ini memang di luar scope.

### Pengajuan

- [x] `/app/absensi/ketidakhadiran` dapat mengajukan Cuti, Izin, dan Perjalanan Dinas.
- [x] Dokumen pendukung wajib untuk semua jenis ketidakhadiran.
- [x] Perjalanan Dinas juga mewajibkan lokasi tujuan dan keperluan tugas.
- [x] File lebih dari 5 MB ditolak.
- [x] Ekstensi atau MIME type tidak valid ditolak.
- [x] `/app/absensi/lembur` menerima dokumen pendukung opsional.
- [x] Tidak ada kalkulasi kompensasi atau uang lembur di UI.
- [x] `/app/pengajuan` menampilkan riwayat pengajuan sendiri dan status ter-update setelah
      diputus. 

## 4. Approval

- [x] `/app/persetujuan` menampilkan inbox approver.
- [x] Atasan hanya melihat pengajuan bawahan langsung.
- [x] `/app/persetujuan/ketidakhadiran/:id` dan `/app/persetujuan/lembur/:id` dapat dibuka
      oleh approver tahap aktif.

Jalur routing yang perlu dilalui seluruhnya:

- [x] Karyawan dengan atasan → Atasan → HR.
- [x] Karyawan tanpa atasan (`karyawan.tanpa.atasan@example.test`) → langsung HR.
- [x] Atasan mengajukan → HR.
- [x] HR mengajukan → Top Management.

Aturan keputusan:

- [x] Reject mengakhiri alur dan mewajibkan catatan.
- [x] Approve Atasan memindahkan pengajuan ke HR.
- [x] Delegasi ke HR tersedia untuk ketidakhadiran. Lembur tidak memiliki delegasi;
      kontrak tidak menyediakannya.
- [x] Membuka detail yang sama di dua tab lalu approve keduanya menghasilkan
      `409 ALREADY_DECIDED` pada tab kedua, bukan dua keputusan.
- [x] Tombol submit disabled selama mutation berlangsung sehingga double submit dicegah.
- [ ] Auto-escalation memindahkan request `menunggu_atasan` yang melebihi 48 jam ke HR.
      Perlu manipulasi timestamp atau menunggu `cron-worker`.
- [ ] Tidak ada auto-escalation dari HR ke Top Management.

## 5. HR — Employee Database

- [x] `/app/karyawan` mendukung search, filter, dan pagination.
- [ ] Filter dan pagination bertahan di URL saat reload atau share link.
- [x] `/app/karyawan/baru` memvalidasi input via Zod; NIP dan email duplikat memunculkan
      error per field.
- [x] `/app/karyawan/:id` menampilkan detail beserta dokumen karyawan dan preview gambar/PDF.
- [x] Upload dokumen karyawan berhasil dan menghormati batas 5 MB.
- [x] `/app/karyawan/:id/edit` dapat memperbarui data.
- [x] Delete melakukan soft-delete: `status='nonaktif'` dan `deleted_at` terisi, record
      tidak hilang dari database.
- [ ] Karyawan nonaktif terpisah dari hitungan dan komposisi departemen karyawan aktif.
- [x] Export karyawan menghasilkan file yang terunduh.
- [x] Bulk CSV membuat akun dan email `@janjikupadamu.id`, serta menampilkan hasil per baris.

## 6. HR — monitoring

- [ ] `/app/dashboard` mendukung filter periode harian, mingguan, bulanan, dan tahunan
      dengan tanggal acuan.
- [ ] Boundary kalender memakai Asia/Jakarta dan minggu dihitung Senin sampai Minggu.
- [x] Kehadiran pada dashboard hanya dihitung jika terdapat clock-in dan clock-out pada
      tanggal yang sama.
- [ ] Gender kosong masuk kategori `belum_diisi`, bukan laki-laki atau perempuan.
- [ ] Kartu Hiring Progress, Recruitment Cost, dan Benefit tetap placeholder Coming Soon.
- [x] `/app/live-feed` menampilkan aliran check-in untuk HR; Top Management ditolak.
- [x] `/app/laporan-kehadiran` mendukung filter dan export untuk HR; Top Management ditolak.
- [x] `/app/master/jenis-izin` dapat membuat dan mengubah jenis izin. HR only.
- [ ] `/app/lembur/rekap` menampilkan rekap lembur. HR only.

## 7. Akses dan Audit Log

- [x] `/app/akses` menampilkan role dan permission untuk HR.
- [x] HR dapat mengubah permission dan perubahannya tersimpan.
- [x] Top Management tidak dapat membuka atau mengubah matriks akses (`403`) sesuai aturan
      role terbaru.
- [ ] `/app/audit` read-only tanpa aksi edit maupun hapus.
- [ ] Audit Log mencatat login, logout, create, update, delete, approve, reject, download,
      dan permission change.
- [ ] Audit Log tidak memuat secret maupun isi dokumen.

## 8. Notifikasi

- [x] `/app/notifikasi` menampilkan daftar notifikasi milik recipient sendiri.
- [x] Badge unread count akurat.
- [x] Mark as read mengurangi badge.
- [x] Dismiss bersifat soft-dismiss dan mengisi `dismissed_at`.
- [ ] Event yang sudah di-dismiss tidak dibuat ulang.
- [ ] Deep link dari notifikasi membuka detail pengajuan yang benar.
- [x] Pengajuan baru mengirim notifikasi ke approver aktif.
- [ ] Perubahan status mengirim notifikasi ke pemohon dan approver tahap berikutnya.
- [ ] Tidak pernah terjadi self-notify.
- [ ] Notifikasi kontrak H-30 dikirim ke atasan aktif dan seluruh HR aktif selain subjek,
      dengan fallback ke satu Top Management aktif bila tidak ada HR aktif lain.

## 9. State UI lintas halaman

Untuk setiap halaman utama, pastikan tersedia dan benar:

- [ ] Loading state.
- [ ] Empty state.
- [ ] Validation error dengan pesan per field.
- [ ] 401, 403, 409, 422, 429, dan 500 state.
- [x] Layout responsif pada viewport mobile.
- [x] Navigasi keyboard penuh dan focus state terlihat.
- [x] Label terlihat pada seluruh input dan kontrol semantic.
- [ ] Status tidak dibedakan hanya lewat warna.
- [ ] Tabel dapat digunakan pada layar sempit.
- [x] Data sensitif tidak dirender atau di-cache sebelum hak akses dipastikan.

---

## Prioritas bila waktu terbatas

1. Kehadiran: seluruh mode WFO/WFH/WFA plus fallback kamera dan geolocation.
2. Jalur approval untuk setiap kombinasi role, termasuk delegasi dan double-decision.
3. Negative authorization melalui URL langsung untuk keempat role.

Tiga area tersebut memiliki cabang terbanyak dan konsekuensi terbesar bila lolos ke rilis.
