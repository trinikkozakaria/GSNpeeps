/**
 * Redaksi sisi client untuk detail Audit Log.
 *
 * Backend sudah meredaksi field sensitif sebelum mengirim response. Lapisan ini bertahan
 * seandainya modul baru menulis field sensitif yang belum masuk daftar server: nilainya tetap
 * tidak pernah muncul di layar HR atau Top Management.
 */
const sensitiveFragments = [
  "password",
  "token",
  "hash",
  "secret",
  "authorization",
  "session",
  "credential",
  "gaji",
  "salary",
  "npwp",
  "nik",
  "ktp",
  "rekening",
  "bank",
  "foto",
  "dokumen_url",
  "file_url",
];

export const redactedPlaceholder = "[REDACTED]";

const isSensitiveKey = (key) => {
  const lowered = String(key).toLowerCase();
  return sensitiveFragments.some((fragment) => lowered.includes(fragment));
};

export const redactAuditDetail = (detail) => {
  if (detail === null || detail === undefined) return detail;
  if (Array.isArray(detail)) return detail.map((item) => redactAuditDetail(item));
  if (typeof detail !== "object") return detail;

  const result = {};
  for (const [key, value] of Object.entries(detail)) {
    result[key] = isSensitiveKey(key) ? redactedPlaceholder : redactAuditDetail(value);
  }
  return result;
};

/**
 * Detail dirender sebagai teks JSON di dalam elemen `pre`, bukan sebagai HTML, sehingga nilai
 * apa pun dari server tidak dapat menyuntikkan markup.
 */
export const formatAuditDetail = (detail) => {
  if (!detail || Object.keys(detail).length === 0) return "";
  return JSON.stringify(redactAuditDetail(detail), null, 2);
};
