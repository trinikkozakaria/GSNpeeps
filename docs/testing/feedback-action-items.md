# Action Item List — Feedback UI HRIS GSNpeeps

**Sumber**: `docs/source/feedback.docx`
**Disusun**: 2026-08-21
**Metodologi**: teks dan seluruh gambar referensi (Current/Adjustment) di dalam file `.docx`
diekstrak dan dibandingkan satu-satu dengan tampilan/kode frontend saat ini (`frontend/src/...`)
agar setiap item mengacu ke lokasi file yang benar, bukan tebakan.

---

## Status Implementasi (2026-08-21)

- ✅ **Selesai: 12 item** — item 1–12.
- 🟡 **Sebagian: 0 item**.
- ⬜ **Belum dimulai: 0 item**.

Legenda: ✅ selesai · 🟡 sebagian · ⬜ belum.

## Keputusan Implementasi

Dua hal yang semula memerlukan klarifikasi sudah diselesaikan dengan keputusan implementasi
berikut:

1. ✅ **Urutan dropdown profil (navbar kanan atas)** — catatan tertulis di doc eksplisit menyebut
   urutan `Account Settings` lalu `Sign Out`, tetapi mockup gambar pada doc yang sama justru
   menampilkan `Sign Out` di atas dan `Account Settings` di bawah. Lihat item 4.
   Implementasi mengikuti catatan tertulis: `Pengaturan Akun`, lalu `Keluar`.
2. ✅ **Upload file/gambar di Company Feed** — endpoint/field untuk attachment (PDF/PNG/JPG) pada
   Company Feed belum ada di API Contract v1.1 / daftar 51 endpoint (CLAUDE.md §7). Ini butuh
   akhirnya ditambahkan dengan keputusan implementasi: maksimal 5 attachment per feed, maksimal
   5 MB per file, tipe PDF/PNG/JPG/JPEG, validasi extension+MIME+signature di backend, dan storage
   melalui driver aktif (MinIO secara default). Lihat item 11.

---

## Ringkasan Prioritas

| Prioritas | Jumlah item | Area |
|---|---|---|
| Tinggi | 1 | Company Feed — upload file & bug Bold/Italic |
| Sedang | 6 | Login logo, Navbar kiri, Navbar kanan, Sidebar, Pagination, Donut chart, Font |
| Rendah | 5 | Favicon, Beranda/Home, Bulk Upload border, Org chart border |

---

## 1. Favicon

**Status: ✅ Selesai** — `logo.svg` dipakai sebagai favicon melalui tag `<link rel="icon">`.

- **Current**: tab browser menampilkan ikon default (globe), tidak ada favicon HRIS.
- **Adjustment**: tab browser menampilkan logo GSN.
- **Action items**:
  - Minta source logo GSN resolusi tinggi/vector (SVG) ke tim desain — asset di `.docx` hanya
    berupa raster PNG hasil export, belum ada file logo resmi di repo.
  - Tambahkan `favicon.svg`/`favicon.ico` ke `frontend/public/`.
  - Tambahkan tag `<link rel="icon">` di `frontend/index.html`.
- **File terdampak**: `frontend/index.html`, `frontend/public/`
- **Prioritas**: Rendah

## 2. Login Page — Logo GSN

**Status: ✅ Selesai** — logo GSN tampil bersama wordmark pada kartu autentikasi.

- **Current**: `LoginPage.jsx` hanya menampilkan teks "HR INFORMATION SYSTEM" / "GSNpeeps" tanpa
  logo.
- **Adjustment**: tambahkan logo GSN pada login page (berdampingan dengan wordmark teks).
- **File terdampak**: `frontend/src/modules/auth/pages/LoginPage.jsx`
- **Prioritas**: Sedang
- **Bergantung pada**: asset logo dari item 1.

## 3. Header/Navbar — kiri atas

**Status: ✅ Selesai** — logo GSN ditambahkan di kiri wordmark header.

