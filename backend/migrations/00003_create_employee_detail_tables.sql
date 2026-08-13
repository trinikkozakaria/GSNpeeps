-- +goose Up
CREATE TABLE employee_addresses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID NOT NULL UNIQUE REFERENCES employees(id) ON DELETE RESTRICT,
    jalan VARCHAR(255) NOT NULL,
    kelurahan VARCHAR(100),
    kecamatan VARCHAR(100),
    kota VARCHAR(100) NOT NULL,
    provinsi VARCHAR(100) NOT NULL
);

CREATE TABLE employee_ktp (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID NOT NULL UNIQUE REFERENCES employees(id) ON DELETE RESTRICT,
    nomor_ktp VARCHAR(20) NOT NULL UNIQUE,
    file_url VARCHAR(255)
);

CREATE TABLE employee_contracts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE RESTRICT,
    nomor_kontrak VARCHAR(50) NOT NULL UNIQUE,
    tanggal_mulai DATE NOT NULL,
    tanggal_berakhir DATE NOT NULL,
    jenis_kontrak VARCHAR(10) NOT NULL,
    file_url VARCHAR(255),
    status VARCHAR(10) NOT NULL DEFAULT 'aktif',
    CONSTRAINT employee_contracts_date_check CHECK (tanggal_berakhir >= tanggal_mulai),
    CONSTRAINT employee_contracts_type_check CHECK (jenis_kontrak IN ('PKWT', 'PKWTT')),
    CONSTRAINT employee_contracts_status_check CHECK (status IN ('aktif', 'berakhir'))
);

CREATE INDEX idx_employee_contracts_employee_id ON employee_contracts (employee_id);
CREATE INDEX idx_employee_contracts_end_date ON employee_contracts (tanggal_berakhir);

CREATE TABLE employee_bpjs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID NOT NULL UNIQUE REFERENCES employees(id) ON DELETE RESTRICT,
    no_bpjs_kesehatan VARCHAR(20),
    no_bpjs_ketenagakerjaan VARCHAR(20)
);

CREATE TABLE employee_npwp (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID NOT NULL UNIQUE REFERENCES employees(id) ON DELETE RESTRICT,
    nomor_npwp VARCHAR(25) UNIQUE,
    file_url VARCHAR(255)
);

CREATE TABLE emergency_contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE RESTRICT,
    nama VARCHAR(150) NOT NULL,
    hubungan VARCHAR(50),
    no_hp VARCHAR(20) NOT NULL
);

CREATE INDEX idx_emergency_contacts_employee_id ON emergency_contacts (employee_id);

CREATE TABLE education (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE RESTRICT,
    jenjang VARCHAR(20),
    institusi VARCHAR(150),
    tahun_lulus SMALLINT,
    CONSTRAINT education_graduation_year_check
        CHECK (tahun_lulus IS NULL OR tahun_lulus BETWEEN 1900 AND 2200)
);

CREATE INDEX idx_education_employee_id ON education (employee_id);

CREATE TABLE position_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE RESTRICT,
    jabatan VARCHAR(100) NOT NULL,
    tanggal_mulai DATE NOT NULL,
    tanggal_selesai DATE,
    CONSTRAINT position_history_date_check
        CHECK (tanggal_selesai IS NULL OR tanggal_selesai >= tanggal_mulai)
);

CREATE INDEX idx_position_history_employee_id ON position_history (employee_id);

CREATE TABLE salaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE RESTRICT,
    periode CHAR(7) NOT NULL,
    gaji_pokok DECIMAL(14, 2) NOT NULL,
    slip_url VARCHAR(255),
    UNIQUE (employee_id, periode),
    CONSTRAINT salaries_period_check CHECK (periode ~ '^[0-9]{4}-(0[1-9]|1[0-2])$'),
    CONSTRAINT salaries_base_amount_check CHECK (gaji_pokok >= 0)
);

CREATE INDEX idx_salaries_employee_id ON salaries (employee_id);

CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE RESTRICT,
    jenis_dokumen VARCHAR(100) NOT NULL,
    file_url VARCHAR(255) NOT NULL,
    uploaded_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_documents_employee_id ON documents (employee_id);

-- +goose Down
DROP TABLE IF EXISTS documents;
DROP TABLE IF EXISTS salaries;
DROP TABLE IF EXISTS position_history;
DROP TABLE IF EXISTS education;
DROP TABLE IF EXISTS emergency_contacts;
DROP TABLE IF EXISTS employee_npwp;
DROP TABLE IF EXISTS employee_bpjs;
DROP TABLE IF EXISTS employee_contracts;
DROP TABLE IF EXISTS employee_ktp;
DROP TABLE IF EXISTS employee_addresses;
