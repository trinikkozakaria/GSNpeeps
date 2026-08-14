export const roles = {
  employee: "karyawan",
  supervisor: "atasan",
  hr: "hr",
  topManagement: "top_management",
};

const allRoles = Object.values(roles);
const personalRoles = [roles.employee, roles.supervisor, roles.hr];
const approvalRoles = [roles.supervisor, roles.hr, roles.topManagement];
const hrOnly = [roles.hr];

export const navigationItems = [
  { label: "Beranda", path: "/app", roles: allRoles },
  {
    label: "Pribadi",
    roles: personalRoles,
    children: [
      { label: "Profil Saya", path: "/app/profil", roles: personalRoles },
      { label: "Metrik Personal", path: "/app/metrik-personal", roles: personalRoles },
      { label: "Kehadiran Saya", path: "/app/absensi", roles: personalRoles },
      { label: "Koreksi Absensi", path: "/app/absensi/koreksi", roles: personalRoles },
    ],
  },
  {
    label: "Pengajuan",
    roles: personalRoles,
    children: [
      {
        label: "Ajukan Ketidakhadiran",
        path: "/app/absensi/ketidakhadiran",
        roles: personalRoles,
      },
      { label: "Ajukan Lembur", path: "/app/absensi/lembur", roles: personalRoles },
      { label: "Pengajuan Saya", path: "/app/pengajuan", roles: personalRoles },
    ],
  },
  { label: "Persetujuan", path: "/app/persetujuan", roles: approvalRoles },
  {
    label: "Organisasi",
    roles: hrOnly,
    children: [{ label: "Employee Database", path: "/app/karyawan", roles: hrOnly }],
  },
  {
    label: "Monitoring",
    roles: hrOnly,
    children: [
      { label: "Dashboard HR", path: "/app/dashboard", roles: hrOnly },
      { label: "Live Feed Absensi", path: "/app/live-feed", roles: hrOnly },
      { label: "Laporan Kehadiran", path: "/app/laporan-kehadiran", roles: hrOnly },
      { label: "Rekap Lembur", path: "/app/lembur/rekap", roles: hrOnly },
    ],
  },
  {
    label: "Informasi",
    roles: allRoles,
    children: [
      { label: "Kalender", path: "/app/kalender", roles: allRoles },
      { label: "Company Feed", path: "/app/company-feed", roles: hrOnly },
    ],
  },
  {
    label: "Master Data",
    roles: hrOnly,
    children: [
      { label: "Master Jenis Dokumen", path: "/app/master/jenis-dokumen", roles: hrOnly },
      { label: "Master Jenis Izin", path: "/app/master/jenis-izin", roles: hrOnly },
    ],
  },
  {
    label: "Administrasi",
    roles: hrOnly,
    children: [
      { label: "AKSES", path: "/app/akses", roles: hrOnly },
      { label: "Audit Log", path: "/app/audit", roles: hrOnly },
    ],
  },
  {
    label: "Akun",
    roles: allRoles,
    children: [
      { label: "Notifikasi", path: "/app/notifikasi", roles: allRoles },
      { label: "Keamanan Akun", path: "/app/keamanan", roles: allRoles },
    ],
  },
];

export const canAccessRoles = (role, allowedRoles) =>
  allRoles.includes(role) && allowedRoles.includes(role);

export const navigationForRole = (role) => {
  if (!allRoles.includes(role)) return [];

  return navigationItems.flatMap((item) => {
    if (!canAccessRoles(role, item.roles)) return [];
    if (!item.children) return [item];

    const children = item.children.filter((child) => canAccessRoles(role, child.roles));
    return children.length ? [{ ...item, children }] : [];
  });
};

export const roleLabel = {
  karyawan: "Karyawan",
  atasan: "Atasan",
  hr: "HR",
  top_management: "Top Management",
};
