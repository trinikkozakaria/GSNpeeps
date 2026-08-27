-- +goose Up
-- Company Feed baru mengirim notifikasi in-app ke seluruh pengguna aktif (selain penulis).
-- Idempotensi tetap ditegakkan UNIQUE (recipient_user_id, event_key) dengan event_key
-- `company_feed:<feed_id>`. CHECK `tipe` dan `referensi_tipe` diperluas agar baris ini valid
-- dan dapat men-deep-link ke halaman Beranda tempat feed tampil.
ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_tipe_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_tipe_check CHECK (tipe IN (
    'ketidakhadiran_baru',
    'lembur_baru',
    'keputusan_approve',
    'keputusan_reject',
    'auto_escalate',
    'delegasi',
    'kontrak_akan_habis',
    'company_feed_baru'
));

ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_referensi_tipe_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_referensi_tipe_check CHECK (
    referensi_tipe IS NULL OR referensi_tipe IN ('ketidakhadiran', 'lembur', 'karyawan', 'company_feed')
);

-- +goose Down
DELETE FROM notifications WHERE tipe = 'company_feed_baru';

ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_referensi_tipe_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_referensi_tipe_check CHECK (
    referensi_tipe IS NULL OR referensi_tipe IN ('ketidakhadiran', 'lembur', 'karyawan')
);

ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_tipe_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_tipe_check CHECK (tipe IN (
    'ketidakhadiran_baru',
    'lembur_baru',
    'keputusan_approve',
    'keputusan_reject',
    'auto_escalate',
    'delegasi',
    'kontrak_akan_habis'
));
