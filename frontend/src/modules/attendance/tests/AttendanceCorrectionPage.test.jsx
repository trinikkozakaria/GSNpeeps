import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AttendanceCorrectionPage } from "../pages/AttendanceCorrectionPage";

const { authState, correctionsState, createMock, createState } = vi.hoisted(() => ({
  authState: { current: { role: "karyawan", user: { id: "user-1" } } },
  correctionsState: { current: {} },
  createMock: vi.fn(),
  createState: { current: {} },
}));

vi.mock("../../auth/hooks/useAuth", () => ({ useAuth: () => authState.current }));

vi.mock("../hooks/useAttendanceCorrections", () => ({
  useMyAttendanceCorrections: () => correctionsState.current,
  useCreateAttendanceCorrection: () => ({ mutateAsync: createMock, ...createState.current }),
}));

const correction = (overrides = {}) => ({
  id: "correction-1",
  tanggal: "2026-08-14",
  waktu_check_in: "09:15",
  waktu_check_out: null,
  alasan: "Perangkat absensi tidak dapat digunakan.",
  status: "menunggu_atasan",
  created_at: "2026-08-14T02:30:00Z",
  ...overrides,
});

const meta = (overrides = {}) => ({ page: 1, limit: 10, total_data: 0, total_page: 0, ...overrides });

const renderPage = () =>
  render(
    <MemoryRouter initialEntries={["/app/absensi/koreksi"]}>
      <AttendanceCorrectionPage />
    </MemoryRouter>,
  );

describe("AttendanceCorrectionPage", () => {
  beforeEach(() => {
    authState.current = { role: "karyawan", user: { id: "user-1" } };
    correctionsState.current = { data: { items: [], meta: meta() }, isPending: false, isError: false };
    createMock.mockReset();
    createMock.mockResolvedValue({});
    createState.current = { isPending: false, isError: false };
  });

  it("requires a date, one corrected time, and a meaningful reason", async () => {
    const user = userEvent.setup();
    renderPage();
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
    renderPage();
    expect(screen.getByText("Belum ada koreksi absensi.")).toBeInTheDocument();
  });

  // Halaman ini hanya riwayat pribadi; keputusan Setujui/Tolak untuk pengajuan orang
  // lain ada di halaman Persetujuan, bukan di sini — jadi tidak ada tombol keputusan
  // sama sekali, apa pun role dan status pengajuan sendiri.
  it.each(["karyawan", "atasan", "hr"])(
    "never shows decision controls for role %s, only the request's own status",
    (role) => {
      authState.current = { role, user: { id: "user-1" } };
      correctionsState.current = {
        data: { items: [correction({ status: "menunggu_hr" })], meta: meta({ total_data: 1, total_page: 1 }) },
        isPending: false,
        isError: false,
      };
      renderPage();

      expect(screen.getByText("Riwayat koreksi saya")).toBeInTheDocument();
      expect(screen.getByText(/Menunggu HR/)).toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "Setujui" })).not.toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "Tolak" })).not.toBeInTheDocument();
    },
  );

  it("shows the status label for a finished correction", () => {
    authState.current = { role: "hr", user: { id: "hr-1" } };
    correctionsState.current = {
      data: { items: [correction({ status: "disetujui" })], meta: meta({ total_data: 1, total_page: 1 }) },
      isPending: false,
      isError: false,
    };
    renderPage();

    expect(screen.getByText(/Disetujui/)).toBeInTheDocument();
  });

  it("paginates the personal history list", async () => {
    const user = userEvent.setup();
    authState.current = { role: "hr", user: { id: "hr-1" } };
    correctionsState.current = {
      data: {
        items: [correction()],
        meta: { page: 1, limit: 10, total_data: 25, total_page: 3 },
      },
      isPending: false,
      isError: false,
    };
    renderPage();

    expect(screen.getByText("25 koreksi")).toBeInTheDocument();
    const nextButton = screen.getByRole("button", { name: "Berikutnya" });
    expect(nextButton).toBeEnabled();
    await user.click(nextButton);
    // useSearchParams-driven page change re-renders with the mocked (unchanged) query
    // result; asserting the control exists and is wired is enough at this mock boundary.
    expect(screen.getByRole("button", { name: "Sebelumnya" })).toBeDisabled();
  });
});
