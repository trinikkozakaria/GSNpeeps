-- +goose Up
-- Hasil review query plan pada 20.000 baris audit: filter `user_id` yang dipadukan dengan
-- urutan `created_at DESC` memakai index waktu lalu membuang ratusan baris melalui filter.
-- Biayanya tumbuh linear terhadap ukuran tabel append-only ini.
--
-- Index komposit membuat filter aktor dan urutan terbaru dilayani satu index, sekaligus tetap
-- melayani `idx_audit_logs_user_id` sehingga index lama tidak lagi diperlukan.
CREATE INDEX idx_audit_logs_user_created_at ON audit_logs (user_id, created_at DESC);
DROP INDEX IF EXISTS idx_audit_logs_user_id;

-- +goose Down
CREATE INDEX idx_audit_logs_user_id ON audit_logs (user_id);
DROP INDEX IF EXISTS idx_audit_logs_user_created_at;
