export const roles = {
  employee: "karyawan",
  supervisor: "atasan",
  hr: "hr",
  topManagement: "top_management",
};

const allRoles = Object.values(roles);
const personalRoles = [roles.employee, roles.supervisor, roles.hr];
const hrOnly = [roles.hr];

export const navigationItems = [
  { label: "Dashboard HR", path: "/app/dashboard", roles: hrOnly },
  { label: "Beranda", path: "/app", roles: allRoles },
  { label: "Profil Saya", path: "/app/profil", roles: personalRoles },
  { label: "Metrik Personal", path: "/app/metrik-personal", roles: personalRoles },
  {
    label: "Kehadiran Saya",
    path: "/app/absensi",
    roles: personalRoles,
    // NavLink tanpa `end` sudah mencocokkan berdasarkan prefix path, sehingga item ini
    // otomatis tersorot aktif ketika sub-alur /app/absensi/ketidakhadiran atau
    // /app/absensi/lembur sedang dibuka; tidak perlu logika aktif tambahan.
  },
  { label: "Koreksi Absensi", path: "/app/absensi/koreksi", roles: personalRoles },
  {
    // "Pengajuan" hanya label pengelompokan (UI saja); tidak pernah menjadi link sendiri.
    // Ajukan Ketidakhadiran dan Ajukan Lembur tetap memakai route yang sama seperti
    // sebelumnya, hanya ditampilkan bertingkat/indented di sidebar.
    label: "Pengajuan",
    roles: personalRoles,
    children: [
      { label: "Ajukan Ketidakhadiran", path: "/app/absensi/ketidakhadiran" },
      { label: "Ajukan Lembur", path: "/app/absensi/lembur" },
    ],
  },
  { label: "Pengajuan Saya", path: "/app/pengajuan", roles: personalRoles },
  {
    label: "Persetujuan",
    path: "/app/persetujuan",
    roles: [roles.supervisor, roles.hr, roles.topManagement],
  },
  { label: "Employee Database", path: "/app/karyawan", roles: hrOnly },
  { label: "Live Feed Absensi", path: "/app/live-feed", roles: hrOnly },
  { label: "Laporan Kehadiran", path: "/app/laporan-kehadiran", roles: hrOnly },
  { label: "Kalender", path: "/app/kalender", roles: allRoles },
  { label: "Company Feed", path: "/app/company-feed", roles: hrOnly },
  { label: "Master Jenis Dokumen", path: "/app/master/jenis-dokumen", roles: hrOnly },
  { label: "Master Jenis Izin", path: "/app/master/jenis-izin", roles: [roles.hr] },
  { label: "Rekap Lembur", path: "/app/lembur/rekap", roles: [roles.hr] },
  { label: "AKSES", path: "/app/akses", roles: hrOnly },
  { label: "Audit Log", path: "/app/audit", roles: hrOnly },
  { label: "Notifikasi", path: "/app/notifikasi", roles: allRoles },
  { label: "Keamanan Akun", path: "/app/keamanan", roles: allRoles },
];

export const canAccessRoles = (role, allowedRoles) =>
  allRoles.includes(role) && allowedRoles.includes(role);

export const navigationForRole = (role) =>
  navigationItems.filter((item) => canAccessRoles(role, item.roles));

export const roleLabel = {
  karyawan: "Karyawan",
  atasan: "Atasan",
  hr: "HR",
  top_management: "Top Management",
};

