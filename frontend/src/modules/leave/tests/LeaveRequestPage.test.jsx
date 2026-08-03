import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { LeaveRequestPage } from "../pages/LeaveRequestPage";
import {
  createOvertimeFormSchema,
} from "../../overtime/schemas/overtime-schema";
import { validateSupportingDocument } from "../schemas/leave-schema";

const { createMock, typesState } = vi.hoisted(() => ({
  createMock: vi.fn(),
  typesState: { current: { data: [], isError: false } },
}));

vi.mock("../hooks/useLeave", () => ({
  useCreateLeaveRequest: () => ({ mutateAsync: createMock, isPending: false }),
  useLeaveTypes: () => typesState.current,
}));

const annualLeave = {
  id: "11111111-1111-4111-8111-111111111111",
  kode: "CT",
  nama: "Cuti Tahunan",
  kuota_tahunan: 12,
  memerlukan_dokumen: false,
  is_active: true,
};

const travelLeave = {
  id: "22222222-2222-4222-8222-222222222222",
  kode: "PD",
  nama: "Perjalanan Dinas",
  kuota_tahunan: 0,
  memerlukan_dokumen: true,
  is_active: true,
};

const documentFile = (name = "surat.pdf", size = 2048) => {
  const file = new File(["x"], name, { type: "application/pdf" });
  Object.defineProperty(file, "size", { value: size });
  return file;
};

const fillBasics = async (user) => {
  await user.type(screen.getByLabelText("Tanggal mulai"), "2026-08-10");
  await user.type(screen.getByLabelText("Tanggal selesai"), "2026-08-12");
  await user.type(screen.getByLabelText("Alasan"), "Keperluan keluarga sintetis");
};

describe("validateSupportingDocument", () => {
  it("requires a document only when the leave type demands it", () => {
    expect(validateSupportingDocument(null, true)).toMatch(/wajib diunggah/i);
    expect(validateSupportingDocument(null, false)).toBe("");
  });

  it("rejects unsupported formats and oversize files", () => {
    expect(validateSupportingDocument(documentFile("arsip.zip"), false)).toMatch(/tidak didukung/i);
    expect(validateSupportingDocument(documentFile("besar.pdf", 6 * 1024 * 1024), false)).toMatch(/5 MB/);
    expect(validateSupportingDocument(documentFile(), false)).toBe("");
  });
});

describe("createOvertimeFormSchema", () => {
  it("requires the end time to be after the start time", () => {
    const base = { tanggal: "2026-08-10", alasan: "Penyelesaian rilis sintetis" };

    expect(
      createOvertimeFormSchema.safeParse({ ...base, waktu_mulai: "18:00", waktu_selesai: "20:00" })
        .success,
    ).toBe(true);
    expect(
      createOvertimeFormSchema.safeParse({ ...base, waktu_mulai: "20:00", waktu_selesai: "18:00" })
        .success,
    ).toBe(false);
    expect(
      createOvertimeFormSchema.safeParse({ ...base, waktu_mulai: "18:00", waktu_selesai: "18:00" })
        .success,
    ).toBe(false);
  });
});

