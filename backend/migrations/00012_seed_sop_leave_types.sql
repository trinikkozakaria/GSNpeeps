-- +goose Up
-- Master cuti dan izin berdasarkan SOP Cuti, Kehadiran, Absensi, dan Izin
-- bagian 1.4-1.9. Upsert membuat migrasi aman untuk database yang sudah pernah
-- diisi manual tanpa menghasilkan kode ganda.
INSERT INTO leave_types (
    kode, nama, kuota_tahunan, kategori, maksimal_hari,
    memerlukan_dokumen, is_active
)
VALUES
    ('CUTI-TAHUNAN', 'Cuti Tahunan', 12, 'cuti', NULL, FALSE, TRUE),
    ('IZIN-SAKIT', 'Izin Sakit', 0, 'izin', 365, TRUE, TRUE),
    ('IZIN-NIKAH', 'Pernikahan Karyawan', 0, 'izin', 3, TRUE, TRUE),
    ('IZIN-NIKAH-ANAK', 'Menikahkan Anak', 0, 'izin', 2, TRUE, TRUE),
    ('IZIN-KHITAN-BAPTIS', 'Khitanan/Pembaptisan Anak', 0, 'izin', 2, TRUE, TRUE),
    ('IZIN-ISTRI-MELAHIRKAN', 'Istri Melahirkan/Keguguran', 0, 'izin', 5, TRUE, TRUE),
    ('IZIN-KEGUGURAN', 'Izin Keguguran', 0, 'izin', 45, TRUE, TRUE),
    ('IZIN-HAID', 'Izin Haid', 0, 'izin', 1, FALSE, TRUE),
    ('IZIN-DUKA-INTI', 'Keluarga Inti Meninggal Dunia', 0, 'izin', 3, TRUE, TRUE),
    ('IZIN-DUKA-SERUMAH', 'Keluarga Serumah (Bukan Inti) Meninggal Dunia', 0, 'izin', 1, TRUE, TRUE),
    ('IZIN-HAJI', 'Ibadah Haji', 0, 'izin', 30, TRUE, TRUE),
    ('IZIN-UMROH', 'Ibadah Umroh', 0, 'izin', 14, TRUE, TRUE),
    ('IZIN-RAWAT-KELUARGA', 'Merawat Keluarga Inti yang Dirawat Inap', 0, 'izin', 1, TRUE, TRUE),
    ('PATERNITY-LEAVE', 'Paternity Leave', 0, 'izin', 5, TRUE, TRUE),
    ('MATERNITY-LEAVE', 'Maternity Leave', 0, 'izin', 90, TRUE, TRUE)
-- Tanpa target konflik agar nama yang sebelumnya sudah dibuat HR dengan kode
-- berbeda juga tidak membuat migrasi berhenti.
ON CONFLICT DO NOTHING;

-- Sinkronkan nilai resmi untuk baris yang menggunakan kode SOP. Baris manual
-- dengan nama sama namun kode berbeda tetap dipertahankan sebagai data milik HR.
UPDATE leave_types AS lt
SET nama = source.nama,
    kuota_tahunan = source.kuota_tahunan,
    kategori = source.kategori,
    maksimal_hari = source.maksimal_hari,
    memerlukan_dokumen = source.memerlukan_dokumen,
    is_active = TRUE,
    updated_at = NOW()
FROM (VALUES
    ('CUTI-TAHUNAN', 'Cuti Tahunan', 12, 'cuti', NULL::smallint, FALSE),
    ('IZIN-SAKIT', 'Izin Sakit', 0, 'izin', 365::smallint, TRUE),
    ('IZIN-NIKAH', 'Pernikahan Karyawan', 0, 'izin', 3::smallint, TRUE),
    ('IZIN-NIKAH-ANAK', 'Menikahkan Anak', 0, 'izin', 2::smallint, TRUE),
    ('IZIN-KHITAN-BAPTIS', 'Khitanan/Pembaptisan Anak', 0, 'izin', 2::smallint, TRUE),
    ('IZIN-ISTRI-MELAHIRKAN', 'Istri Melahirkan/Keguguran', 0, 'izin', 5::smallint, TRUE),
    ('IZIN-KEGUGURAN', 'Izin Keguguran', 0, 'izin', 45::smallint, TRUE),
    ('IZIN-HAID', 'Izin Haid', 0, 'izin', 1::smallint, FALSE),
    ('IZIN-DUKA-INTI', 'Keluarga Inti Meninggal Dunia', 0, 'izin', 3::smallint, TRUE),
    ('IZIN-DUKA-SERUMAH', 'Keluarga Serumah (Bukan Inti) Meninggal Dunia', 0, 'izin', 1::smallint, TRUE),
    ('IZIN-HAJI', 'Ibadah Haji', 0, 'izin', 30::smallint, TRUE),
    ('IZIN-UMROH', 'Ibadah Umroh', 0, 'izin', 14::smallint, TRUE),
    ('IZIN-RAWAT-KELUARGA', 'Merawat Keluarga Inti yang Dirawat Inap', 0, 'izin', 1::smallint, TRUE),
    ('PATERNITY-LEAVE', 'Paternity Leave', 0, 'izin', 5::smallint, TRUE),
    ('MATERNITY-LEAVE', 'Maternity Leave', 0, 'izin', 90::smallint, TRUE)
) AS source(kode, nama, kuota_tahunan, kategori, maksimal_hari, memerlukan_dokumen)
WHERE lt.kode = source.kode;

-- Berikan saldo cuti tahunan tahun berjalan kepada seluruh akun karyawan aktif.
INSERT INTO leave_balances (user_id, tahun, saldo_awal, leave_type_id)
SELECT u.id, EXTRACT(YEAR FROM NOW())::int, 12, lt.id
FROM users u
JOIN employees e ON e.id = u.employee_id
JOIN leave_types lt ON lt.kode = 'CUTI-TAHUNAN'
WHERE e.deleted_at IS NULL AND e.status = 'aktif'
ON CONFLICT (user_id, tahun, leave_type_id) DO NOTHING;

-- +goose Down
-- Sengaja tidak menghapus master maupun saldo agar rollback tidak merusak
-- pengajuan yang sudah mereferensikan jenis izin SOP ini.
