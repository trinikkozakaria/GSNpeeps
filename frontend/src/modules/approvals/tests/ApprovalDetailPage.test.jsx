import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApprovalDetailPage } from "../pages/ApprovalDetailPage";

const { authState, leaveState, decideMock, delegateMock, decideOvertimeMock, refetchMock } =
  vi.hoisted(() => ({
    authState: { current: { role: "atasan", user: { id: "user-1" } } },
    leaveState: { current: {} },
    decideMock: vi.fn(),
    delegateMock: vi.fn(),
    decideOvertimeMock: vi.fn(),
    refetchMock: vi.fn(),
  }));

vi.mock("../../auth/hooks/useAuth", () => ({ useAuth: () => authState.current }));

vi.mock("../../leave/hooks/useLeave", () => ({
  useLeaveRequestDetail: () => leaveState.current,
  useDecideLeaveRequest: () => ({ mutateAsync: decideMock, isPending: false }),
  useDelegateLeaveRequest: () => ({ mutateAsync: delegateMock, isPending: false }),
}));

vi.mock("../../overtime/hooks/useOvertime", () => ({
  useOvertimeDetail: () => ({ data: undefined, isPending: false, isError: false }),
  useDecideOvertimeRequest: () => ({ mutateAsync: decideOvertimeMock, isPending: false }),
}));

const leaveDetail = (status = "menunggu_atasan") => ({
  id: "11111111-1111-4111-8111-111111111111",
  employee_id: "22222222-2222-4222-8222-222222222222",
  nama_karyawan: "Karyawan Uji",
  jenis_izin: "Cuti Tahunan",
  tanggal_mulai: "2026-08-10",
  tanggal_selesai: "2026-08-12",
  jumlah_hari: 3,
  status,
  created_at: "2026-08-03T02:00:00Z",
  alasan: "Keperluan keluarga sintetis",
  dokumen_url: null,
  approval_history: [],
});

const renderPage = () =>
  render(
    <MemoryRouter initialEntries={["/app/persetujuan/ketidakhadiran/req-1"]}>
      <Routes>
        <Route
          path="/app/persetujuan/ketidakhadiran/:id"
          element={<ApprovalDetailPage kind="ketidakhadiran" />}
        />
      </Routes>
    </MemoryRouter>,
  );

