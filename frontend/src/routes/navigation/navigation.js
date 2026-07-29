export const roles = {
  employee: "karyawan",
  supervisor: "atasan",
  hr: "hr",
  topManagement: "top_management",
};

const allRoles = Object.values(roles);
const personalRoles = [roles.employee, roles.supervisor, roles.hr];
const monitoringRoles = [roles.hr, roles.topManagement];

export const navigationItems = [
  { label: "Beranda", path: "/app", roles: allRoles },
  { label: "Profil Saya", path: "/app/profil", roles: personalRoles },
  { label: "Metrik Personal", path: "/app/metrik-personal", roles: personalRoles },
  { label: "Kehadiran Saya", path: "/app/absensi", roles: personalRoles },
  { label: "Pengajuan Saya", path: "/app/pengajuan", roles: personalRoles },
  {
    label: "Persetujuan",
    path: "/app/persetujuan",
    roles: [roles.supervisor, roles.hr, roles.topManagement],
  },
  { label: "Employee Database", path: "/app/karyawan", roles: monitoringRoles },
  { label: "Dashboard HR", path: "/app/dashboard", roles: monitoringRoles },
  { label: "Laporan Kehadiran", path: "/app/laporan-kehadiran", roles: monitoringRoles },
  { label: "AKSES", path: "/app/akses", roles: monitoringRoles },
  { label: "Audit Log", path: "/app/audit", roles: monitoringRoles },
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

