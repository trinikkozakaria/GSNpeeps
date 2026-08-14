import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { HolidayCalendarPage } from "../pages/HolidayCalendarPage";

const { authState, holidaysState, saveMock, saveState } = vi.hoisted(() => ({
  authState: { current: { role: "hr" } },
  holidaysState: { current: {} },
  saveMock: vi.fn(),
  saveState: { current: {} },
}));

vi.mock("../../auth/hooks/useAuth", () => ({ useAuth: () => authState.current }));

vi.mock("@tanstack/react-query", async (importOriginal) => ({
  ...(await importOriginal()),
  useQuery: () => holidaysState.current,
  useMutation: () => ({ mutateAsync: saveMock, ...saveState.current }),
}));

describe("HolidayCalendarPage", () => {
  beforeEach(() => {
    authState.current = { role: "hr" };
    holidaysState.current = { data: [], isPending: false, isError: false };
    saveMock.mockReset();
    saveMock.mockResolvedValue({});
    saveState.current = { isPending: false, isError: false };
  });

  it("shows a retryable calendar error", () => {
    holidaysState.current = {
      isPending: false,
      isError: true,
      error: { message: "Kalender gagal." },
      refetch: vi.fn(),
    };
    render(<HolidayCalendarPage />);
    expect(screen.getByRole("alert")).toHaveTextContent("Kalender gagal");
    expect(screen.getByRole("button", { name: "Coba lagi" })).toBeInTheDocument();
  });

  it("keeps bulk editing HR-only", () => {
    authState.current = { role: "karyawan" };
    render(<HolidayCalendarPage />);
    expect(screen.queryByRole("heading", { name: "Bulk insert / update hari libur" })).not.toBeInTheDocument();
  });

  it("trims and saves multiple holiday fields", async () => {
    const user = userEvent.setup();
    render(<HolidayCalendarPage />);
    const submit = screen.getByRole("button", { name: "Simpan sekaligus" });
    expect(submit).toBeDisabled();

    fireEvent.change(screen.getByLabelText("Tanggal 1"), { target: { value: "2026-08-17" } });
    await user.type(screen.getByLabelText("Nama 1"), "  Hari Kemerdekaan  ");
    await user.type(screen.getByLabelText("Keterangan 1"), "  Libur nasional  ");
    await user.click(submit);

    await waitFor(() => expect(saveMock).toHaveBeenCalledWith([{
      tanggal: "2026-08-17",
      nama: "Hari Kemerdekaan",
      keterangan: "Libur nasional",
    }]));
    expect(await screen.findByText("Hari libur berhasil disimpan.")).toBeInTheDocument();
  });
});
