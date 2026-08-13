import { notificationReferenceTypes } from "../schemas/notification-schema";

/**
 * Label dan ikon per tipe notifikasi. Status tidak pernah dibedakan hanya oleh warna: setiap
 * baris membawa ikon dan teks kategori sehingga tetap terbaca pada mode kontras tinggi.
 */
const presentation = {
  ketidakhadiran_baru: { icon: "📝", label: "Pengajuan ketidakhadiran" },
  lembur_baru: { icon: "🌙", label: "Pengajuan lembur" },
  keputusan_approve: { icon: "✅", label: "Disetujui" },
  keputusan_reject: { icon: "⛔", label: "Ditolak" },
  auto_escalate: { icon: "⏫", label: "Eskalasi otomatis" },
  delegasi: { icon: "🔁", label: "Delegasi" },
  kontrak_akan_habis: { icon: "📄", label: "Kontrak akan berakhir" },
};

const fallbackPresentation = { icon: "🔔", label: "Notifikasi" };

export const notificationPresentation = (tipe) => presentation[tipe] ?? fallbackPresentation;

/**
 * Pemetaan deep link internal. Backend tidak pernah mengirim URL, hanya pasangan
 * `reference_type` dan `reference_id`, sehingga tautan tidak dapat mengarah keluar aplikasi.
 * Pasangan yang tidak dikenal menghasilkan null dan barisnya tampil tanpa tautan.
 */
const referenceRoutes = {
  ketidakhadiran: (id) => `/app/persetujuan/ketidakhadiran/${id}`,
  lembur: (id) => `/app/persetujuan/lembur/${id}`,
  karyawan: (id) => `/app/karyawan/${id}`,
};

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

export const notificationDeepLink = (notification) => {
  const type = notification?.reference_type;
  const id = notification?.reference_id;
  if (!type || !id) return null;
  if (!notificationReferenceTypes.includes(type)) return null;
  // ID harus berbentuk UUID; nilai lain tidak pernah disusun menjadi path.
  if (!uuidPattern.test(id)) return null;
  return referenceRoutes[type]?.(id) ?? null;
};

/**
 * Pemohon tidak memiliki halaman detail approval untuk pengajuannya sendiri, sehingga
 * notifikasi keputusan mengarah ke daftar pengajuan miliknya.
 */
const decisionTypes = new Set(["keputusan_approve", "keputusan_reject"]);

export const notificationTargetForRole = (notification, role) => {
  if (decisionTypes.has(notification?.tipe)) {
    return role === "top_management" ? null : "/app/pengajuan";
  }
  if (notification?.reference_type === "karyawan" && role !== "hr" && role !== "top_management") {
    // Hanya HR dan Top Management dapat membuka detail karyawan.
    return null;
  }
  return notificationDeepLink(notification);
};
