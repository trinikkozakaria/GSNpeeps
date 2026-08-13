-- +goose Up
-- Notifikasi in-app. Idempotensi ditegakkan database melalui UNIQUE (recipient_user_id,
-- event_key); producer yang mengulang event yang sama tidak menambah baris.
-- `judul` dan `read_at` mengikuti schema response Notification (D-030).
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipient_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    tipe VARCHAR(40) NOT NULL,
    judul VARCHAR(150) NOT NULL,
    pesan VARCHAR(500) NOT NULL,
    referensi_id UUID,
    referensi_tipe VARCHAR(30),
    event_key VARCHAR(200) NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    read_at TIMESTAMP,
    dismissed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT notifications_tipe_check CHECK (tipe IN (
        'ketidakhadiran_baru',
        'lembur_baru',
        'keputusan_approve',
        'keputusan_reject',
        'auto_escalate',
        'delegasi',
        'kontrak_akan_habis'
    )),
    CONSTRAINT notifications_referensi_tipe_check CHECK (
        referensi_tipe IS NULL OR referensi_tipe IN ('ketidakhadiran', 'lembur', 'karyawan')
    ),
    -- `read_at` tidak boleh menyimpang dari `is_read` sehingga response selalu konsisten.
    CONSTRAINT notifications_read_at_check CHECK (
        (is_read AND read_at IS NOT NULL) OR (NOT is_read AND read_at IS NULL)
    ),
    CONSTRAINT notifications_event_key_not_blank CHECK (LENGTH(TRIM(event_key)) > 0),
    CONSTRAINT notifications_recipient_event_key_unique UNIQUE (recipient_user_id, event_key)
);

-- Inbox dibaca per penerima, terbaru lebih dahulu, dan tidak pernah memuat yang di-dismiss.
CREATE INDEX idx_notifications_recipient_inbox
    ON notifications (recipient_user_id, created_at DESC)
    WHERE dismissed_at IS NULL;

-- Badge unread adalah COUNT murni; partial index membuatnya index-only.
CREATE INDEX idx_notifications_recipient_unread
    ON notifications (recipient_user_id)
    WHERE dismissed_at IS NULL AND is_read = FALSE;

CREATE INDEX idx_notifications_event_key ON notifications (event_key);

-- Audit log bersifat append-only. Trigger menolak UPDATE dan DELETE apa pun rolenya,
-- termasuk superuser yang melewati pemeriksaan privilege.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION reject_audit_log_mutation()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'audit_logs is append-only'
        USING ERRCODE = '42501';
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_audit_logs_append_only
BEFORE UPDATE OR DELETE ON audit_logs
FOR EACH ROW EXECUTE FUNCTION reject_audit_log_mutation();

-- TRUNCATE tidak memicu trigger baris, sehingga jalur itu ditutup terpisah.
CREATE TRIGGER trg_audit_logs_no_truncate
BEFORE TRUNCATE ON audit_logs
FOR EACH STATEMENT EXECUTE FUNCTION reject_audit_log_mutation();

-- Lapis kedua: role aplikasi runtime kehilangan privilege mutation. REVOKE tidak berlaku
-- untuk superuser, karena itu trigger di atas tetap menjadi penegak utama.
-- +goose StatementBegin
DO $$
BEGIN
    EXECUTE FORMAT('REVOKE UPDATE, DELETE, TRUNCATE ON audit_logs FROM %I', CURRENT_USER);
EXCEPTION
    WHEN insufficient_privilege THEN
        RAISE NOTICE 'revoke audit_logs mutation privileges skipped for %', CURRENT_USER;
END;
$$;
-- +goose StatementEnd

-- Matriks permission dipetakan ulang ke kosakata modul/aksi kontrak (D-031). Nilai lama
-- seperti `view_own` tidak dapat direpresentasikan schema Permission pada OpenAPI.
DELETE FROM permissions;

