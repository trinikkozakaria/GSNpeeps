-- +goose Up
-- Tabel ke-26 sesuai keputusan D-013. Koordinat resmi di-seed kemudian; master boleh kosong.
CREATE TABLE office_locations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kode VARCHAR(50) NOT NULL UNIQUE,
    nama VARCHAR(150) NOT NULL,
    alamat VARCHAR(255),
    latitude DECIMAL(10, 7) NOT NULL,
    longitude DECIMAL(10, 7) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT office_locations_latitude_check CHECK (latitude BETWEEN -90 AND 90),
    CONSTRAINT office_locations_longitude_check CHECK (longitude BETWEEN -180 AND 180)
);

CREATE INDEX idx_office_locations_is_active ON office_locations (is_active);

-- Kolom office_location_id, distance_meters, dan enum status mengikuti schema Attendance
-- pada OpenAPI; lihat keputusan D-022.
CREATE TABLE attendances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    tanggal DATE NOT NULL,
    tipe VARCHAR(10) NOT NULL,
    mode_kerja VARCHAR(3) NOT NULL,
    waktu_network TIMESTAMP NOT NULL,
    waktu_local TIMESTAMP NOT NULL,
    gps_lat DECIMAL(10, 7) NOT NULL,
    gps_long DECIMAL(10, 7) NOT NULL,
    office_location_id UUID REFERENCES office_locations(id) ON DELETE RESTRICT,
    distance_meters DECIMAL(10, 2),
    alamat VARCHAR(255),
    foto_url VARCHAR(255),
    status VARCHAR(15) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT attendances_tipe_check CHECK (tipe IN ('check_in', 'check_out')),
    CONSTRAINT attendances_mode_kerja_check CHECK (mode_kerja IN ('WFO', 'WFH', 'WFA')),
    CONSTRAINT attendances_status_check
        CHECK (status IN ('tepat_waktu', 'terlambat', 'pulang_cepat', 'valid')),
    CONSTRAINT attendances_gps_lat_check CHECK (gps_lat BETWEEN -90 AND 90),
    CONSTRAINT attendances_gps_long_check CHECK (gps_long BETWEEN -180 AND 180),
    CONSTRAINT attendances_distance_check CHECK (distance_meters IS NULL OR distance_meters >= 0),
    -- WFO wajib membawa lokasi kantor tepercaya; WFH dan WFA tidak menyimpannya.
    CONSTRAINT attendances_office_location_check CHECK (
        (mode_kerja = 'WFO' AND office_location_id IS NOT NULL)
        OR (mode_kerja <> 'WFO' AND office_location_id IS NULL)
    )
);

CREATE INDEX idx_attendances_user_tanggal ON attendances (user_id, tanggal);
-- Menegakkan satu check-in dan satu check-out per user per tanggal.
CREATE UNIQUE INDEX uq_attendances_user_tanggal_tipe ON attendances (user_id, tanggal, tipe);
-- Dipakai job retensi foto tiga bulan.
CREATE INDEX idx_attendances_foto_retention ON attendances (tanggal) WHERE foto_url IS NOT NULL;

-- Kolom kode, memerlukan_dokumen, dan is_active mengikuti schema LeaveType (D-023).
CREATE TABLE leave_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kode VARCHAR(50) NOT NULL UNIQUE,
    nama VARCHAR(150) NOT NULL UNIQUE,
    kuota_tahunan INT NOT NULL,
    memerlukan_dokumen BOOLEAN NOT NULL DEFAULT TRUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT leave_types_kuota_check CHECK (kuota_tahunan >= 0)
);

CREATE INDEX idx_leave_types_is_active ON leave_types (is_active);

CREATE TABLE leave_balances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    tahun SMALLINT NOT NULL,
    saldo_awal SMALLINT NOT NULL,
    saldo_terpakai SMALLINT NOT NULL DEFAULT 0,
    saldo_sisa SMALLINT GENERATED ALWAYS AS (saldo_awal - saldo_terpakai) STORED,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, tahun),
    CONSTRAINT leave_balances_saldo_awal_check CHECK (saldo_awal >= 0),
    -- Menegakkan saldo tidak pernah negatif pada level database.
    CONSTRAINT leave_balances_saldo_terpakai_check
        CHECK (saldo_terpakai >= 0 AND saldo_terpakai <= saldo_awal),
    CONSTRAINT leave_balances_tahun_check CHECK (tahun BETWEEN 1900 AND 2200)
);

