- HR bisa menambahkan jenis dokumen yang perlu diupload untuk masing - masing karyawan. Jenis dokumen ini seragam untuk semua karyawan
contoh: HR membuat jenis dokumen Foto KTP kemudian setiap karyawan perlu upload dokumen dengan jenis Foto KTP  
- absensi terhitung hadir ketika clock-in dan clock-out
- Fitur post company feed yang berisi WYSIWYG dan akan terpublish di timeline yang dapat dilihat semua karyawan. hanya HR yang memiliki akses
- beranda kalau bisa menampilkan company feed, saldo cuti berdasarkan jenis cuti, pengajuan yang perlu disetujui, dan pengajuan ketidakhadiran pribadi
- "Ketidakhadiran" perlu menambahkan pembeda antara izin dan cuti, dimana izin tidak memiliki saldo seperti cuti. Izin memiliki batas hari maksimal yang dapat diatur di "Master Jenis Izin". Apabila cuti mengurangi saldo cuti, maka izin memiliki maksimal hari izin. Sebagai contoh izin berduka maksimal 3 hari yang otomatis terhitung dari hari pertama berduka.
- untuk komposisi departemen menggunakan pie chart saja, berisi headcount tiap departemen
- untuk rasio gender menggunakan icon gender saja
contoh: metrik rasio gender menggunakan icon
- fitur untuk kalender. fitur ini menampilkan kalender dan bisa menandai hari libur. untuk update hari libur menggunakan bulk insert / bulk update tanggal
contoh: tiap tahun update kalender libur
- dibagian profil saya ditambahkan usia sesuai dengan tanggal lahirnya
- untuk fitur "Ajukan Ketidakhadiran" dan "Ajukan Lembur" dijadikan sub menu (menu awal "Pengajuan" isi submenu nya ada "Ketidakhadiran" dan "Lembur")
contoh: ketika "Pengajuan" di klik akan muncul dropdown "Ketidakhadiran" dan "Lembur"
- Sidebar menggunakan hover dan menampilkan detail submenu pada tiap modul 
contoh: hanya menampilkan parent dropdown dan diklik untuk child dropdown, sekaligus bisa collapse dan ketika hover menampilkan child dropdown
- menambahkan profile picture
bug: imagenya belum muncul
- ukuran button dikecilkan menjadi size medium
- untuk role top management hilangkan fitur: employee database, dashboard hr, live feed absensi, laporan kehadiran, akses, audit log.
- status terlambat / pulang cepat akan menampilkan background warna danger saat clock in dan clock out, saat ini menggunakan success
- button Tambah Karyawan pada Employee Database inline dengan export
- form pendidikan ada tahun masuk dan tahun lulus. tahun lulus bisa opsional menunjukkan sedang pendidikan
- semua waktu menggunakan format 24 jam timezone jakarta jika ditampilkan ke user. defeault sistem menggunakan UTC
- gunakan format delimiter uang menggunakan rupiah
- fitur untuk bulk upload database karyawan ke sistem
- user dashboard ketika bulk upload karyawan akan otomatis terbuat user dengan email menggunakan format {nama pertama}@janjikupadamu.id jika terdapat duplikat pakai {nama pertama + nama kedua}@janjikupadamu.id
- image yang terupload belum tampil
- fitur koreksi absensi clock in dan clock out, ketika user ingin clock in / out tapi dengan memilih waktu clock in / clock out, maka harus menggunakan approval sesuai hirarki (karyawan > atasan > HR)