WITH capability(modul, aksi) AS (
    VALUES
        ('karyawan', 'create'), ('karyawan', 'read'), ('karyawan', 'update'),
        ('karyawan', 'delete'), ('karyawan', 'export'),
        ('dashboard', 'read'),
        ('absensi', 'create'), ('absensi', 'read'),
        ('laporan_kehadiran', 'read'), ('laporan_kehadiran', 'export'),
        ('ketidakhadiran', 'create'), ('ketidakhadiran', 'read'),
        ('ketidakhadiran', 'update'), ('ketidakhadiran', 'approve'),
        ('lembur', 'create'), ('lembur', 'read'), ('lembur', 'approve'),
        ('notifikasi', 'read'), ('notifikasi', 'update'), ('notifikasi', 'delete'),
        ('akses', 'read'), ('akses', 'update'),
        ('audit', 'read')
),
granted(role_name, modul, aksi) AS (
    VALUES
        ('karyawan', 'absensi', 'create'),
        ('karyawan', 'absensi', 'read'),
        ('karyawan', 'ketidakhadiran', 'create'),
        ('karyawan', 'ketidakhadiran', 'read'),
        ('karyawan', 'lembur', 'create'),
        ('karyawan', 'lembur', 'read'),
        ('karyawan', 'notifikasi', 'read'),
        ('karyawan', 'notifikasi', 'update'),
        ('karyawan', 'notifikasi', 'delete'),
        ('atasan', 'absensi', 'create'),
        ('atasan', 'absensi', 'read'),
        ('atasan', 'ketidakhadiran', 'create'),
        ('atasan', 'ketidakhadiran', 'read'),
        ('atasan', 'ketidakhadiran', 'approve'),
        ('atasan', 'lembur', 'create'),
        ('atasan', 'lembur', 'read'),
        ('atasan', 'lembur', 'approve'),
        ('atasan', 'notifikasi', 'read'),
        ('atasan', 'notifikasi', 'update'),
        ('atasan', 'notifikasi', 'delete'),
        ('hr', 'karyawan', 'create'),
        ('hr', 'karyawan', 'read'),
        ('hr', 'karyawan', 'update'),
        ('hr', 'karyawan', 'delete'),
        ('hr', 'karyawan', 'export'),
        ('hr', 'dashboard', 'read'),
        ('hr', 'absensi', 'create'),
        ('hr', 'absensi', 'read'),
        ('hr', 'laporan_kehadiran', 'read'),
        ('hr', 'laporan_kehadiran', 'export'),
        ('hr', 'ketidakhadiran', 'create'),
        ('hr', 'ketidakhadiran', 'read'),
        ('hr', 'ketidakhadiran', 'update'),
        ('hr', 'ketidakhadiran', 'approve'),
        ('hr', 'lembur', 'create'),
        ('hr', 'lembur', 'read'),
        ('hr', 'lembur', 'approve'),
        ('hr', 'notifikasi', 'read'),
        ('hr', 'notifikasi', 'update'),
        ('hr', 'notifikasi', 'delete'),
        ('hr', 'akses', 'read'),
        ('hr', 'akses', 'update'),
        ('hr', 'audit', 'read'),
        -- Top Management read-only; satu-satunya mutation adalah keputusan pengajuan HR.
        ('top_management', 'karyawan', 'read'),
        ('top_management', 'dashboard', 'read'),
        ('top_management', 'absensi', 'read'),
        ('top_management', 'laporan_kehadiran', 'read'),
        ('top_management', 'ketidakhadiran', 'read'),
        ('top_management', 'ketidakhadiran', 'approve'),
        ('top_management', 'lembur', 'read'),
        ('top_management', 'lembur', 'approve'),
        ('top_management', 'notifikasi', 'read'),
        ('top_management', 'notifikasi', 'update'),
        ('top_management', 'notifikasi', 'delete'),
        ('top_management', 'akses', 'read'),
        ('top_management', 'audit', 'read')
)
INSERT INTO permissions (role_id, modul, aksi, diizinkan)
SELECT
    roles.id,
    capability.modul,
    capability.aksi,
    EXISTS (
        SELECT 1 FROM granted
        WHERE granted.role_name = roles.nama
          AND granted.modul = capability.modul
          AND granted.aksi = capability.aksi
    )
FROM roles
CROSS JOIN capability
ON CONFLICT (role_id, modul, aksi) DO UPDATE SET diizinkan = EXCLUDED.diizinkan;

-- +goose Down
DELETE FROM permissions;

WITH permission_seed(role_name, modul, aksi) AS (
    VALUES
        ('karyawan', 'profil', 'view_own'),
        ('karyawan', 'absensi', 'create_own'),
        ('karyawan', 'ketidakhadiran', 'create_own'),
        ('karyawan', 'ketidakhadiran', 'view_own'),
        ('karyawan', 'lembur', 'create_own'),
        ('karyawan', 'lembur', 'view_own'),
        ('karyawan', 'notifikasi', 'manage_own'),
        ('atasan', 'profil', 'view_own'),
        ('atasan', 'absensi', 'create_own'),
        ('atasan', 'ketidakhadiran', 'manage_own'),
        ('atasan', 'ketidakhadiran', 'approve_direct'),
        ('atasan', 'lembur', 'manage_own'),
        ('atasan', 'lembur', 'approve_direct'),
        ('atasan', 'notifikasi', 'manage_own'),
        ('hr', 'profil', 'view_own'),
        ('hr', 'karyawan', 'manage'),
        ('hr', 'dashboard', 'view'),
        ('hr', 'absensi', 'monitor'),
        ('hr', 'ketidakhadiran', 'approve'),
        ('hr', 'lembur', 'approve'),
        ('hr', 'permission', 'manage'),
        ('hr', 'audit', 'view'),
        ('hr', 'notifikasi', 'manage_own'),
        ('top_management', 'karyawan', 'view'),
        ('top_management', 'dashboard', 'view'),
        ('top_management', 'absensi', 'monitor'),
        ('top_management', 'ketidakhadiran', 'approve_hr'),
        ('top_management', 'lembur', 'approve_hr'),
        ('top_management', 'permission', 'view'),
        ('top_management', 'audit', 'view'),
        ('top_management', 'notifikasi', 'manage_own')
)
INSERT INTO permissions (role_id, modul, aksi, diizinkan)
SELECT roles.id, permission_seed.modul, permission_seed.aksi, TRUE
FROM permission_seed
JOIN roles ON roles.nama = permission_seed.role_name
ON CONFLICT (role_id, modul, aksi)
DO UPDATE SET diizinkan = EXCLUDED.diizinkan;

-- +goose StatementBegin
DO $$
BEGIN
    EXECUTE FORMAT('GRANT UPDATE, DELETE ON audit_logs TO %I', CURRENT_USER);
EXCEPTION
    WHEN insufficient_privilege THEN
        RAISE NOTICE 'grant audit_logs mutation privileges skipped for %', CURRENT_USER;
END;
$$;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS trg_audit_logs_no_truncate ON audit_logs;
DROP TRIGGER IF EXISTS trg_audit_logs_append_only ON audit_logs;
DROP FUNCTION IF EXISTS reject_audit_log_mutation();
DROP TABLE IF EXISTS notifications;