CREATE TABLE leave_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    leave_type_id UUID NOT NULL REFERENCES leave_types(id) ON DELETE RESTRICT,
    tanggal_mulai DATE NOT NULL,
    tanggal_selesai DATE NOT NULL,
    jumlah_hari SMALLINT NOT NULL,
    alasan TEXT NOT NULL,
    -- Nullable karena kewajiban dokumen ditentukan master jenis izin (D-024).
    dokumen_url VARCHAR(255),
    lokasi_tujuan VARCHAR(150),
    keperluan_tugas VARCHAR(255),
    status VARCHAR(30) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT leave_requests_date_check CHECK (tanggal_selesai >= tanggal_mulai),
    CONSTRAINT leave_requests_jumlah_hari_check CHECK (jumlah_hari >= 1),
    CONSTRAINT leave_requests_status_check CHECK (status IN (
        'menunggu_atasan', 'menunggu_hr', 'menunggu_top_management',
        'disetujui', 'ditolak', 'dibatalkan'
    ))
);

CREATE INDEX idx_leave_requests_user_id ON leave_requests (user_id);
CREATE INDEX idx_leave_requests_status ON leave_requests (status);
-- Dipakai worker auto-escalation untuk memindai tahap atasan yang melewati SLA.
CREATE INDEX idx_leave_requests_escalation ON leave_requests (created_at)
    WHERE status = 'menunggu_atasan';

CREATE TABLE leave_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    leave_request_id UUID NOT NULL REFERENCES leave_requests(id) ON DELETE RESTRICT,
    -- NULL ketika keputusan dipicu sistem melalui auto_escalate.
    approver_id UUID REFERENCES users(id) ON DELETE RESTRICT,
    tahap VARCHAR(20) NOT NULL,
    keputusan VARCHAR(20) NOT NULL,
    catatan TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT leave_approvals_tahap_check CHECK (tahap IN ('atasan', 'hr', 'top_management')),
    CONSTRAINT leave_approvals_keputusan_check
        CHECK (keputusan IN ('approve', 'reject', 'auto_escalate', 'delegate')),
    CONSTRAINT leave_approvals_approver_check CHECK (
        (keputusan = 'auto_escalate' AND approver_id IS NULL)
        OR (keputusan <> 'auto_escalate' AND approver_id IS NOT NULL)
    )
);

CREATE INDEX idx_leave_approvals_request_id ON leave_approvals (leave_request_id, created_at);

CREATE TABLE overtime_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    tanggal DATE NOT NULL,
    jam_mulai TIME NOT NULL,
    jam_selesai TIME NOT NULL,
    -- Durasi dihitung database agar tidak dapat menyimpang dari jam yang tersimpan.
    durasi_jam DECIMAL(4, 2) GENERATED ALWAYS AS (
        EXTRACT(EPOCH FROM (jam_selesai - jam_mulai)) / 3600
    ) STORED,
    alasan TEXT NOT NULL,
    dokumen_url VARCHAR(255),
    status VARCHAR(30) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT overtime_requests_time_check CHECK (jam_selesai > jam_mulai),
    CONSTRAINT overtime_requests_status_check CHECK (status IN (
        'menunggu_atasan', 'menunggu_hr', 'menunggu_top_management',
        'disetujui', 'ditolak', 'dibatalkan'
    ))
);

CREATE INDEX idx_overtime_requests_user_id ON overtime_requests (user_id);
CREATE INDEX idx_overtime_requests_status ON overtime_requests (status);
CREATE INDEX idx_overtime_requests_tanggal ON overtime_requests (tanggal);
CREATE INDEX idx_overtime_requests_escalation ON overtime_requests (created_at)
    WHERE status = 'menunggu_atasan';

CREATE TABLE overtime_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    overtime_request_id UUID NOT NULL REFERENCES overtime_requests(id) ON DELETE RESTRICT,
    approver_id UUID REFERENCES users(id) ON DELETE RESTRICT,
    tahap VARCHAR(20) NOT NULL,
    keputusan VARCHAR(20) NOT NULL,
    catatan TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT overtime_approvals_tahap_check CHECK (tahap IN ('atasan', 'hr', 'top_management')),
    CONSTRAINT overtime_approvals_keputusan_check
        CHECK (keputusan IN ('approve', 'reject', 'auto_escalate', 'delegate')),
    CONSTRAINT overtime_approvals_approver_check CHECK (
        (keputusan = 'auto_escalate' AND approver_id IS NULL)
        OR (keputusan <> 'auto_escalate' AND approver_id IS NOT NULL)
    )
);

CREATE INDEX idx_overtime_approvals_request_id
    ON overtime_approvals (overtime_request_id, created_at);

-- +goose Down
DROP TABLE IF EXISTS overtime_approvals;
DROP TABLE IF EXISTS overtime_requests;
DROP TABLE IF EXISTS leave_approvals;
DROP TABLE IF EXISTS leave_requests;
DROP TABLE IF EXISTS leave_balances;
DROP TABLE IF EXISTS leave_types;
DROP TABLE IF EXISTS attendances;
DROP TABLE IF EXISTS office_locations;