describe("LeaveRequestPage", () => {
  beforeEach(() => {
    createMock.mockReset();
    createMock.mockResolvedValue({ id: "req-1", status: "menunggu_atasan" });
    typesState.current = { data: [annualLeave, travelLeave], isError: false };
  });

  it("hides travel fields until a travel leave type is selected", async () => {
    const user = userEvent.setup();
    render(<LeaveRequestPage />);

    expect(screen.queryByLabelText("Lokasi tujuan")).not.toBeInTheDocument();

    await user.selectOptions(screen.getByLabelText("Jenis izin"), travelLeave.id);

    expect(await screen.findByLabelText("Lokasi tujuan")).toBeInTheDocument();
    expect(screen.getByLabelText("Keperluan tugas")).toBeInTheDocument();
  });

  it("requires travel fields for Perjalanan Dinas", async () => {
    const user = userEvent.setup();
    render(<LeaveRequestPage />);

    await user.selectOptions(screen.getByLabelText("Jenis izin"), travelLeave.id);
    await fillBasics(user);
    await user.upload(screen.getByLabelText(/dokumen pendukung/i), documentFile());
    await user.click(screen.getByRole("button", { name: "Kirim pengajuan" }));

    expect(await screen.findByText(/lokasi tujuan wajib diisi/i)).toBeInTheDocument();
    expect(createMock).not.toHaveBeenCalled();
  });

  // Kewajiban dokumen berasal dari master jenis izin, bukan ditebak client.
  it("requires a document when the leave type demands one", async () => {
    const user = userEvent.setup();
    render(<LeaveRequestPage />);

    await user.selectOptions(screen.getByLabelText("Jenis izin"), travelLeave.id);
    await fillBasics(user);
    await user.type(screen.getByLabelText("Lokasi tujuan"), "Surabaya");
    await user.type(screen.getByLabelText("Keperluan tugas"), "Audit cabang");
    await user.click(screen.getByRole("button", { name: "Kirim pengajuan" }));

    expect(await screen.findByText(/wajib diunggah untuk jenis izin ini/i)).toBeInTheDocument();
    expect(createMock).not.toHaveBeenCalled();
  });

  it("submits without a document when the leave type does not require one", async () => {
    const user = userEvent.setup();
    render(<LeaveRequestPage />);

    await user.selectOptions(screen.getByLabelText("Jenis izin"), annualLeave.id);
    await fillBasics(user);
    await user.click(screen.getByRole("button", { name: "Kirim pengajuan" }));

    await waitFor(() => expect(createMock).toHaveBeenCalledTimes(1));
    const payload = createMock.mock.calls[0][0];
    expect(payload.jenis_izin_id).toBe(annualLeave.id);
    expect(payload.dokumen_pendukung).toBeNull();
    expect(payload.lokasi_tujuan).toBeUndefined();
  });

  it("rejects an end date before the start date", async () => {
    const user = userEvent.setup();
    render(<LeaveRequestPage />);

    await user.selectOptions(screen.getByLabelText("Jenis izin"), annualLeave.id);
    await user.type(screen.getByLabelText("Tanggal mulai"), "2026-08-12");
    await user.type(screen.getByLabelText("Tanggal selesai"), "2026-08-10");
    await user.type(screen.getByLabelText("Alasan"), "Keperluan keluarga sintetis");
    await user.click(screen.getByRole("button", { name: "Kirim pengajuan" }));

    expect(await screen.findByText(/tidak boleh sebelum tanggal mulai/i)).toBeInTheDocument();
    expect(createMock).not.toHaveBeenCalled();
  });

  it("explains an insufficient balance without inventing an approved status", async () => {
    createMock.mockRejectedValue({ code: "INSUFFICIENT_LEAVE_BALANCE", message: "raw" });
    const user = userEvent.setup();
    render(<LeaveRequestPage />);

    await user.selectOptions(screen.getByLabelText("Jenis izin"), annualLeave.id);
    await fillBasics(user);
    await user.click(screen.getByRole("button", { name: "Kirim pengajuan" }));

    expect(await screen.findByText(/saldo atau kuota cuti tidak mencukupi/i)).toBeInTheDocument();
    expect(screen.queryByText(/menunggu keputusan approver/i)).not.toBeInTheDocument();
  });

  it("maps server field errors onto the form", async () => {
    createMock.mockRejectedValue({
      status: 422,
      fields: { alasan: "Alasan terlalu singkat menurut server" },
      message: "Data belum valid",
    });
    const user = userEvent.setup();
    render(<LeaveRequestPage />);

    await user.selectOptions(screen.getByLabelText("Jenis izin"), annualLeave.id);
    await fillBasics(user);
    await user.click(screen.getByRole("button", { name: "Kirim pengajuan" }));

    expect(await screen.findByText("Alasan terlalu singkat menurut server")).toBeInTheDocument();
  });

  it("explains when the leave type master cannot be read", () => {
    typesState.current = { data: undefined, isError: true, error: { status: 403 } };
    render(<LeaveRequestPage />);

    expect(screen.getByRole("alert")).toHaveTextContent(/hubungi hr/i);
  });
});
