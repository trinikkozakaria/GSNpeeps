-- +goose Up
-- Jenis dokumen berlaku seragam untuk seluruh karyawan.
CREATE TABLE document_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kode VARCHAR(50) NOT NULL UNIQUE,
    nama VARCHAR(150) NOT NULL UNIQUE,
    wajib BOOLEAN NOT NULL DEFAULT TRUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE employee_documents ADD COLUMN document_type_id UUID REFERENCES document_types(id) ON DELETE RESTRICT;
CREATE INDEX idx_employee_documents_type ON employee_documents (document_type_id);

-- Feed perusahaan hanya ditulis HR dan dibaca seluruh user terautentikasi.
CREATE TABLE company_feeds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    judul VARCHAR(200) NOT NULL,
    konten_html TEXT NOT NULL,
    published_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT company_feeds_content_check CHECK (length(trim(konten_html)) > 0)
);
CREATE INDEX idx_company_feeds_published ON company_feeds (published_at DESC);

-- Kalender libur mendukung upsert massal per tanggal.
CREATE TABLE holidays (
    tanggal DATE PRIMARY KEY,
    nama VARCHAR(200) NOT NULL,
    keterangan TEXT,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Cuti memakai saldo; izin memakai batas hari per kejadian.
ALTER TABLE leave_types ADD COLUMN kategori VARCHAR(10) NOT NULL DEFAULT 'cuti';
ALTER TABLE leave_types ADD COLUMN maksimal_hari SMALLINT;
ALTER TABLE leave_types ADD CONSTRAINT leave_types_kategori_check CHECK (kategori IN ('cuti', 'izin'));
ALTER TABLE leave_types ADD CONSTRAINT leave_types_limit_check CHECK (
    (kategori = 'cuti' AND maksimal_hari IS NULL)
    OR (kategori = 'izin' AND maksimal_hari IS NOT NULL AND maksimal_hari > 0)
);
ALTER TABLE leave_balances ADD COLUMN leave_type_id UUID REFERENCES leave_types(id) ON DELETE RESTRICT;
UPDATE leave_balances SET leave_type_id = (SELECT id FROM leave_types WHERE kategori='cuti' ORDER BY created_at,id LIMIT 1)
WHERE leave_type_id IS NULL;
ALTER TABLE leave_balances DROP CONSTRAINT IF EXISTS leave_balances_user_id_tahun_key;
CREATE UNIQUE INDEX uq_leave_balances_user_year_type ON leave_balances (user_id, tahun, leave_type_id) NULLS NOT DISTINCT;

-- Pendidikan dapat ditandai masih berjalan dengan tahun lulus NULL.
ALTER TABLE employee_education ADD COLUMN tahun_masuk SMALLINT;
ALTER TABLE employee_education ADD CONSTRAINT employee_education_entry_year_check
    CHECK (tahun_masuk IS NULL OR tahun_masuk BETWEEN 1900 AND 2200);
ALTER TABLE employee_education ADD CONSTRAINT employee_education_year_order_check
    CHECK (tahun_lulus IS NULL OR tahun_masuk IS NULL OR tahun_lulus >= tahun_masuk);

-- Koreksi jam absensi mengikuti approval atasan lalu HR.
CREATE TABLE attendance_corrections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    tanggal DATE NOT NULL,
    waktu_check_in TIME,
    waktu_check_out TIME,
    alasan TEXT NOT NULL,
    status VARCHAR(30) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT attendance_corrections_time_check CHECK (waktu_check_in IS NOT NULL OR waktu_check_out IS NOT NULL),
    CONSTRAINT attendance_corrections_status_check CHECK (status IN ('menunggu_atasan','menunggu_hr','disetujui','ditolak','dibatalkan'))
);
CREATE INDEX idx_attendance_corrections_user ON attendance_corrections (user_id, created_at DESC);
CREATE INDEX idx_attendance_corrections_pending ON attendance_corrections (status) WHERE status IN ('menunggu_atasan','menunggu_hr');

CREATE TABLE attendance_correction_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    correction_id UUID NOT NULL REFERENCES attendance_corrections(id) ON DELETE CASCADE,
    approver_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    tahap VARCHAR(10) NOT NULL CHECK (tahap IN ('atasan','hr')),
    keputusan VARCHAR(10) NOT NULL CHECK (keputusan IN ('approve','reject')),
    catatan TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS attendance_correction_approvals;
DROP TABLE IF EXISTS attendance_corrections;
ALTER TABLE employee_education DROP CONSTRAINT IF EXISTS employee_education_year_order_check;
ALTER TABLE employee_education DROP CONSTRAINT IF EXISTS employee_education_entry_year_check;
ALTER TABLE employee_education DROP COLUMN IF EXISTS tahun_masuk;
DROP INDEX IF EXISTS uq_leave_balances_user_year_type;
DELETE FROM leave_balances a USING leave_balances b
WHERE a.user_id=b.user_id AND a.tahun=b.tahun AND a.id>b.id;
ALTER TABLE leave_balances DROP COLUMN IF EXISTS leave_type_id;
ALTER TABLE leave_balances ADD CONSTRAINT leave_balances_user_id_tahun_key UNIQUE (user_id, tahun);
ALTER TABLE leave_types DROP CONSTRAINT IF EXISTS leave_types_limit_check;
ALTER TABLE leave_types DROP CONSTRAINT IF EXISTS leave_types_kategori_check;
ALTER TABLE leave_types DROP COLUMN IF EXISTS maksimal_hari;
ALTER TABLE leave_types DROP COLUMN IF EXISTS kategori;
DROP TABLE IF EXISTS holidays;
DROP TABLE IF EXISTS company_feeds;
ALTER TABLE employee_documents DROP COLUMN IF EXISTS document_type_id;
DROP TABLE IF EXISTS document_types;
