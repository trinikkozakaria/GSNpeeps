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

// Bagian berikut mengikuti schema EmployeeDetail pada OpenAPI. Field yang tidak termasuk
// `required` boleh tidak dikirim backend, sehingga dimodelkan optional dan bukan ditebak.
const employeeBpjsSchema = z.object({
  jenis: z.enum(["kesehatan", "ketenagakerjaan"]),
  nomor: z.string(),
  status: z.enum(["aktif", "nonaktif"]).optional(),
});

const employeeNpwpSchema = z.object({
  nomor_npwp: z.string(),
  status_ptkp: z.string().optional(),
  file_url: z.string().nullable().optional(),
});

const emergencyContactSchema = z.object({
  nama: z.string(),
  hubungan: z.string().nullable().optional(),
  nomor_telepon: z.string(),
  is_primary: z.boolean().optional(),
});

const educationHistorySchema = z.object({
  jenjang: z.string().nullable().optional(),
  institusi: z.string().nullable().optional(),
  jurusan: z.string().nullable().optional(),
  tahun_lulus: z.number().int().nullable().optional(),
});

const positionHistorySchema = z.object({
  departemen: departmentSchema.optional(),
  jabatan: positionSchema.optional(),
  tanggal_mulai: dateSchema,
  tanggal_selesai: dateSchema.nullable(),
});

const currentSalarySchema = z.object({
  periode: z.string().regex(/^\d{4}-(0[1-9]|1[0-2])$/),
  gaji_pokok: z.number(),
  tunjangan: z.number().optional(),
  potongan: z.number().optional(),
  take_home_pay: z.number().optional(),
});

export const employeeDocumentSchema = z.object({
  id: uuidSchema,
  jenis_dokumen: z.string(),
  nama_file: z.string(),
  file_url: z.string(),
  created_at: z.string(),
});

export const employeeDocumentListSchema = z.array(employeeDocumentSchema);

export const employeeDetailSchema = employeeSummarySchema.extend({
  foto_profil_url: z.string().nullable().optional(),
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
  bpjs: z.array(employeeBpjsSchema).optional().default([]),
  npwp: employeeNpwpSchema.optional(),
  kontak_darurat: z.array(emergencyContactSchema).optional().default([]),
  pendidikan: z.array(educationHistorySchema).optional().default([]),
  riwayat_jabatan: z.array(positionHistorySchema).optional().default([]),
  gaji_berjalan: currentSalarySchema.optional(),
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

// Bagian berikut adalah bentuk form untuk create/edit karyawan (Input schema OpenAPI
// 0.6.0), bukan bentuk read (lihat employeeDetailSchema di atas). BPJS dan NPWP satu baris
// per employee; kontak darurat/pendidikan/riwayat jabatan memakai semantik replace-all
// sehingga selalu dikirim sebagai daftar lengkap yang berlaku.
export const bpjsFormSchema = z.object({
  nomor_kesehatan: z.string().trim().max(20, "Maksimal 20 karakter"),
  nomor_ketenagakerjaan: z.string().trim().max(20, "Maksimal 20 karakter"),
});

export const npwpFormSchema = z.object({
  nomor_npwp: z.string().trim().max(25, "Maksimal 25 karakter"),
});

export const emergencyContactFormSchema = z.object({
  nama: z.string().trim().min(1, "Nama wajib diisi").max(150),
  hubungan: z.string().trim().max(50),
  nomor_telepon: z.string().trim().min(1, "Nomor telepon wajib diisi").max(20),
});

export const educationFormSchema = z.object({
  jenjang: z.string().trim().max(20),
  institusi: z.string().trim().max(150),
  tahun_lulus: z.union([
    z.string().trim().regex(/^\d{4}$/, "Tahun tidak valid"),
    z.literal(""),
  ]),
});

export const positionHistoryFormSchema = z
  .object({
    department_id: z.union([uuidSchema, z.literal("")]),
    position_id: z.union([uuidSchema, z.literal("")]),
    tanggal_mulai: dateSchema,
    tanggal_selesai: z.union([dateSchema, z.literal("")]),
  })
  .refine((value) => value.tanggal_selesai === "" || value.tanggal_selesai >= value.tanggal_mulai, {
    path: ["tanggal_selesai"],
    message: "Tanggal selesai tidak boleh sebelum tanggal mulai",
  });

export const currentSalaryFormSchema = z.object({
  periode: z.string().regex(/^\d{4}-(0[1-9]|1[0-2])$/, "Periode wajib diisi (YYYY-MM)"),
  gaji_pokok: z.union([z.string().trim().min(1, "Gaji pokok wajib diisi"), z.literal("")]),
  tunjangan: z.string().trim(),
  potongan: z.string().trim(),
});

const employeeDetailFormFields = {
  bpjs: bpjsFormSchema,
  npwp: npwpFormSchema,
  kontak_darurat: z.array(emergencyContactFormSchema),
  pendidikan: z.array(educationFormSchema),
  riwayat_jabatan: z.array(positionHistoryFormSchema),
  gaji_berjalan: currentSalaryFormSchema,
};

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
  ...employeeDetailFormFields,
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
  ...employeeDetailFormFields,
}).refine(
  (value) => value.kontrak.tanggal_berakhir >= value.kontrak.tanggal_mulai,
  {
    path: ["kontrak", "tanggal_berakhir"],
    message: "Tanggal berakhir tidak boleh sebelum tanggal mulai",
  },
);

