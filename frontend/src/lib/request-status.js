// Enum status pengajuan berasal dari API dan tidak pernah diubah di client (D-005).
// Label Bahasa Indonesia hanya dibuat di sini untuk kebutuhan tampilan.
export const requestStatuses = [
  "menunggu_atasan",
  "menunggu_hr",
  "menunggu_top_management",
  "disetujui",
  "ditolak",
  "dibatalkan",
];

export const requestStatusLabel = {
  menunggu_atasan: "Menunggu Atasan",
  menunggu_hr: "Menunggu HR",
  menunggu_top_management: "Menunggu Top Management",
  disetujui: "Disetujui",
  ditolak: "Ditolak",
  dibatalkan: "Dibatalkan",
};

export const isPendingStatus = (status) =>
  status === "menunggu_atasan" ||
  status === "menunggu_hr" ||
  status === "menunggu_top_management";

// Label keputusan pada riwayat approval; `auto_eskalasi` dipicu sistem setelah SLA 2x24 jam.
export const approvalDecisionLabel = {
  disetujui: "Disetujui",
  ditolak: "Ditolak",
  didelegasikan: "Didelegasikan ke HR",
  auto_eskalasi: "Dieskalasi otomatis ke HR",
};

export const approvalStageLabel = {
  atasan: "Atasan",
  hr: "HR",
  top_management: "Top Management",
};

export const workModeLabel = { WFO: "WFO", WFH: "WFH", WFA: "WFA" };

export const attendanceStatusLabel = {
  tepat_waktu: "Tepat waktu",
  terlambat: "Terlambat",
  pulang_cepat: "Pulang cepat",
  valid: "Valid",
};
