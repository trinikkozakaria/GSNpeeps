import { z } from "zod";

import { paginationMetaSchema } from "../../notifications/schemas/notification-schema";

// `user_id` dan `nama_user` kosong untuk aktor sistem seperti auto-escalation dan scheduler.
export const auditLogSchema = z.object({
  id: z.string(),
  user_id: z.string().nullish(),
  nama_user: z.string().nullish(),
  aksi: z.string(),
  modul: z.string(),
  resource_id: z.string().nullish(),
  detail: z.record(z.string(), z.unknown()).nullish(),
  ip_address: z.string().nullish(),
  created_at: z.string(),
});

export const auditLogListSchema = z.object({
  items: z.array(auditLogSchema),
  meta: paginationMetaSchema,
});
