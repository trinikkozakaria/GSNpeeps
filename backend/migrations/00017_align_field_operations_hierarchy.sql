-- +goose Up
-- Menyamakan jalur Field Operations dengan bagan organisasi resmi. Pembaruan ini
-- eksplisit karena 00016 mungkin sudah terpasang pada instalasi yang sedang berjalan.
WITH hierarchy(child_name, supervisor_name) AS (
    VALUES
        ('Yohanes Otto Hasudungan', 'Pekik Satria Andika'),
        ('Afdah adi ugi', 'Pekik Satria Andika'),
        ('Riza Perdana', 'Pekik Satria Andika'),
        ('Yosafat', 'Riza Perdana'),
        ('Rapidal Ajis', 'Riza Perdana')
)
UPDATE employees AS child
SET atasan_id = supervisor.id, updated_at = NOW()
FROM hierarchy AS relation
JOIN LATERAL (
    SELECT id
    FROM employees
    WHERE nama = relation.supervisor_name AND deleted_at IS NULL
    ORDER BY created_at, id
    LIMIT 1
) AS supervisor ON TRUE
WHERE child.nama = relation.child_name
  AND child.deleted_at IS NULL
  AND child.id <> supervisor.id;

-- +goose Down
-- Relasi tidak dipulihkan otomatis karena dapat sudah dipakai dalam alur persetujuan.
SELECT 1;
