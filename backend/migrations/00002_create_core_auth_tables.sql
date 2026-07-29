-- +goose Up
CREATE TABLE departments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nama VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE positions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nama VARCHAR(100) NOT NULL,
    department_id UUID NOT NULL REFERENCES departments(id) ON DELETE RESTRICT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (department_id, nama)
);

CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nama VARCHAR(30) NOT NULL UNIQUE,
    CONSTRAINT roles_nama_check CHECK (nama IN ('karyawan', 'atasan', 'hr', 'top_management'))
);

CREATE TABLE employees (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nip VARCHAR(20) NOT NULL UNIQUE,
    nama VARCHAR(150) NOT NULL,
    jenis_kelamin CHAR(1) NOT NULL,
    tanggal_lahir DATE NOT NULL,
    tanggal_join DATE NOT NULL,
    department_id UUID REFERENCES departments(id) ON DELETE RESTRICT,
    position_id UUID REFERENCES positions(id) ON DELETE RESTRICT,
    atasan_id UUID REFERENCES employees(id) ON DELETE RESTRICT,
    status_pernikahan VARCHAR(20),
    status VARCHAR(10) NOT NULL DEFAULT 'aktif',
    deleted_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT employees_jenis_kelamin_check CHECK (jenis_kelamin IN ('L', 'P')),
    CONSTRAINT employees_status_pernikahan_check
        CHECK (status_pernikahan IS NULL OR status_pernikahan IN ('lajang', 'menikah', 'cerai')),
    CONSTRAINT employees_status_check CHECK (status IN ('aktif', 'nonaktif')),
    CONSTRAINT employees_supervisor_not_self CHECK (atasan_id IS NULL OR atasan_id <> id)
);

CREATE INDEX idx_employees_atasan_id ON employees (atasan_id);
CREATE INDEX idx_employees_department_id ON employees (department_id);
CREATE INDEX idx_employees_status ON employees (status) WHERE deleted_at IS NULL;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID NOT NULL UNIQUE REFERENCES employees(id) ON DELETE RESTRICT,
    email VARCHAR(150) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
    failed_login_count INT NOT NULL DEFAULT 0,
    account_locked BOOLEAN NOT NULL DEFAULT FALSE,
    last_login_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT users_failed_login_count_check CHECK (failed_login_count >= 0),
    CONSTRAINT users_email_lowercase_check CHECK (email = LOWER(email))
);

CREATE INDEX idx_users_role_id ON users (role_id);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION enforce_single_top_management()
RETURNS TRIGGER AS $$
DECLARE
    top_management_role_id UUID;
BEGIN
    SELECT id INTO top_management_role_id FROM roles WHERE nama = 'top_management';
    IF NEW.role_id = top_management_role_id THEN
        PERFORM pg_advisory_xact_lock(hashtext('gsnpeeps_single_top_management'));
        IF EXISTS (
            SELECT 1 FROM users
            WHERE role_id = top_management_role_id
              AND id <> NEW.id
        ) THEN
            RAISE EXCEPTION 'only one top_management account is allowed'
                USING ERRCODE = '23505';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_single_top_management
BEFORE INSERT OR UPDATE OF role_id ON users
FOR EACH ROW EXECUTE FUNCTION enforce_single_top_management();

CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
    modul VARCHAR(50) NOT NULL,
    aksi VARCHAR(30) NOT NULL,
    diizinkan BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (role_id, modul, aksi)
);

CREATE INDEX idx_permissions_role_id ON permissions (role_id);

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE RESTRICT,
    aksi VARCHAR(30) NOT NULL,
    modul VARCHAR(50) NOT NULL,
    data_id UUID,
    detail JSONB,
    ip_address VARCHAR(45),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs (user_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at);

INSERT INTO roles (nama)
VALUES ('karyawan'), ('atasan'), ('hr'), ('top_management')
ON CONFLICT (nama) DO NOTHING;

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

-- +goose Down
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS permissions;
DROP TRIGGER IF EXISTS trg_single_top_management ON users;
DROP FUNCTION IF EXISTS enforce_single_top_management();
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS employees;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS positions;
DROP TABLE IF EXISTS departments;
