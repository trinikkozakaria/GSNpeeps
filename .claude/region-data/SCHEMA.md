# Data wilayah

Folder ini disiapkan untuk dataset wilayah Indonesia yang dapat dipakai pada alamat karyawan.

Jangan membuat atau menebak data wilayah. Jika dataset resmi diberikan, gunakan nama file berikut:

- `provinces.csv`: `id,name`
- `regencies.csv`: `id,province_id,name`
- `districts.csv`: `id,regency_id,name`
- `villages.csv`: `id,district_id,name`

Validasi encoding UTF-8, keunikan ID, dan seluruh foreign key sebelum membuat seed/import. Catat sumber dan tanggal versi dataset di file `SOURCE.md` saat data nyata ditambahkan.