describe("ApprovalDetailPage", () => {
  beforeEach(() => {
    decideMock.mockReset();
    decideMock.mockResolvedValue({ id: "req-1", status: "menunggu_hr" });
    delegateMock.mockReset();
    delegateMock.mockResolvedValue({ id: "req-1", status: "menunggu_hr" });
    refetchMock.mockReset();
    authState.current = { role: "atasan", user: { id: "user-1" } };
    leaveState.current = {
      data: leaveDetail(),
      isPending: false,
      isError: false,
      refetch: refetchMock,
    };
  });

  it("gives the supervisor decision and delegation controls on their stage", () => {
    renderPage();

    expect(screen.getByRole("button", { name: "Setujui" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Tolak" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Delegasikan ke HR" })).toBeInTheDocument();
  });

  // HR memutus pada tahapnya sendiri dan tidak memiliki aksi delegasi.
  it("hides delegation from HR", () => {
    authState.current = { role: "hr", user: { id: "user-2" } };
    leaveState.current = {
      data: leaveDetail("menunggu_hr"),
      isPending: false,
      isError: false,
      refetch: refetchMock,
    };
    renderPage();

    expect(screen.getByRole("button", { name: "Setujui" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Delegasikan ke HR" })).not.toBeInTheDocument();
  });

  it("hides decision controls when the request sits on another stage", () => {
    leaveState.current = {
      data: leaveDetail("menunggu_hr"),
      isPending: false,
      isError: false,
      refetch: refetchMock,
    };
    renderPage();

    expect(screen.queryByRole("button", { name: "Setujui" })).not.toBeInTheDocument();
    expect(screen.getByText(/tahap approver lain/i)).toBeInTheDocument();
  });

  it("hides decision controls once the request is final", () => {
    leaveState.current = {
      data: leaveDetail("disetujui"),
      isPending: false,
      isError: false,
      refetch: refetchMock,
    };
    renderPage();

    expect(screen.queryByRole("button", { name: "Setujui" })).not.toBeInTheDocument();
    expect(screen.getByText(/sudah final/i)).toBeInTheDocument();
  });

  // Penolakan wajib memiliki catatan minimal lima karakter.
  it("requires a note before rejecting", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "Tolak" }));
    await user.click(screen.getByRole("button", { name: "Konfirmasi" }));

    expect(await screen.findByText(/catatan wajib diisi minimal 5 karakter/i)).toBeInTheDocument();
    expect(decideMock).not.toHaveBeenCalled();

    await user.type(screen.getByLabelText(/catatan/i), "Beban kerja tinggi");
    await user.click(screen.getByRole("button", { name: "Konfirmasi" }));

    await waitFor(() => expect(decideMock).toHaveBeenCalledTimes(1));
    expect(decideMock).toHaveBeenCalledWith({
      keputusan: "tolak",
      catatan: "Beban kerja tinggi",
    });
  });

  it("approves without requiring a note", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "Setujui" }));
    await user.click(screen.getByRole("button", { name: "Konfirmasi" }));

    await waitFor(() => expect(decideMock).toHaveBeenCalledTimes(1));
    expect(decideMock).toHaveBeenCalledWith({ keputusan: "setujui", catatan: undefined });
  });

  it("requires a note before delegating", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "Delegasikan ke HR" }));
    await user.click(screen.getByRole("button", { name: "Konfirmasi" }));

    expect(await screen.findByText(/catatan wajib diisi minimal 5 karakter/i)).toBeInTheDocument();
    expect(delegateMock).not.toHaveBeenCalled();
  });

  // 409 berarti pihak lain sudah memutus; detail dimuat ulang, bukan ditimpa optimistis.
  it("refreshes the detail when the server reports ALREADY_DECIDED", async () => {
    decideMock.mockRejectedValue({ status: 409, code: "ALREADY_DECIDED", message: "raw" });
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "Setujui" }));
    await user.click(screen.getByRole("button", { name: "Konfirmasi" }));

    expect(await screen.findByText(/sudah diproses oleh pihak lain/i)).toBeInTheDocument();
    expect(refetchMock).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("keeps the dialog open and shows the reason on other failures", async () => {
    decideMock.mockRejectedValue({ status: 500, message: "Layanan tidak tersedia." });
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "Setujui" }));
    await user.click(screen.getByRole("button", { name: "Konfirmasi" }));

    expect(await screen.findByText("Layanan tidak tersedia.")).toBeInTheDocument();
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("does not leak existence when the request cannot be read", () => {
    leaveState.current = {
      data: undefined,
      isPending: false,
      isError: true,
      error: { status: 403 },
      refetch: refetchMock,
    };
    renderPage();

    expect(screen.getByRole("alert")).toHaveTextContent(
      /tidak ditemukan atau tidak dapat diakses/i,
    );
  });

  it("renders the approval timeline including system escalation", () => {
    leaveState.current = {
      data: {
        ...leaveDetail("menunggu_hr"),
        approval_history: [
          {
            tahap: "atasan",
            approver_id: null,
            approver_nama: null,
            keputusan: "auto_eskalasi",
            catatan: null,
            decided_at: "2026-08-05T02:00:00Z",
          },
        ],
      },
      isPending: false,
      isError: false,
      refetch: refetchMock,
    };
    renderPage();

    expect(screen.getByText(/Dieskalasi otomatis ke HR/)).toBeInTheDocument();
    expect(screen.getByText(/Dipicu sistem/)).toBeInTheDocument();
  });
});
