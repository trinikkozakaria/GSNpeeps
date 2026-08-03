// Label Bahasa Indonesia untuk modul dan aksi. Nilai wire (`modul`, `aksi`) tidak pernah
// diubah; hanya tampilannya yang diterjemahkan.
export const moduleLabel = {
  karyawan: "Employee Database",
  dashboard: "Dashboard HR",
  absensi: "Kehadiran",
  laporan_kehadiran: "Laporan Kehadiran",
  ketidakhadiran: "Ketidakhadiran",
  lembur: "Lembur",
  notifikasi: "Notifikasi",
  akses: "AKSES",
  audit: "Audit Log",
};

export const actionLabel = {
  create: "Tambah",
  read: "Lihat",
  update: "Ubah",
  delete: "Hapus",
  approve: "Setujui",
  export: "Export",
};

export const readableModule = (modul) => moduleLabel[modul] ?? modul;
export const readableAction = (aksi) => actionLabel[aksi] ?? aksi;

/**
 * Mengelompokkan matriks datar dari API menjadi baris per modul untuk satu role. Kontrak
 * mengembalikan seluruh role dalam satu response (D-034), sehingga pengelompokan dilakukan
 * di client tanpa permintaan tambahan.
 */
export const groupPermissionsByModule = (permissions, roleId) => {
  const grouped = new Map();
  for (const permission of permissions) {
    if (permission.role_id !== roleId) continue;
    const actions = grouped.get(permission.modul) ?? [];
    actions.push(permission);
    grouped.set(permission.modul, actions);
  }
  return [...grouped.entries()]
    .map(([modul, actions]) => ({
      modul,
      label: readableModule(modul),
      actions: [...actions].sort((left, right) => left.aksi.localeCompare(right.aksi)),
    }))
    .sort((left, right) => left.label.localeCompare(right.label, "id-ID"));
};
