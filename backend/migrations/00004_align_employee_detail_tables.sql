-- +goose Up
-- Menyelaraskan nama tabel detail karyawan dengan Database Schema v1.1 dan menambahkan
-- kolom yang dibutuhkan schema OpenAPI 0.4.0. Lihat keputusan D-018.
ALTER TABLE emergency_contacts RENAME TO employee_emergency_contacts;
ALTER INDEX idx_emergency_contacts_employee_id RENAME TO idx_employee_emergency_contacts_employee_id;

ALTER TABLE education RENAME TO employee_education;
ALTER INDEX idx_education_employee_id RENAME TO idx_employee_education_employee_id;
ALTER TABLE employee_education
    RENAME CONSTRAINT education_graduation_year_check TO employee_education_graduation_year_check;

ALTER TABLE position_history RENAME TO employee_position_history;
ALTER INDEX idx_position_history_employee_id RENAME TO idx_employee_position_history_employee_id;
ALTER TABLE employee_position_history
    RENAME CONSTRAINT position_history_date_check TO employee_position_history_date_check;

ALTER TABLE salaries RENAME TO employee_salaries;
ALTER INDEX idx_salaries_employee_id RENAME TO idx_employee_salaries_employee_id;
ALTER TABLE employee_salaries
    RENAME CONSTRAINT salaries_period_check TO employee_salaries_period_check;
ALTER TABLE employee_salaries
    RENAME CONSTRAINT salaries_base_amount_check TO employee_salaries_base_amount_check;

ALTER TABLE documents RENAME TO employee_documents;
ALTER INDEX idx_documents_employee_id RENAME TO idx_employee_documents_employee_id;

-- Relasi organisasi pada riwayat jabatan agar response dapat mengirim objek
-- Department/Position tanpa menebak identitas.
ALTER TABLE employee_position_history
    ADD COLUMN department_id UUID REFERENCES departments(id) ON DELETE RESTRICT,
    ADD COLUMN position_id UUID REFERENCES positions(id) ON DELETE RESTRICT;

CREATE INDEX idx_employee_position_history_department_id
    ON employee_position_history (department_id);
CREATE INDEX idx_employee_position_history_position_id
    ON employee_position_history (position_id);

-- Komponen gaji yang dibutuhkan CurrentSalary dan formula payroll D-015.
ALTER TABLE employee_salaries
    ADD COLUMN tunjangan DECIMAL(14, 2) NOT NULL DEFAULT 0,
    ADD COLUMN potongan DECIMAL(14, 2) NOT NULL DEFAULT 0,
    ADD COLUMN take_home_pay DECIMAL(14, 2)
        GENERATED ALWAYS AS (gaji_pokok + tunjangan - potongan) STORED,
    ADD CONSTRAINT employee_salaries_allowance_check CHECK (tunjangan >= 0),
    ADD CONSTRAINT employee_salaries_deduction_check CHECK (potongan >= 0);

CREATE INDEX idx_employee_salaries_periode ON employee_salaries (periode);

-- Nama file asli dibutuhkan schema EmployeeDocument; uploaded_at dipetakan ke created_at.
ALTER TABLE employee_documents
    ADD COLUMN nama_file VARCHAR(255) NOT NULL DEFAULT '';

CREATE INDEX idx_employee_documents_uploaded_at ON employee_documents (uploaded_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_employee_documents_uploaded_at;
ALTER TABLE employee_documents DROP COLUMN IF EXISTS nama_file;

DROP INDEX IF EXISTS idx_employee_salaries_periode;
ALTER TABLE employee_salaries
    DROP CONSTRAINT IF EXISTS employee_salaries_deduction_check,
    DROP CONSTRAINT IF EXISTS employee_salaries_allowance_check,
    DROP COLUMN IF EXISTS take_home_pay,
    DROP COLUMN IF EXISTS potongan,
    DROP COLUMN IF EXISTS tunjangan;

DROP INDEX IF EXISTS idx_employee_position_history_position_id;
DROP INDEX IF EXISTS idx_employee_position_history_department_id;
ALTER TABLE employee_position_history
    DROP COLUMN IF EXISTS position_id,
    DROP COLUMN IF EXISTS department_id;

ALTER INDEX idx_employee_documents_employee_id RENAME TO idx_documents_employee_id;
ALTER TABLE employee_documents RENAME TO documents;

ALTER TABLE employee_salaries
    RENAME CONSTRAINT employee_salaries_base_amount_check TO salaries_base_amount_check;
ALTER TABLE employee_salaries
    RENAME CONSTRAINT employee_salaries_period_check TO salaries_period_check;
ALTER INDEX idx_employee_salaries_employee_id RENAME TO idx_salaries_employee_id;
ALTER TABLE employee_salaries RENAME TO salaries;

ALTER TABLE employee_position_history
    RENAME CONSTRAINT employee_position_history_date_check TO position_history_date_check;
ALTER INDEX idx_employee_position_history_employee_id RENAME TO idx_position_history_employee_id;
ALTER TABLE employee_position_history RENAME TO position_history;

ALTER TABLE employee_education
    RENAME CONSTRAINT employee_education_graduation_year_check TO education_graduation_year_check;
ALTER INDEX idx_employee_education_employee_id RENAME TO idx_education_employee_id;
ALTER TABLE employee_education RENAME TO education;

ALTER INDEX idx_employee_emergency_contacts_employee_id RENAME TO idx_emergency_contacts_employee_id;
ALTER TABLE employee_emergency_contacts RENAME TO emergency_contacts;
