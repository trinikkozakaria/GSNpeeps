import { z } from "zod";

const uuidSchema = z.string().uuid();
const nullableUuidSchema = uuidSchema.nullable();
const dateSchema = z.string().regex(/^\d{4}-\d{2}-\d{2}$/);

export const departmentSchema = z.object({
  id: uuidSchema,
  nama: z.string(),
});

export const positionSchema = z.object({
  id: uuidSchema,
  department_id: uuidSchema,
  nama: z.string(),
});

export const employeeSummarySchema = z.object({
  id: uuidSchema,
  nip: z.string(),
  nama: z.string(),
  email: z.string(),
  departemen: z.string(),
  jabatan: z.string(),
  status: z.enum(["aktif", "nonaktif"]),
});

const employeeAddressSchema = z.object({
  jalan: z.string(),
  kelurahan: z.string().nullable(),
  kecamatan: z.string().nullable(),
  kota: z.string(),
  provinsi: z.string(),
});

const employeeKtpSchema = z.object({
  nomor_ktp: z.string(),
  file_url: z.string().nullable(),
});

const employeeContractSchema = z.object({
  nomor_kontrak: z.string(),
  tanggal_mulai: dateSchema,
  tanggal_berakhir: dateSchema,
  jenis_kontrak: z.enum(["PKWT", "PKWTT"]),
  file_url: z.string().nullable(),
  status: z.enum(["aktif", "berakhir"]),
});

export const employeeDetailSchema = employeeSummarySchema.extend({
  jenis_kelamin: z.enum(["L", "P"]),
  tanggal_lahir: dateSchema,
  tanggal_join: dateSchema,
  department_id: nullableUuidSchema,
  position_id: nullableUuidSchema,
  atasan_id: nullableUuidSchema,
  status_pernikahan: z.enum(["lajang", "menikah", "cerai"]).nullable(),
  alamat: employeeAddressSchema.nullable(),
  ktp: employeeKtpSchema.nullable(),
  kontrak: z.array(employeeContractSchema),
});

export const departmentListSchema = z.array(departmentSchema);
export const positionListSchema = z.array(positionSchema);

export const employeeListSchema = z.object({
  items: z.array(employeeSummarySchema),
  meta: z.object({
    page: z.number().int().positive(),
    limit: z.number().int().positive(),
    total_data: z.number().int().nonnegative(),
    total_page: z.number().int().nonnegative(),
  }),
});

export const updateEmployeeSchema = z.object({
  nama: z.string().trim().min(1, "Nama wajib diisi").max(150, "Nama maksimal 150 karakter"),
  email: z.string().trim().email("Format email tidak valid").max(150),
  jenis_kelamin: z.enum(["L", "P"], { message: "Jenis kelamin wajib dipilih" }),
  tanggal_lahir: dateSchema,
  tanggal_join: dateSchema,
  department_id: uuidSchema,
  position_id: uuidSchema,
  status_pernikahan: z.enum(["", "lajang", "menikah", "cerai"]),
  status: z.enum(["aktif", "nonaktif"]),
});

const optionalUuidFormSchema = z.union([uuidSchema, z.literal("")]);

export const createEmployeeSchema = z.object({
  nip: z.string().trim().min(1, "NIP wajib diisi").max(20, "NIP maksimal 20 karakter"),
  nama: z.string().trim().min(1, "Nama wajib diisi").max(150, "Nama maksimal 150 karakter"),
  email: z.string().trim().email("Format email tidak valid").max(150),
  jenis_kelamin: z.enum(["L", "P"], { message: "Jenis kelamin wajib dipilih" }),
  tanggal_lahir: dateSchema,
  tanggal_join: dateSchema,
  department_id: uuidSchema,
  position_id: uuidSchema,
  atasan_id: optionalUuidFormSchema,
  status_pernikahan: z.enum(["", "lajang", "menikah", "cerai"]),
  role: z.enum(["karyawan", "atasan", "hr", "top_management"]),
  alamat: z.object({
    jalan: z.string().trim().min(1, "Jalan wajib diisi").max(255),
    kelurahan: z.string().trim().max(100),
    kecamatan: z.string().trim().max(100),
    kota: z.string().trim().min(1, "Kota wajib diisi").max(100),
    provinsi: z.string().trim().min(1, "Provinsi wajib diisi").max(100),
  }),
  ktp: z.object({
    nomor_ktp: z.string().regex(/^\d{16}$/, "Nomor KTP harus tepat 16 digit"),
  }),
  kontrak: z.object({
    nomor_kontrak: z.string().trim().min(1, "Nomor kontrak wajib diisi").max(50),
    jenis_kontrak: z.enum(["PKWT", "PKWTT"]),
    tanggal_mulai: dateSchema,
    tanggal_berakhir: dateSchema,
  }),
}).refine(
  (value) => value.kontrak.tanggal_berakhir >= value.kontrak.tanggal_mulai,
  {
    path: ["kontrak", "tanggal_berakhir"],
    message: "Tanggal berakhir tidak boleh sebelum tanggal mulai",
  },
);
