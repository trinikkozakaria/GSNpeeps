-- +goose Up
-- Mengisi garis pelaporan yang masih kosong dari struktur organisasi yang disetujui.
-- Relasi yang sudah diatur pengguna tidak ditimpa dan roster yang belum diimpor menjadi no-op.
WITH hierarchy(child_name, supervisor_name) AS (
    VALUES
        ('Rachmat Efendi', 'Anthony Hamonangan Sihombing'),
        ('Nadya Theresia Sihombing', 'Anthony Hamonangan Sihombing'),
        ('Mesianti Puspawardhany', 'Anthony Hamonangan Sihombing'),
        ('Dewa Pambudhi', 'Anthony Hamonangan Sihombing'),
        ('Beniqno Joe Prasetyo Siilitonga', 'Anthony Hamonangan Sihombing'),
        ('Bima sanjaya', 'Rachmat Efendi'),
        ('Arif Triantoro', 'Rachmat Efendi'),
        ('Rini Yulianto', 'Rachmat Efendi'),
        ('Refina Anastasya', 'Rachmat Efendi'),
        ('Nadia Cahya Rani', 'Nadya Theresia Sihombing'),
        ('Afrizal Ari Jotivian', 'Nadia Cahya Rani'),
        ('Ray Strainer, S.H.', 'Nadia Cahya Rani'),
        ('Destaria Dwi Maryastuti', 'Nadia Cahya Rani'),
        ('Arif Hidayat', 'Afrizal Ari Jotivian'),
        ('Saher Remal Agungta Ketaren', 'Mesianti Puspawardhany'),
        ('Adisty Aulia Rosadi', 'Saher Remal Agungta Ketaren'),
        ('Ratu Fasya Dwinata Yusup', 'Saher Remal Agungta Ketaren'),
        ('Fajriani Oktavianur', 'Saher Remal Agungta Ketaren'),
        ('Naini Pujiati', 'Dewa Pambudhi'),
        ('Pekik Satria Andika', 'Dewa Pambudhi'),
        ('Yohanes Otto Hasudungan', 'Dewa Pambudhi'),
        ('Afdah adi ugi', 'Dewa Pambudhi'),
        ('Riza Perdana', 'Dewa Pambudhi'),
        ('Wira Sanjaya', 'Dewa Pambudhi'),
        ('Akhyar Muhataris Adhin', 'Dewa Pambudhi'),
        ('Muhammad Rifqy Prawira', 'Dewa Pambudhi'),
        ('Syifa Handayani', 'Naini Pujiati'),
        ('Elis Syubban Al Fatih', 'Naini Pujiati'),
        ('Aji Braja Yudha', 'Pekik Satria Andika'),
        ('Aris Setiawan', 'Yohanes Otto Hasudungan'),
        ('Adam Purwa Bagaskara', 'Aris Setiawan'),
        ('Muhamad Rizaldi', 'Afdah adi ugi'),
        ('Rendi Oktobiwanto', 'Afdah adi ugi'),
        ('David Vio Ariyanda Putra', 'Riza Perdana'),
        ('Yosafat', 'David Vio Ariyanda Putra'),
        ('Rapidal Ajis', 'David Vio Ariyanda Putra'),
        ('Bimawan Zakaria', 'Wira Sanjaya'),
        ('Wawan Suryana', 'Bimawan Zakaria'),
        ('Adhi Prihatmoko', 'Bimawan Zakaria'),
        ('Darrel Hutomi', 'Bimawan Zakaria'),
        ('Alfian Purnomo', 'Akhyar Muhataris Adhin'),
        ('Tri Nikko Zakaria', 'Akhyar Muhataris Adhin'),
        ('Benedicto Joko Ferdinand Silitonga', 'Akhyar Muhataris Adhin'),
        ('Muhammad Al-Ghazali', 'Akhyar Muhataris Adhin'),
        ('Agil Wahyudi Ariyanto', 'Muhammad Al-Ghazali'),
        ('Zainuddin oky wijaya', 'Muhammad Al-Ghazali'),
        ('Dimas Artha Prasetya', 'Beniqno Joe Prasetyo Siilitonga'),
        ('Varrel Arya Yudhanto', 'Beniqno Joe Prasetyo Siilitonga')
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
  AND child.atasan_id IS NULL
  AND child.id <> supervisor.id;

-- +goose Down
-- Data organisasi tidak dihapus otomatis saat rollback karena relasi dapat sudah dipakai
-- oleh approval. Koreksi berikutnya harus dilakukan lewat migration maju yang eksplisit.
SELECT 1;
