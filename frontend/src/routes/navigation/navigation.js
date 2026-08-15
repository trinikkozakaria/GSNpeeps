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
  {
    label: "Persetujuan",
    roles: approvalRoles,
    children: [
      {
        label: "Ketidakhadiran",
        path: "/app/persetujuan",
        query: { tab: "ketidakhadiran" },
        isDefaultQuery: true,
        roles: approvalRoles,
      },
      {
        label: "Lembur",
        path: "/app/persetujuan",
        query: { tab: "lembur" },
        roles: approvalRoles,
      },
      {
        label: "Koreksi Absensi",
        // Sama teksnya dengan item "Koreksi Absensi" di grup Pribadi (pengajuan sendiri)
        // tapi menuju antrean persetujuan; aria-label dibedakan agar accessible name unik.
        ariaLabel: "Koreksi Absensi (Persetujuan)",
        path: "/app/persetujuan",
        query: { tab: "koreksi" },
        roles: [roles.supervisor, roles.hr],
      },
    ],
  },
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

// Seluruh path leaf yang terdaftar di sidebar, dipakai isPathActive untuk menentukan item
// mana yang paling spesifik terhadap path yang sedang diakses.
const flattenPaths = (items) =>
  items.flatMap((item) => (item.children ? item.children.map((child) => child.path) : item.path ? [item.path] : []));

export const allNavigationPaths = flattenPaths(navigationItems);

// isPathActive mencegah item sidebar seperti /app/absensi tetap "aktif" hanya karena path
// yang sedang diakses (mis. /app/absensi/lembur) kebetulan berbagi prefix — item lain yang
// path-nya terdaftar dan lebih spesifik selalu menang.
export const isPathActive = (itemPath, pathname, allPaths = allNavigationPaths) => {
  if (pathname === itemPath) return true;
  if (!pathname.startsWith(`${itemPath}/`)) return false;
  return !allPaths.some(
    (otherPath) =>
      otherPath !== itemPath &&
      otherPath.length > itemPath.length &&
      (pathname === otherPath || pathname.startsWith(`${otherPath}/`)),
  );
};

// isNavItemActive menangani dua bentuk item: path biasa (isPathActive) dan item yang
// membedakan diri lewat query string pada path yang sama, seperti tab Persetujuan.
export const isNavItemActive = (item, pathname, search) => {
  if (!item.query) return isPathActive(item.path, pathname);
  if (item.path !== pathname) return false;
  const current = new URLSearchParams(search);
  return Object.entries(item.query).every(([key, value]) => {
    const currentValue = current.get(key);
    return currentValue === null ? Boolean(item.isDefaultQuery) : currentValue === value;
  });
};

// navLinkTarget membangun props `to` React Router untuk item biasa maupun item berbasis query.
export const navLinkTarget = (item) =>
  item.query ? { pathname: item.path, search: `?${new URLSearchParams(item.query).toString()}` } : item.path;

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
