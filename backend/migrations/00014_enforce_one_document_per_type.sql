-- +goose Up
-- Setiap karyawan hanya boleh memiliki satu dokumen terdaftar per jenis dokumen master
-- (mis. hanya satu dokumen "NPWP"). Upload berikutnya untuk jenis yang sama menggantikan
-- (upsert) baris yang ada, bukan menambah baris baru. document_type_id sebelumnya tidak
-- pernah diisi oleh kode aplikasi sehingga seluruh baris lama bernilai NULL dan constraint
-- ini aman diterapkan; NULL tetap diperbolehkan berulang sesuai semantik UNIQUE Postgres.
ALTER TABLE employee_documents
    ADD CONSTRAINT employee_documents_employee_type_unique UNIQUE (employee_id, document_type_id);

-- +goose Down
ALTER TABLE employee_documents
    DROP CONSTRAINT IF EXISTS employee_documents_employee_type_unique;
