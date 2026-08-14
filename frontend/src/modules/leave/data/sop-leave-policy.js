const policyByCode = {
  "CUTI-TAHUNAN": "Hak 12 hari kerja per tahun setelah masa kerja 3 bulan, dihitung pro-rata. Sisa cuti dapat dibawa maksimal 6 hari ke tahun berikutnya sesuai persetujuan.",
  "IZIN-SAKIT": "Ketidakhadiran wajib dilaporkan kepada atasan paling lambat pukul 08.00. Sakit lebih dari 1 hari wajib dilengkapi surat dokter maksimal 2 hari kerja setelah kembali bekerja.",
  "MATERNITY-LEAVE": "Hak standar 90 hari. Perpanjangan bulan ke-4 sampai ke-6 merupakan benefit tambahan perusahaan dan diproses bersama HR (gaji berturut-turut 75%, 50%, dan 25% base salary).",
  "EXT-MATERNITY": "Benefit tambahan bulan ke-4 sampai ke-6 setelah Maternity Leave standar. Pembayaran base salary: bulan ke-4 75%, bulan ke-5 50%, dan bulan ke-6 25%.",
  "UNPAID-LEAVE": "Untuk keperluan pribadi di luar kategori izin khusus. Diajukan sebagai izin tidak dibayar atau dapat dialihkan ke Cuti Tahunan sesuai persetujuan atasan dan HR.",
};

const SOP_SPECIAL_LEAVE_CODES = new Set([
  "IZIN-NIKAH",
  "IZIN-NIKAH-ANAK",
  "IZIN-KHITAN-BAPTIS",
  "IZIN-ISTRI-MELAHIRKAN",
  "IZIN-KEGUGURAN",
  "IZIN-HAID",
  "IZIN-DUKA-INTI",
  "IZIN-DUKA-SERUMAH",
  "IZIN-HAJI",
  "IZIN-UMROH",
  "IZIN-RAWAT-KELUARGA",
  "PATERNITY-LEAVE",
]);

export const getLeavePolicy = (type) => {
  if (!type) return "";
  if (policyByCode[type.kode]) return policyByCode[type.kode];
  if (SOP_SPECIAL_LEAVE_CODES.has(type.kode)) {
    return "Izin khusus berbayar sesuai durasi maksimal pada master dan tetap memerlukan persetujuan atasan/HR.";
  }
  return "Ketentuan mengikuti master jenis izin dan keputusan approver.";
};

export const formatLeaveAllowance = (type) => {
  if (!type) return "";
  if (type.kategori === "cuti") return `${type.kuota_tahunan} hari kerja per tahun`;
  if (type.kode === "IZIN-SAKIT") return "Sesuai durasi sakit (batas sistem 365 hari per pengajuan)";
  if (type.kode === "UNPAID-LEAVE") return "Durasi sesuai persetujuan (batas sistem 365 hari per pengajuan)";
  return `Maksimal ${type.maksimal_hari} hari per pengajuan`;
};

// Perhitungan backend memakai hari kalender inklusif. Karena itu izin 3 hari yang
// dimulai tanggal 10 memiliki tanggal selesai maksimal tanggal 12.
export const getMaximumEndDate = (startDate, maximumDays) => {
  if (!startDate || !Number.isInteger(maximumDays) || maximumDays < 1) return "";
  const date = new Date(`${startDate}T00:00:00Z`);
  if (Number.isNaN(date.getTime())) return "";
  date.setUTCDate(date.getUTCDate() + maximumDays - 1);
  return date.toISOString().slice(0, 10);
};
