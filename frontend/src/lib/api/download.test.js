import { describe, expect, it } from "vitest";

import { __testing } from "./download";

const { filenameFromDisposition } = __testing;

describe("filenameFromDisposition", () => {
  it("uses the fallback when the header is missing", () => {
    expect(filenameFromDisposition(null, "karyawan.xlsx")).toBe("karyawan.xlsx");
    expect(filenameFromDisposition("", "karyawan.xlsx")).toBe("karyawan.xlsx");
  });

  it("reads a quoted filename", () => {
    expect(
      filenameFromDisposition('attachment; filename="karyawan-20260801.xlsx"', "fallback.xlsx"),
    ).toBe("karyawan-20260801.xlsx");
  });

  it("reads an RFC 5987 encoded filename", () => {
    expect(
      filenameFromDisposition("attachment; filename*=UTF-8''laporan%20karyawan.pdf", "fallback.pdf"),
    ).toBe("laporan-karyawan.pdf");
  });

  it("strips path components supplied by the server", () => {
    expect(
      filenameFromDisposition('attachment; filename="../../etc/passwd"', "fallback.xlsx"),
    ).toBe("passwd");
    expect(
      filenameFromDisposition('attachment; filename="C:\\Windows\\system.ini"', "fallback.xlsx"),
    ).toBe("system.ini");
  });

  it("falls back when the header yields no safe characters", () => {
    expect(filenameFromDisposition('attachment; filename="///"', "karyawan.xlsx")).toBe(
      "karyawan.xlsx",
    );
  });
});