- **Current**: `AppShell.jsx` hanya menampilkan teks "GSNpeeps" + "HR INFORMATION SYSTEM", tanpa
  logo.
- **Adjustment**: tambahkan logo GSN di kiri wordmark, konsisten dengan login page.
- **File terdampak**: `frontend/src/components/layout/AppShell.jsx`
- **Prioritas**: Sedang
- **Bergantung pada**: asset logo dari item 1.

## 4. Header/Navbar — kanan atas (profil & dropdown)

**Status: ✅ Selesai** — nama semi-bold, avatar/nama menjadi trigger, chevron tersedia, dan
dropdown berisi `Pengaturan Akun` lalu `Keluar`.

- **Current**: menampilkan bell icon, nama pengguna bold + role di bawahnya, avatar placeholder,
  tombol "Keluar" langsung terlihat tanpa interaksi dropdown.
- **Adjustment**:
  - Ubah bobot teks nama dari bold menjadi **semi-bold**.
  - Ubah interaksi: blok nama+avatar diklik → memunculkan dropdown (bukan tombol Keluar yang
    selalu tampil).
  - Tambahkan chevron icon di sebelah nama sebagai indikator dropdown.
  - Isi dropdown: "Account Settings" dan "Sign Out" — **urutan perlu konfirmasi**, lihat
    catatan konflik di atas. Sarankan label tetap Bahasa Indonesia ("Pengaturan Akun", "Keluar")
    mengikuti konvensi CLAUDE.md §5, bukan istilah Inggris pada mockup.
- **File terdampak**: `frontend/src/components/layout/AppShell.jsx`
- **Prioritas**: Sedang

## 5. Beranda/Home (halaman ringkasan setelah login)

**Status: ✅ Selesai** — border kartu Ringkasan dan placeholder Company Feed memakai border
solid abu-abu muda.

- **Current vs Adjustment**: secara layout identik; perbedaan hanya polish visual minor:
  - Border card "Ringkasan" (Perlu disetujui / Ketidakhadiran pribadi / Saldo cuti) diperhalus
    dari border gelap solid menjadi border abu-abu muda.
  - Placeholder "Belum ada informasi perusahaan" diubah dari dashed border menjadi solid border
    tipis.
- **File terdampak**: halaman landing role (`frontend/src/modules/auth/pages/RoleLandingPage.jsx`)
  dan komponen card ringkasan terkait — verifikasi nama komponen persis saat implementasi.
- **Prioritas**: Rendah

## 6. Sidebar

**Status: ✅ Selesai** — branding duplikat di sidebar dihapus sesuai revisi terakhir, label dan
proporsi menu diselaraskan dengan referensi, submenu regular, grup dapat dibuka/tutup, dan
submenu aktif memakai teks biru. Ikon Administrasi/Akun memakai ikon outline dari sistem ikon
internal agar konsisten.

- **Current**: menu aktif "Pribadi" memakai background biru solid saat expanded; submenu
  ("Profil Saya", dst.) bold; ikon "Administrasi" dan "Akun" masih generik.
