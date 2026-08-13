import { z } from "zod";

// Katalog tipe notifikasi mengikuti CHECK constraint tabel notifications dan katalog event
// backend. Tipe di luar katalog tetap dirender dengan label netral, bukan ditolak, agar satu
// tipe baru dari server tidak mengosongkan inbox pengguna.
export const notificationTypes = [
  "ketidakhadiran_baru",
  "lembur_baru",
  "keputusan_approve",
  "keputusan_reject",
  "auto_escalate",
  "delegasi",
  "kontrak_akan_habis",
];

export const notificationReferenceTypes = ["ketidakhadiran", "lembur", "karyawan"];

export const notificationSchema = z.object({
  id: z.string(),
  user_id: z.string(),
  tipe: z.string(),
  judul: z.string(),
  pesan: z.string(),
  reference_id: z.string().nullish(),
  reference_type: z.string().nullish(),
  is_read: z.boolean(),
  read_at: z.string().nullish(),
  created_at: z.string(),
});

export const paginationMetaSchema = z.object({
  page: z.number(),
  limit: z.number(),
  total_data: z.number(),
  total_page: z.number(),
});

export const notificationListSchema = z.object({
  items: z.array(notificationSchema),
  meta: paginationMetaSchema,
});

export const unreadCountSchema = z.object({
  unread_count: z.number().int().min(0),
});
