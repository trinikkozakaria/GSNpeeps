import { z } from "zod";

const uuidSchema = z.string().uuid();

export const workModes = ["WFO", "WFH", "WFA"];
export const attendanceTypes = ["check_in", "check_out"];

export const officeLocationSchema = z.object({
  id: uuidSchema,
  kode: z.string(),
  nama: z.string(),
  alamat: z.string().nullable().optional(),
  latitude: z.number(),
  longitude: z.number(),
  is_active: z.boolean(),
});

export const officeLocationListSchema = z.array(officeLocationSchema);

export const attendanceSchema = z.object({
  id: uuidSchema,
  employee_id: uuidSchema,
  tanggal: z.string(),
  tipe: z.enum(attendanceTypes),
  mode_kerja: z.enum(workModes),
  waktu: z.string(),
  gps_lat: z.number().optional(),
  gps_long: z.number().optional(),
  office_location_id: uuidSchema.nullable().optional(),
  distance_meters: z.number().nullable().optional(),
  foto_url: z.string().nullable().optional(),
  status: z.enum(["tepat_waktu", "terlambat", "pulang_cepat", "valid"]),
});

export const liveFeedItemSchema = attendanceSchema.extend({
  nama_karyawan: z.string(),
  departemen: z.string().optional(),
});

export const liveFeedListSchema = z.array(liveFeedItemSchema);

export const attendanceReportItemSchema = z.object({
  employee_id: uuidSchema,
  nama_karyawan: z.string(),
  departemen: z.string().optional(),
  hadir: z.number().int().nonnegative(),
  terlambat: z.number().int().nonnegative(),
  izin: z.number().int().nonnegative(),
  alpha: z.number().int().nonnegative(),
  total_jam_kerja: z.number().optional(),
});

export const attendanceReportSchema = z.object({
  items: z.array(attendanceReportItemSchema),
  meta: z.object({
    page: z.number().int().positive(),
    limit: z.number().int().positive(),
    total_data: z.number().int().nonnegative(),
    total_page: z.number().int().nonnegative(),
  }),
});

// Batas berkas foto absensi sesuai kontrak.
export const maxPhotoBytes = 5 * 1024 * 1024;
const allowedPhotoExtensions = [".jpg", ".jpeg", ".png"];

export const validatePhotoFile = (file) => {
  if (!file) return "Foto absensi wajib diambil atau diunggah.";
  const extension = file.name.slice(file.name.lastIndexOf(".")).toLowerCase();
  if (!file.name.includes(".") || !allowedPhotoExtensions.includes(extension)) {
    return "Format foto tidak didukung. Gunakan JPG atau PNG.";
  }
  if (file.size > maxPhotoBytes) return "Ukuran foto melebihi batas 5 MB.";
  if (file.size === 0) return "Berkas foto kosong.";
  return "";
};
