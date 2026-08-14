import { z } from "zod";

import { requestStatuses } from "../../../lib/request-status";

const uuidSchema = z.string().uuid();
const dateSchema = z.string().regex(/^\d{4}-\d{2}-\d{2}$/, "Format tanggal tidak valid");

export const leaveTypeSchema = z.object({
  id: uuidSchema,
  kode: z.string(),
  nama: z.string(),
  kuota_tahunan: z.number().int().nonnegative(),
  kategori: z.enum(["cuti", "izin"]).default("cuti"),
  maksimal_hari: z.number().int().positive().nullable().optional(),
  memerlukan_dokumen: z.boolean().optional(),
  is_active: z.boolean(),
});

export const leaveTypeListSchema = z.array(leaveTypeSchema);

export const approvalHistorySchema = z.object({
  tahap: z.enum(["atasan", "hr", "top_management"]),
  approver_id: uuidSchema.nullable().optional(),
  approver_nama: z.string().nullable().optional(),
  keputusan: z.enum(["disetujui", "ditolak", "didelegasikan", "auto_eskalasi"]),
  catatan: z.string().nullable().optional(),
  decided_at: z.string(),
});

export const leaveRequestSummarySchema = z.object({
  id: uuidSchema,
  employee_id: uuidSchema,
  nama_karyawan: z.string(),
  jenis_izin: z.string(),
  tanggal_mulai: dateSchema,
  tanggal_selesai: dateSchema,
  jumlah_hari: z.number().int().positive(),
  status: z.enum(requestStatuses),
  created_at: z.string(),
});

export const leaveRequestDetailSchema = leaveRequestSummarySchema.extend({
  alasan: z.string().optional(),
  dokumen_url: z.string().nullable().optional(),
  lokasi_tujuan: z.string().nullable().optional(),
  keterangan_lokasi: z.string().nullable().optional(),
  approval_history: z.array(approvalHistorySchema).optional().default([]),
});

const paginationSchema = z.object({
  page: z.number().int().positive(),
  limit: z.number().int().positive(),
  total_data: z.number().int().nonnegative(),
  total_page: z.number().int().nonnegative(),
});

export const leaveRequestListSchema = z.object({
  items: z.array(leaveRequestSummarySchema),
  meta: paginationSchema,
});

export const requestStateSchema = z.object({
  id: uuidSchema,
  status: z.enum(requestStatuses),
});

// Nama jenis izin yang menandai Perjalanan Dinas; field lokasi menjadi wajib untuknya.
export const isTravelLeaveType = (name) =>
  typeof name === "string" && name.toLowerCase().includes("perjalanan dinas");

export const maxDocumentBytes = 5 * 1024 * 1024;
const allowedDocumentExtensions = [".pdf", ".jpg", ".jpeg", ".png"];

export const validateSupportingDocument = (file, required) => {
  if (!file) return required ? "Dokumen pendukung wajib diunggah untuk jenis izin ini." : "";
  const extension = file.name.slice(file.name.lastIndexOf(".")).toLowerCase();
  if (!file.name.includes(".") || !allowedDocumentExtensions.includes(extension)) {
    return "Format dokumen tidak didukung. Gunakan PDF, JPG, atau PNG.";
  }
  if (file.size > maxDocumentBytes) return "Ukuran dokumen melebihi batas 5 MB.";
  if (file.size === 0) return "Berkas dokumen kosong.";
  return "";
};

export const createLeaveFormSchema = z
  .object({
    jenis_izin_id: uuidSchema.or(z.literal("")).refine((value) => value !== "", {
      message: "Jenis izin wajib dipilih",
    }),
    tanggal_mulai: dateSchema,
    tanggal_selesai: dateSchema,
    alasan: z
      .string()
      .trim()
      .min(10, "Alasan minimal 10 karakter")
      .max(2000, "Alasan maksimal 2000 karakter"),
    lokasi_tujuan: z.string().trim().max(150).optional(),
    keterangan_lokasi: z.string().trim().max(255).optional(),
  })
  .refine((value) => value.tanggal_selesai >= value.tanggal_mulai, {
    path: ["tanggal_selesai"],
    message: "Tanggal selesai tidak boleh sebelum tanggal mulai",
  });

const leaveTypeMutableFieldsSchema = z.object({
  nama: z.string().trim().min(1, "Nama wajib diisi").max(150),
  kuota_tahunan: z.coerce.number().int().min(0, "Kuota tidak boleh negatif"),
  kategori: z.enum(["cuti", "izin"]),
  maksimal_hari: z.coerce.number().int().min(0).max(365).optional(),
  memerlukan_dokumen: z.boolean(),
}).superRefine((value, context) => {
  if (value.kategori === "izin" && (!value.maksimal_hari || value.maksimal_hari < 1)) {
    context.addIssue({
      code: "custom",
      path: ["maksimal_hari"],
      message: "Maksimal hari wajib diisi untuk kategori izin",
    });
  }
});

export const createLeaveTypeFormSchema = z.object({
  kode: z.string().trim().min(1, "Kode wajib diisi").max(50),
  nama: z.string().trim().min(1, "Nama wajib diisi").max(150),
  kuota_tahunan: z.coerce.number().int().min(0, "Kuota tidak boleh negatif"),
  kategori: z.enum(["cuti", "izin"]),
  maksimal_hari: z.coerce.number().int().min(0).max(365).optional(),
  memerlukan_dokumen: z.boolean(),
}).superRefine((value, context) => {
  if (value.kategori === "izin" && (!value.maksimal_hari || value.maksimal_hari < 1)) {
    context.addIssue({
      code: "custom",
      path: ["maksimal_hari"],
      message: "Maksimal hari wajib diisi untuk kategori izin",
    });
  }
});

export const updateLeaveTypeFormSchema = leaveTypeMutableFieldsSchema;
