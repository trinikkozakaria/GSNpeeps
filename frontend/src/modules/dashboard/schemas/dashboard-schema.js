import { z } from "zod";

export const dashboardPeriods = ["harian", "mingguan", "bulanan", "tahunan"];

const namedCountSchema = z.object({
  nama: z.string(),
  jumlah: z.number().int().nonnegative(),
});

export const genderCategories = ["laki_laki", "perempuan", "belum_diisi"];

const genderCountSchema = z.object({
  kategori: z.enum(genderCategories),
  jumlah: z.number().int().nonnegative(),
});

// Org chart bersifat rekursif; kedalaman ditentukan relasi atasan_id dari backend.
const organizationNodeSchema = z.object({
  employee_id: z.string().uuid(),
  nama: z.string(),
  departemen: z.string(),
  jabatan: z.string(),
  get bawahan() {
    return z.array(organizationNodeSchema);
  },
});

const periodRangeSchema = z.object({
  tipe: z.enum(dashboardPeriods),
  tanggal_mulai: z.string().regex(/^\d{4}-\d{2}-\d{2}$/),
  tanggal_selesai: z.string().regex(/^\d{4}-\d{2}-\d{2}$/),
  timezone: z.string(),
});

export const dashboardMetricsSchema = z.object({
  periode: periodRangeSchema,
  total_karyawan: z.number().int().nonnegative(),
  karyawan_aktif: z.number().int().nonnegative(),
  karyawan_nonaktif: z.number().int().nonnegative(),
  karyawan_baru: z.number().int().nonnegative(),
  resign: z.number().int().nonnegative(),
  turnover_rate: z.number().nonnegative(),
  hadir_valid: z.number().int().nonnegative(),
  terlambat: z.number().int().nonnegative(),
  hari_izin_disetujui: z.number().int().nonnegative(),
  estimasi_biaya_payroll: z.number().nonnegative(),
  pengajuan_menunggu: z.number().int().nonnegative(),
  komposisi_departemen_aktif: z.array(namedCountSchema),
  komposisi_departemen_nonaktif: z.array(namedCountSchema),
  rasio_gender: z.array(genderCountSchema),
  organization_chart: z.array(organizationNodeSchema),
});
