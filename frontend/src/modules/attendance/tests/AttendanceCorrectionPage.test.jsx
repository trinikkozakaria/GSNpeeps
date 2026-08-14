import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AttendanceCorrectionPage } from "../pages/AttendanceCorrectionPage";

const { authState, correctionsState, createMock, createState, decideMock, decideState } =
  vi.hoisted(() => ({
    authState: { current: { role: "karyawan", user: { id: "user-1" } } },
    correctionsState: { current: {} },
    createMock: vi.fn(),
    createState: { current: {} },
    decideMock: vi.fn(),
    decideState: { current: {} },
  }));

vi.mock("../../auth/hooks/useAuth", () => ({ useAuth: () => authState.current }));

vi.mock("../hooks/useAttendanceCorrections", () => ({
  useAttendanceCorrections: () => correctionsState.current,
  useCreateAttendanceCorrection: () => ({ mutateAsync: createMock, ...createState.current }),
  useDecideAttendanceCorrection: () => ({ mutateAsync: decideMock, ...decideState.current }),
}));

const correction = (overrides = {}) => ({
  id: "correction-1",
  nama_karyawan: "Karyawan Sintetis",
  tanggal: "2026-08-14",
  waktu_check_in: "09:15",
  waktu_check_out: null,
  alasan: "Perangkat absensi tidak dapat digunakan.",
  status: "menunggu_atasan",
  created_at: "2026-08-14T02:30:00Z",
  ...overrides,
});

describe("AttendanceCorrectionPage", () => {
  beforeEach(() => {
    authState.current = { role: "karyawan", user: { id: "user-1" } };
    correctionsState.current = { data: [], isPending: false, isError: false };
    createMock.mockReset();
    createMock.mockResolvedValue({});
    createState.current = { isPending: false, isError: false };
    decideMock.mockReset();
    decideMock.mockResolvedValue({});
    decideState.current = { isPending: false };
  });

  it("requires a date, one corrected time, and a meaningful reason", async () => {
    const user = userEvent.setup();
    render(<AttendanceCorrectionPage />);
    const submit = screen.getByRole("button", { name: "Ajukan koreksi" });

    expect(submit).toBeDisabled();
    fireEvent.change(screen.getByLabelText("Tanggal"), { target: { value: "2026-08-14" } });
    fireEvent.change(screen.getByLabelText("Jam masuk"), { target: { value: "09:15" } });
    await user.type(screen.getByLabelText(/^Alasan/), "Perangkat absensi bermasalah");
    expect(submit).toBeEnabled();

    await user.click(submit);
    await waitFor(() => expect(createMock).toHaveBeenCalledWith({
      tanggal: "2026-08-14",
      waktu_check_in: "09:15",
      waktu_check_out: null,
      alasan: "Perangkat absensi bermasalah",
    }));
    expect(await screen.findByRole("status")).toHaveTextContent("berhasil diajukan");
  });

  it("shows a clear empty state", () => {
    render(<AttendanceCorrectionPage />);
    expect(screen.getByText("Belum ada koreksi absensi.")).toBeInTheDocument();
  });

  it("lets the active supervisor stage approve or reject", async () => {
    const user = userEvent.setup();
    authState.current = { role: "atasan", user: { id: "supervisor-1" } };
    correctionsState.current = { data: [correction()], isPending: false, isError: false };
    render(<AttendanceCorrectionPage />);

    await user.click(screen.getByRole("button", { name: "Setujui" }));
    expect(decideMock).toHaveBeenCalledWith({ id: "correction-1", keputusan: "setujui" });
  });

  it("does not show decision controls for completed corrections", () => {
    authState.current = { role: "hr", user: { id: "hr-1" } };
    correctionsState.current = {
      data: [correction({ status: "disetujui" })],
      isPending: false,
      isError: false,
    };
    render(<AttendanceCorrectionPage />);

    expect(screen.queryByRole("button", { name: "Setujui" })).not.toBeInTheDocument();
    expect(screen.getByText(/Disetujui/)).toBeInTheDocument();
  });
});
