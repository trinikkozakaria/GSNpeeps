-- +goose Up
-- Migration 00009 pernah tercatat pada sejumlah database sebelum seluruh objek UAT di
-- bawah ini masuk ke file tersebut. Goose tidak menjalankan ulang nomor versi yang sudah
-- tercatat, jadi migration baru ini memperbaiki schema lama secara idempotent. Pada
-- instalasi baru, seluruh statement menjadi no-op setelah 00009 berhasil dijalankan.

CREATE TABLE IF NOT EXISTS document_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kode VARCHAR(50) NOT NULL UNIQUE,
    nama VARCHAR(150) NOT NULL UNIQUE,
    wajib BOOLEAN NOT NULL DEFAULT TRUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE employee_documents
    ADD COLUMN IF NOT EXISTS document_type_id UUID REFERENCES document_types(id) ON DELETE RESTRICT;
CREATE INDEX IF NOT EXISTS idx_employee_documents_type ON employee_documents (document_type_id);

CREATE TABLE IF NOT EXISTS company_feeds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    judul VARCHAR(200) NOT NULL,
    konten_html TEXT NOT NULL,
    published_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT company_feeds_content_check CHECK (length(trim(konten_html)) > 0)
);
CREATE INDEX IF NOT EXISTS idx_company_feeds_published ON company_feeds (published_at DESC);

CREATE TABLE IF NOT EXISTS holidays (
    tanggal DATE PRIMARY KEY,
    nama VARCHAR(200) NOT NULL,
    keterangan TEXT,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE leave_types ADD COLUMN IF NOT EXISTS kategori VARCHAR(10) NOT NULL DEFAULT 'cuti';
ALTER TABLE leave_types ADD COLUMN IF NOT EXISTS maksimal_hari SMALLINT;

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'leave_types_kategori_check') THEN
        ALTER TABLE leave_types ADD CONSTRAINT leave_types_kategori_check
            CHECK (kategori IN ('cuti', 'izin'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'leave_types_limit_check') THEN
        ALTER TABLE leave_types ADD CONSTRAINT leave_types_limit_check CHECK (
            (kategori = 'cuti' AND maksimal_hari IS NULL)
            OR (kategori = 'izin' AND maksimal_hari IS NOT NULL AND maksimal_hari > 0)
        );
    END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE leave_balances
    ADD COLUMN IF NOT EXISTS leave_type_id UUID REFERENCES leave_types(id) ON DELETE RESTRICT;
UPDATE leave_balances
SET leave_type_id = (SELECT id FROM leave_types WHERE kategori = 'cuti' ORDER BY created_at, id LIMIT 1)
WHERE leave_type_id IS NULL;
ALTER TABLE leave_balances DROP CONSTRAINT IF EXISTS leave_balances_user_id_tahun_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_leave_balances_user_year_type
    ON leave_balances (user_id, tahun, leave_type_id) NULLS NOT DISTINCT;

ALTER TABLE employee_education ADD COLUMN IF NOT EXISTS tahun_masuk SMALLINT;
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'employee_education_entry_year_check') THEN
        ALTER TABLE employee_education ADD CONSTRAINT employee_education_entry_year_check
            CHECK (tahun_masuk IS NULL OR tahun_masuk BETWEEN 1900 AND 2200);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'employee_education_year_order_check') THEN
        ALTER TABLE employee_education ADD CONSTRAINT employee_education_year_order_check
            CHECK (tahun_lulus IS NULL OR tahun_masuk IS NULL OR tahun_lulus >= tahun_masuk);
    END IF;
END $$;
-- +goose StatementEnd

CREATE TABLE IF NOT EXISTS attendance_corrections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    tanggal DATE NOT NULL,
    waktu_check_in TIME,
    waktu_check_out TIME,
    alasan TEXT NOT NULL,
    status VARCHAR(30) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT attendance_corrections_time_check
        CHECK (waktu_check_in IS NOT NULL OR waktu_check_out IS NOT NULL),
    CONSTRAINT attendance_corrections_status_check
        CHECK (status IN ('menunggu_atasan','menunggu_hr','disetujui','ditolak','dibatalkan'))
);
CREATE INDEX IF NOT EXISTS idx_attendance_corrections_user
    ON attendance_corrections (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_attendance_corrections_pending
    ON attendance_corrections (status) WHERE status IN ('menunggu_atasan','menunggu_hr');

CREATE TABLE IF NOT EXISTS attendance_correction_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    correction_id UUID NOT NULL REFERENCES attendance_corrections(id) ON DELETE CASCADE,
    approver_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    tahap VARCHAR(10) NOT NULL CHECK (tahap IN ('atasan','hr')),
    keputusan VARCHAR(10) NOT NULL CHECK (keputusan IN ('approve','reject')),
    catatan TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
-- Repair migration sengaja tidak menghapus objek yang secara canonical dimiliki 00009.
SELECT 1;
