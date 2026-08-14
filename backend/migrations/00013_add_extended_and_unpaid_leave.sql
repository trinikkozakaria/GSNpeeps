-- +goose Up
-- SOP juga mengizinkan keperluan pribadi lain sebagai unpaid leave dan
-- menyebut benefit Extended Maternity Leave bulan ke-4 sampai ke-6.
INSERT INTO leave_types (
    kode, nama, kuota_tahunan, kategori, maksimal_hari,
    memerlukan_dokumen, is_active
)
VALUES
    ('UNPAID-LEAVE', 'Izin Pribadi Lainnya (Unpaid Leave)', 0, 'izin', 365, FALSE, TRUE),
    ('EXT-MATERNITY', 'Extended Maternity Leave', 0, 'izin', 90, TRUE, TRUE)
ON CONFLICT DO NOTHING;

UPDATE leave_types AS lt
SET nama = source.nama,
    kuota_tahunan = 0,
    kategori = 'izin',
    maksimal_hari = source.maksimal_hari,
    memerlukan_dokumen = source.memerlukan_dokumen,
    is_active = TRUE,
    updated_at = NOW()
FROM (VALUES
    ('UNPAID-LEAVE', 'Izin Pribadi Lainnya (Unpaid Leave)', 365::smallint, FALSE),
    ('EXT-MATERNITY', 'Extended Maternity Leave', 90::smallint, TRUE)
) AS source(kode, nama, maksimal_hari, memerlukan_dokumen)
WHERE lt.kode = source.kode;

-- +goose Down
-- Tidak menghapus master yang mungkin sudah direferensikan pengajuan.