- **Adjustment**:
  - Tambahkan logo GSN di kiri atas sidebar.
  - Pastikan seluruh label menu rata kiri di semua state.
  - Ubah bobot teks submenu ("Profil Saya", "Metrik Personal", "Kehadiran Saya", "Koreksi
    Absensi") dari bold menjadi regular.
  - Saat menu "Pribadi" diklik, submenu tampil sebagai dropdown list (bukan sekadar highlight
    penuh biru).
  - Saat submenu "Profil Saya" aktif/diklik, warna teksnya otomatis menjadi biru.
  - Ganti ikon "Administrasi" dan "Akun" dengan ikon yang lebih proporsional — **icon set final
    belum ditentukan di feedback**, perlu referensi tambahan dari desainer atau pilih dari
    icon set yang sudah dipakai di komponen lain agar konsisten.
- **File terdampak**: `frontend/src/routes/navigation/navigation.js`,
  `frontend/src/components/layout/AppShell.jsx`
- **Prioritas**: Sedang

## 7. Employee Database — border tombol "Bulk Upload"

**Status: ✅ Selesai** — seluruh tombol menggunakan varian secondary dengan border yang sama dan
label telah menjadi `Bulk Upload`.

- **Current**: tombol "XLSX" dan "PDF" tidak memiliki border yang terlihat (fill saja), tombol
  "Bulk upload" punya border sehingga tampak tidak seragam.
- **Adjustment**: samakan style border ketiga tombol (XLSX, PDF, Bulk Upload) dan perbaiki
  kapitalisasi label menjadi "Bulk Upload".
- **File terdampak**: `frontend/src/components/data-table/ExportButton.jsx`,
  `frontend/src/modules/employees/components/EmployeeBulkUpload.jsx`
- **Prioritas**: Rendah

## 8. Employee Database — fitur pagination "Halaman"

**Status: ✅ Selesai** — tersedia selector compact 10/20/50/100 dengan model `Showing ... from
...`, nomor halaman dalam kotak, navigasi chevron dengan `aria-label`, serta `page` dan `limit`
disimpan di URL.

- **Current**: `Pagination.jsx` hanya menampilkan teks "Halaman X dari Y · Z data" plus tombol
  teks "Sebelumnya"/"Berikutnya".
- **Adjustment**: tambahkan page-size selector ("Showing [100 ▾] from N employee") dan navigasi
  halaman berbentuk chevron icon di samping indikator "halaman X dari Y".
- **Action items**:
  - Tambahkan dropdown jumlah data per halaman (opsi umum: 10/20/50/100) yang mengubah param
    `limit` sesuai `meta.limit` pada Standard API Response (CLAUDE.md §5).
  - Ganti tombol teks prev/next menjadi ikon chevron, tetap sediakan `aria-label` untuk
    aksesibilitas karena label teks dihilangkan secara visual.
  - Pastikan state `page`/`limit` tetap tersimpan di URL (CLAUDE.md §5: "pertahankan
    filter/pagination di URL bila memungkinkan").
- **File terdampak**: `frontend/src/components/data-table/Pagination.jsx` (komponen shared —
  cek semua halaman list yang memakainya agar tidak ada regresi tak disengaja)
- **Prioritas**: Sedang

## 9. Dashboard HR I — Pie chart komposisi departemen

**Status: ✅ Selesai** — chart menjadi donut berukuran lebih besar, warna dibangkitkan unik per
departemen, persentase tampil pada slice serta legenda, dan hover/focus pada warna menampilkan
tooltip nama departemen, jumlah karyawan, dan persentasenya.

- **Current**: `PieChart.jsx`/`CompositionChart.jsx` merender pie chart penuh (bukan donat),
  beberapa departemen memakai warna yang sama (warna ter-reuse), ukuran chart relatif kecil,
  tanpa label persentase per slice.
- **Adjustment**:
  - Ubah menjadi donut chart (hollow center).
  - Perbesar ukuran chart.
  - Pastikan setiap departemen mendapat warna berbeda — buat palet warna sejumlah departemen
    aktif, jangan mengulang warna antar kategori.
  - Tampilkan label persentase pada tiap slice.
- **File terdampak**: `frontend/src/components/charts/PieChart.jsx`,
  `frontend/src/components/charts/CompositionChart.jsx`
- **Prioritas**: Sedang
- **Catatan**: gunakan skill `dataviz` untuk pemilihan palet warna yang konsisten & accessible.

## 10. Dashboard HR II — border struktur organisasi

**Status: ✅ Selesai** — node organisasi sekarang memakai background dan border putih dengan
shadow tipis untuk pemisahan visual.

- **Current**: `OrganizationChart.jsx` — card node karyawan memakai background/border abu-abu,
  kontrasnya rendah terhadap card luar yang juga abu-abu muda.
- **Adjustment**: ubah background/border card node dari abu-abu menjadi putih.
- **File terdampak**: `frontend/src/modules/dashboard/components/OrganizationChart.jsx`
- **Prioritas**: Rendah

## 11. Company Feed — upload file/gambar & bug formatting

**Status: ✅ Selesai**

- ✅ Bold/Italic: toolbar bawaan `react-simple-wysiwyg` dipertahankan, font Inter normal dan
  italic di-self-host, dan style `<strong>/<b>/<em>/<i>` dipastikan tampil pada editor serta feed.
- ✅ Attachment PDF/PNG/JPG/JPEG: upload multi-file tersedia saat membuat/mengedit feed, file
  tersimpan melalui storage driver aktif, metadata tersimpan di `company_feed_attachments`, dan
  preview/download tampil pada kartu feed.
- ✅ Keamanan/lifecycle: maksimal 5 file per feed dan 5 MB per file, extension+MIME+signature
  divalidasi backend, attachment dapat dihapus saat edit, dan object storage dibersihkan ketika
  attachment atau feed dihapus.

- **Current**: form "Feed Baru" (modul `uat`) hanya punya toolbar teks (Bold, Italic, Underline,
  Strikethrough, list, link, superscript, code, styles) tanpa opsi lampiran file. Tombol Bold dan
  Italic dilaporkan tidak berfungsi saat mengetik di editor.
- **Adjustment**:
  - Tambahkan fitur upload file PDF dan gambar (png/jpg/jpeg) agar poster/dokumen terkait bisa
    tampil di Company Feed.
  - Perbaiki bug tombol Bold/Italic yang tidak berfungsi saat mengetik (`react-simple-wysiwyg`
    sesuai baseline stack CLAUDE.md §3).
- **File terdampak**: `frontend/src/modules/uat/pages/CompanyFeedPage.jsx`,
  `frontend/src/modules/uat/components/FeedCard.jsx`,
  `frontend/src/modules/uat/components/CompanyFeedInfiniteList.jsx`; kemungkinan perlu endpoint
  attachment baru di backend (lihat catatan klarifikasi di atas) dengan validasi ukuran/MIME
  type sesuai standar File Security CLAUDE.md §8 (maks 5 MB, validasi signature, via `filestore`
  interface MinIO/Nextcloud).
- **Prioritas**: Tinggi — ada bug fungsional aktif (Bold/Italic) plus fitur baru yang berdampak
  ke kontrak API dan storage.

## 12. Perubahan Font — Eloquia → Inter

**Status: ✅ Selesai** — Inter variable normal dan italic berformat WOFF2 sudah di-self-host pada
`frontend/public/fonts/` dan menjadi font utama untuk seluruh teks dashboard, termasuk heading,
tombol, input, select, textarea, tabel, dan teks SVG/chart.

- **Current**: `frontend/src/styles/index.css` menetapkan `--font-sans`/`--font-display` dengan
  "Eloquia Text"/"Eloquia Display" sebagai font utama (via `@font-face`, file di
  `frontend/public/fonts/Eloquia*.otf`); Inter hanya fallback.
- **Adjustment**: jadikan Inter font utama di seluruh HRIS.
- **Action items**:
  - Sediakan file font Inter (self-host `@font-face`, taruh di `frontend/public/fonts/`) —
    hindari memuat dari CDN eksternal tanpa keputusan produk.
  - Update `--font-sans` dan `--font-display` di `frontend/src/styles/index.css` agar Inter jadi
    prioritas utama.
  - Putuskan apakah Eloquia dihapus total atau dipertahankan untuk elemen tertentu (mis.
    wordmark/logo) — feedback tidak menyebutkan pengecualian, default: ganti seluruhnya ke
    Inter kecuali ada instruksi lain.
- **File terdampak**: `frontend/src/styles/index.css`, `frontend/public/fonts/`
- **Prioritas**: Sedang — perubahan global yang berdampak visual ke seluruh aplikasi; sarankan
  dikerjakan setelah item polish lain selesai agar tidak bentrok saat review UI per halaman.
