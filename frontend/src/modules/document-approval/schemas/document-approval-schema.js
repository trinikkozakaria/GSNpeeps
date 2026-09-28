import { z } from "zod";

const uuidSchema = z.string().uuid();

// --- Workflow templates (§9.6/§9.7) ---------------------------------------------------

export const workflowStageSchema = z.object({
  id: uuidSchema.optional(),
  template_id: uuidSchema.optional(),
  urutan: z.number().int().nonnegative(),
  nama_tahap: z.string(),
  position_id: uuidSchema,
});

export const workflowTemplateSchema = z.object({
  id: uuidSchema,
  nama: z.string(),
  aktif: z.boolean(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const workflowTemplateListSchema = z.array(workflowTemplateSchema);

export const workflowTemplateDetailSchema = workflowTemplateSchema.extend({
  tahap: z.array(workflowStageSchema).default([]),
});

// --- Approval requests (§9.1-§9.5) -----------------------------------------------------

export const documentApprovalStatuses = [
  "berjalan",
  "disetujui",
  "ditolak",
  "dibatalkan",
  "bermasalah",
];

export const approvalRequestSchema = z.object({
  id: uuidSchema,
  template_id: uuidSchema,
  document_url: z.string(),
  document_url_normalized: z.string(),
  judul: z.string(),
  pemohon_user_id: uuidSchema,
  stage_index: z.number().int().nonnegative(),
  status: z.enum(documentApprovalStatuses),
  created_at: z.string(),
  updated_at: z.string(),
});

const paginationSchema = z.object({
  page: z.number().int().positive(),
  limit: z.number().int().positive(),
  total_data: z.number().int().nonnegative(),
  total_page: z.number().int().nonnegative(),
});

export const approvalRequestListSchema = z.object({
  items: z.array(approvalRequestSchema),
  meta: paginationSchema,
});

export const stageApproverSchema = z.object({
  user_id: uuidSchema,
  nama: z.string(),
});

export const stageProgressSchema = z.object({
  urutan: z.number().int().nonnegative(),
  nama_tahap: z.string(),
  position_id: uuidSchema,
  position_nama: z.string().nullable().optional(),
  pemegang_posisi: z.array(stageApproverSchema).default([]),
  selesai: z.boolean(),
  aktif: z.boolean(),
});

export const approvalDecisionSchema = z.object({
  urutan_tahap: z.number().int().nonnegative(),
  approver_user_id: uuidSchema,
  approver_nama: z.string(),
  keputusan: z.enum(["approve", "reject"]),
  catatan: z.string().nullable().optional(),
  decided_at: z.string(),
});

export const approvalRequestDetailSchema = approvalRequestSchema.extend({
  template_nama: z.string(),
  tahap: z.array(stageProgressSchema).default([]),
  riwayat: z.array(approvalDecisionSchema).default([]),
});

// --- Forms -------------------------------------------------------------------------

export const createWorkflowTemplateFormSchema = z.object({
  nama: z.string().trim().min(3, "Nama minimal 3 karakter").max(150),
  aktif: z.boolean(),
  tahap: z
    .array(
      z.object({
        nama_tahap: z.string().trim().min(3, "Nama tahap minimal 3 karakter").max(150),
        position_id: uuidSchema.or(z.literal("")).refine((value) => value !== "", {
          message: "Posisi wajib dipilih",
        }),
      }),
    )
    .min(1, "Alur wajib memiliki minimal 1 tahap"),
});

export const submitApprovalFormSchema = z.object({
  document_url: z.string().trim().url("URL dokumen tidak valid"),
  judul: z.string().trim().min(3, "Judul minimal 3 karakter").max(255),
  template_id: uuidSchema.or(z.literal("")).refine((value) => value !== "", {
    message: "Alur persetujuan wajib dipilih",
  }),
});

export const decisionFormSchema = z
  .object({
    keputusan: z.enum(["approve", "reject"]),
    catatan: z.string().trim().max(2000).optional().default(""),
  })
  .refine((value) => value.keputusan !== "reject" || value.catatan.length >= 5, {
    path: ["catatan"],
    message: "Catatan wajib diisi minimal 5 karakter saat menolak",
  });
