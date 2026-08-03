import { z } from "zod";

import { requestStatuses } from "../../../lib/request-status";
import { approvalHistorySchema } from "../../leave/schemas/leave-schema";

const uuidSchema = z.string().uuid();
const dateSchema = z.string().regex(/^\d{4}-\d{2}-\d{2}$/, "Format tanggal tidak valid");
const timeSchema = z.string().regex(/^\d{2}:\d{2}(:\d{2})?$/, "Format jam tidak valid");

export const overtimeSummarySchema = z.object({
  id: uuidSchema,
  employee_id: uuidSchema,
  nama_karyawan: z.string(),
  tanggal: dateSchema,
  waktu_mulai: z.string(),
  waktu_selesai: z.string(),
  total_jam: z.number(),
  status: z.enum(requestStatuses),
  created_at: z.string(),
});

export const overtimeDetailSchema = overtimeSummarySchema.extend({
  alasan: z.string().optional(),
  dokumen_url: z.string().nullable().optional(),
  approval_history: z.array(approvalHistorySchema).optional().default([]),
});

export const overtimeListSchema = z.object({
  items: z.array(overtimeSummarySchema),
  meta: z.object({
    page: z.number().int().positive(),
    limit: z.number().int().positive(),
    total_data: z.number().int().nonnegative(),
    total_page: z.number().int().nonnegative(),
  }),
});

export const overtimeRecapItemSchema = z.object({
  employee_id: uuidSchema,
  nama_karyawan: z.string(),
  departemen: z.string().optional(),
  total_pengajuan: z.number().int().nonnegative(),
  total_jam: z.number().nonnegative(),
});

export const overtimeRecapListSchema = z.array(overtimeRecapItemSchema);

export const createOvertimeFormSchema = z
  .object({
    tanggal: dateSchema,
    waktu_mulai: timeSchema,
    waktu_selesai: timeSchema,
    alasan: z
      .string()
      .trim()
      .min(10, "Alasan minimal 10 karakter")
      .max(2000, "Alasan maksimal 2000 karakter"),
  })
  .refine((value) => value.waktu_selesai > value.waktu_mulai, {
    path: ["waktu_selesai"],
    message: "Jam selesai harus setelah jam mulai",
  });