// Merangkai bagian detail form (bpjs/npwp/kontak_darurat/pendidikan/riwayat_jabatan/
// gaji_berjalan) menjadi bentuk payload API. Objek tunggal (bpjs/npwp/gaji_berjalan)
// dihilangkan sepenuhnya dari payload ketika kosong agar bagian yang tidak disentuh HR
// tidak memicu penulisan baris kosong di server; baris array yang kosong seluruhnya juga
// diabaikan. Lihat OpenAPI 0.6.0 / D-036 untuk semantik replace-all pada array.
export const buildEmployeeDetailPayload = (values) => {
  const payload = {};

  const bpjsHealth = values.bpjs.nomor_kesehatan.trim();
  const bpjsEmployment = values.bpjs.nomor_ketenagakerjaan.trim();
  if (bpjsHealth || bpjsEmployment) {
    payload.bpjs = {
      nomor_kesehatan: bpjsHealth || null,
      nomor_ketenagakerjaan: bpjsEmployment || null,
    };
  }

  const npwpNumber = values.npwp.nomor_npwp.trim();
  if (npwpNumber) {
    payload.npwp = { nomor_npwp: npwpNumber };
  }

  payload.kontak_darurat = values.kontak_darurat
    .map((contact) => ({
      nama: contact.nama.trim(),
      hubungan: contact.hubungan.trim() || null,
      nomor_telepon: contact.nomor_telepon.trim(),
    }))
    .filter((contact) => contact.nama && contact.nomor_telepon);

  payload.pendidikan = values.pendidikan
    .map((entry) => ({
      jenjang: entry.jenjang.trim() || null,
      institusi: entry.institusi.trim() || null,
      tahun_lulus: entry.tahun_lulus ? Number(entry.tahun_lulus) : null,
    }))
    .filter((entry) => entry.jenjang || entry.institusi || entry.tahun_lulus);

  payload.riwayat_jabatan = values.riwayat_jabatan
    .filter((entry) => entry.tanggal_mulai)
    .map((entry) => ({
      department_id: entry.department_id || null,
      position_id: entry.position_id || null,
      tanggal_mulai: entry.tanggal_mulai,
      tanggal_selesai: entry.tanggal_selesai || null,
    }));

  const gajiPokok = values.gaji_berjalan.gaji_pokok.trim();
  if (values.gaji_berjalan.periode && gajiPokok) {
    payload.gaji_berjalan = {
      periode: values.gaji_berjalan.periode,
      gaji_pokok: Number(gajiPokok),
      tunjangan: values.gaji_berjalan.tunjangan.trim() ? Number(values.gaji_berjalan.tunjangan) : 0,
      potongan: values.gaji_berjalan.potongan.trim() ? Number(values.gaji_berjalan.potongan) : 0,
    };
  }

  return payload;
};

export const emptyEmployeeDetailDefaults = {
  bpjs: { nomor_kesehatan: "", nomor_ketenagakerjaan: "" },
  npwp: { nomor_npwp: "" },
  kontak_darurat: [],
  pendidikan: [],
  riwayat_jabatan: [],
  gaji_berjalan: { periode: "", gaji_pokok: "", tunjangan: "", potongan: "" },
};

// Kebalikan buildEmployeeDetailPayload: mengubah bentuk read (EmployeeDetail) menjadi
// default form edit sehingga HR melihat data yang sudah tersimpan sebelum mengubahnya.
export const mapEmployeeDetailToFormDefaults = (detail) => {
  const bpjsKesehatan = detail.bpjs?.find((item) => item.jenis === "kesehatan");
  const bpjsKetenagakerjaan = detail.bpjs?.find((item) => item.jenis === "ketenagakerjaan");
  return {
    bpjs: {
      nomor_kesehatan: bpjsKesehatan?.nomor ?? "",
      nomor_ketenagakerjaan: bpjsKetenagakerjaan?.nomor ?? "",
    },
    npwp: { nomor_npwp: detail.npwp?.nomor_npwp ?? "" },
    kontak_darurat: (detail.kontak_darurat ?? []).map((contact) => ({
      nama: contact.nama,
      hubungan: contact.hubungan ?? "",
      nomor_telepon: contact.nomor_telepon,
    })),
    pendidikan: (detail.pendidikan ?? []).map((entry) => ({
      jenjang: entry.jenjang ?? "",
      institusi: entry.institusi ?? "",
      tahun_lulus: entry.tahun_lulus ? String(entry.tahun_lulus) : "",
    })),
    riwayat_jabatan: (detail.riwayat_jabatan ?? []).map((entry) => ({
      department_id: entry.departemen?.id ?? "",
      position_id: entry.jabatan?.id ?? "",
      tanggal_mulai: entry.tanggal_mulai,
      tanggal_selesai: entry.tanggal_selesai ?? "",
    })),
    gaji_berjalan: {
      periode: detail.gaji_berjalan?.periode ?? "",
      gaji_pokok: detail.gaji_berjalan ? String(detail.gaji_berjalan.gaji_pokok) : "",
      tunjangan: detail.gaji_berjalan ? String(detail.gaji_berjalan.tunjangan) : "",
      potongan: detail.gaji_berjalan ? String(detail.gaji_berjalan.potongan) : "",
    },
  };
};
