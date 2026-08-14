import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { LeaveTypesPage } from "../pages/LeaveTypesPage";

const { createMock, updateMock } = vi.hoisted(() => ({
  createMock: vi.fn(),
  updateMock: vi.fn(),
}));

vi.mock("../../auth/hooks/useAuth", () => ({
  useAuth: () => ({ role: "hr" }),
}));

vi.mock("../hooks/useLeave", () => ({
  useLeaveTypes: () => ({
    data: [{
      id: "44444444-4444-4444-8444-444444444444",
      kode: "IZIN-HAJI",
      nama: "Ibadah Haji",
      kategori: "izin",
      kuota_tahunan: 0,
      maksimal_hari: 30,
      memerlukan_dokumen: true,
      is_active: true,
    }],
    isPending: false,
    isError: false,
  }),
  useCreateLeaveType: () => ({ mutateAsync: createMock, isPending: false }),
  useUpdateLeaveType: () => ({ mutateAsync: updateMock, isPending: false }),
}));

describe("LeaveTypesPage", () => {
  beforeEach(() => {
    createMock.mockReset();
    updateMock.mockReset();
    updateMock.mockResolvedValue({});
  });

  it("explains immutable identifiers and rejects a zero maximum", async () => {
    const user = userEvent.setup();
    render(<LeaveTypesPage />);

    await user.click(screen.getAllByRole("button", { name: "Edit Ibadah Haji" })[0]);
    const dialog = screen.getByRole("dialog", { name: /edit ibadah haji/i });
    expect(dialog).toHaveTextContent(/kode IZIN-HAJI dan kategori izin dikunci/i);

    const maximum = within(dialog).getByLabelText("Maksimal hari izin");
    await user.clear(maximum);
    await user.type(maximum, "0");
    await user.click(within(dialog).getByRole("button", { name: "Simpan perubahan" }));

    expect(await screen.findByText(/maksimal hari wajib diisi/i)).toBeInTheDocument();
    expect(updateMock).not.toHaveBeenCalled();
  });

  it("submits edited fields without changing code or category", async () => {
    const user = userEvent.setup();
    render(<LeaveTypesPage />);

    await user.click(screen.getAllByRole("button", { name: "Edit Ibadah Haji" })[0]);
    const dialog = screen.getByRole("dialog", { name: /edit ibadah haji/i });
    const maximum = within(dialog).getByLabelText("Maksimal hari izin");
    await user.clear(maximum);
    await user.type(maximum, "25");
    const checkbox = within(dialog).getByRole("checkbox", { name: "Dokumen pendukung wajib" });
    await user.click(checkbox);
    await user.click(within(dialog).getByRole("button", { name: "Simpan perubahan" }));

    await waitFor(() => expect(updateMock).toHaveBeenCalledTimes(1));
    expect(updateMock).toHaveBeenCalledWith({
      id: "44444444-4444-4444-8444-444444444444",
      payload: {
        nama: "Ibadah Haji",
        kuota_tahunan: 0,
        maksimal_hari: 25,
        memerlukan_dokumen: false,
      },
    });
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });
});
