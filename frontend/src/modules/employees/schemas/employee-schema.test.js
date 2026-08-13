import { describe, expect, it } from "vitest";

import {
  createEmployeeSchema,
  employeeDetailSchema,
  employeeListSchema,
} from "./employee-schema";

const id = "7e2b4e5e-8c31-44c8-b465-91816ae41f2a";

describe("employee schemas", () => {
  it("accepts a paginated employee response", () => {
    const result = employeeListSchema.parse({
      items: [{
        id,
        nip: "GSN-001",
        nama: "Ayu Lestari",
        email: "ayu@example.test",
        departemen: "Human Resources",
        jabatan: "HR Officer",
        status: "aktif",
      }],
      meta: { page: 1, limit: 10, total_data: 1, total_page: 1 },
    });

    expect(result.items).toHaveLength(1);
  });

  it("requires gender and date-only values on detail", () => {
    expect(() => employeeDetailSchema.parse({
      id,
      nip: "GSN-001",
      nama: "Ayu Lestari",
      email: "ayu@example.test",
      departemen: "Human Resources",
      jabatan: "HR Officer",
      status: "aktif",
      jenis_kelamin: "Unknown",
      tanggal_lahir: "1995-01-01T00:00:00Z",
    })).toThrow();
  });

  it("validates the complete create employee contract", () => {
    const valid = {
      nip: "EMP-001",
      nama: "Karyawan Uji",
      email: "employee@example.test",
      jenis_kelamin: "P",
      tanggal_lahir: "1995-04-10",
      tanggal_join: "2026-07-29",
      department_id: id,
      position_id: "68dc89ae-bc5e-4d80-bcf1-e1fb88447dcb",
      atasan_id: "",
      status_pernikahan: "",
      role: "karyawan",
      alamat: {
        jalan: "Jalan Sintetis",
        kelurahan: "",
        kecamatan: "",
        kota: "Jakarta",
        provinsi: "DKI Jakarta",
      },
      ktp: { nomor_ktp: "3174000000000001" },
      kontrak: {
        nomor_kontrak: "PKWT-TEST-001",
        jenis_kontrak: "PKWT",
        tanggal_mulai: "2026-07-29",
        tanggal_berakhir: "2027-07-28",
      },
    };

    expect(createEmployeeSchema.parse(valid).nip).toBe("EMP-001");
    expect(() => createEmployeeSchema.parse({
      ...valid,
      kontrak: {
        ...valid.kontrak,
        tanggal_berakhir: "2025-01-01",
      },
    })).toThrow();
  });
});
