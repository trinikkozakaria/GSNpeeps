-- +goose Up
-- Kapabilitas matriks baru: `berkas` / `read` menggerbang proxy berkas statis
-- `GET /api/v1/media`. Default diizinkan untuk seluruh role; scoping per-berkas
-- (milik sendiri untuk Karyawan, milik sendiri + bawahan langsung untuk Atasan,
-- seluruhnya untuk HR dan Top Management) tetap ditegakkan di handler. Mematikan sel
-- ini dari halaman AKSES mencabut seluruh akses berkas statis bagi role tersebut.
INSERT INTO permissions (role_id, modul, aksi, diizinkan)
SELECT id, 'berkas', 'read', TRUE
FROM roles
ON CONFLICT (role_id, modul, aksi) DO UPDATE SET diizinkan = EXCLUDED.diizinkan;

-- +goose Down
DELETE FROM permissions WHERE modul = 'berkas';
