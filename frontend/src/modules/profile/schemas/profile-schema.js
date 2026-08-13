import { z } from "zod";

import { employeeDetailSchema } from "../../employees/schemas/employee-schema";

// Profil Saya memakai schema EmployeeDetail yang sama dengan detail karyawan; endpoint
// hanya berbeda pada resolusi identitas di sisi backend.
export const myProfileSchema = employeeDetailSchema;

const clockHistorySchema = z.object({
  tanggal: z.string(),
  check_in: z.string().nullable(),
  check_out: z.string().nullable(),
  status: z.string(),
});

export const personalMetricsSchema = z.object({
  periode: z.string().regex(/^\d{4}-(0[1-9]|1[0-2])$/),
  hadir: z.number().int().nonnegative(),
  terlambat: z.number().int().nonnegative(),
  izin: z.number().int().nonnegative(),
  total_lembur_jam: z.number().nonnegative(),
  riwayat_absensi: z.array(clockHistorySchema),
});
