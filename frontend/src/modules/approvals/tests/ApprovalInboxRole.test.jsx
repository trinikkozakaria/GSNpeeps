import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApprovalInboxPage } from "../pages/ApprovalInboxPage";

const {
  authState,
  correctionEnabled,
  correctionState,
  decideCorrectionMock,
  leaveEnabled,
  overtimeEnabled,
} = vi.hoisted(() => ({
  authState: { current: { role: "atasan", user: { id: "user-1" } } },
  correctionEnabled: { current: null },
  correctionState: { current: null },
  decideCorrectionMock: vi.fn(),
  leaveEnabled: { current: null },
  overtimeEnabled: { current: null },
}));

const emptyPage = {
  data: { items: [], meta: { page: 1, limit: 10, total_data: 0, total_page: 0 } },
  isPending: false,
  isError: false,
};

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

vi.mock("../../attendance/hooks/useAttendanceCorrections", () => ({
  useAttendanceCorrections: (_scope, _params, enabled) => {
    correctionEnabled.current = enabled;
    return correctionState.current ?? emptyPage;
  },
  useDecideAttendanceCorrection: () => ({ mutateAsync: decideCorrectionMock, isPending: false }),
}));

vi.mock("../../auth/hooks/useAuth", () => ({ useAuth: () => authState.current }));

vi.mock("../../leave/hooks/useLeave", () => ({
  useLeaveApprovalInbox: (_scope, _params, enabled) => {
    leaveEnabled.current = enabled;
    return emptyPage;
  },
}));

vi.mock("../../overtime/hooks/useOvertime", () => ({
  useOvertimeList: (_scope, _params, enabled) => {
    overtimeEnabled.current = enabled;
    return emptyPage;
  },
}));

const renderInbox = (entry = "/app/persetujuan") =>
  render(
    <MemoryRouter initialEntries={[entry]}>
      <ApprovalInboxPage />
    </MemoryRouter>,
  );

describe("ApprovalInboxPage", () => {
  beforeEach(() => {
    leaveEnabled.current = null;
    overtimeEnabled.current = null;
    correctionEnabled.current = null;
    correctionState.current = null;
    decideCorrectionMock.mockReset();
    decideCorrectionMock.mockResolvedValue({});
  });

  it("explains the scope for each approver role", () => {
    const expectations = {
      atasan: /bawahan langsung/i,
      hr: /menunggu keputusan hr/i,
      top_management: /milik hr/i,
    };
    for (const [role, expected] of Object.entries(expectations)) {
      authState.current = { role, user: { id: "user-1" } };
      const view = renderInbox();
      expect(screen.getByText(expected)).toBeInTheDocument();
      view.unmount();
    }
  });

  // Karyawan bukan approver dan tidak boleh memicu permintaan antrean.
  it("does not fetch for a role without an approval queue", () => {
    authState.current = { role: "karyawan", user: { id: "user-9" } };
    renderInbox();

    expect(leaveEnabled.current).toBeFalsy();
    expect(overtimeEnabled.current).toBeFalsy();
    expect(correctionEnabled.current).toBeFalsy();
    expect(screen.getByRole("status")).toHaveTextContent(/tidak memiliki antrean/i);
  });

  it("fetches only the active tab", () => {
    authState.current = { role: "hr", user: { id: "user-1" } };
    renderInbox();
    expect(leaveEnabled.current).toBe(true);
    expect(overtimeEnabled.current).toBe(false);
  });

  it("switches to the overtime queue from the URL", () => {
    authState.current = { role: "hr", user: { id: "user-1" } };
    renderInbox("/app/persetujuan?tab=lembur");

    expect(overtimeEnabled.current).toBe(true);
    expect(leaveEnabled.current).toBe(false);
    expect(screen.getByRole("tab", { name: "Lembur" })).toHaveAttribute("aria-selected", "true");
  });

  it.each(["atasan", "hr", "top_management"])(
    "integrates attendance corrections for approver role %s",
    (role) => {
      authState.current = { role, user: { id: "user-1" } };
      renderInbox("/app/persetujuan?tab=koreksi");

      expect(correctionEnabled.current).toBe(true);
      expect(leaveEnabled.current).toBe(false);
      expect(screen.getByRole("tab", { name: "Koreksi Absensi" })).toHaveAttribute("aria-selected", "true");
      expect(screen.getByText("Tidak ada koreksi absensi yang menunggu keputusan Anda.")).toBeInTheDocument();
    },
  );

  it("shows the status and lets the active supervisor stage decide from the queue", async () => {
    const user = userEvent.setup();
    authState.current = { role: "atasan", user: { id: "user-1" } };
    correctionState.current = {
      data: { items: [correction()], meta: { page: 1, limit: 10, total_data: 1, total_page: 1 } },
      isPending: false,
      isError: false,
    };
    renderInbox("/app/persetujuan?tab=koreksi");

    // DataTable merender layout mobile (<ul>) dan desktop (<table>) sekaligus di DOM.
    expect(screen.getAllByText("Menunggu Atasan").length).toBeGreaterThan(0);
    await user.click(screen.getAllByRole("button", { name: "Setujui" })[0]);
    expect(decideCorrectionMock).toHaveBeenCalledWith({ id: "correction-1", keputusan: "setujui" });
  });

  it("hides decision controls once the correction sits on another stage", () => {
    authState.current = { role: "atasan", user: { id: "user-1" } };
    correctionState.current = {
      data: {
        items: [correction({ status: "menunggu_hr" })],
        meta: { page: 1, limit: 10, total_data: 1, total_page: 1 },
      },
      isPending: false,
      isError: false,
    };
    renderInbox("/app/persetujuan?tab=koreksi");

    expect(screen.getAllByText("Menunggu HR").length).toBeGreaterThan(0);
    expect(screen.queryByRole("button", { name: "Setujui" })).not.toBeInTheDocument();
  });

  it("lets Top Management decide corrections submitted by HR from the queue", async () => {
    const user = userEvent.setup();
    authState.current = { role: "top_management", user: { id: "user-1" } };
    correctionState.current = {
      data: {
        items: [correction({ nama_karyawan: "HR Sintetis", status: "menunggu_top_management" })],
        meta: { page: 1, limit: 10, total_data: 1, total_page: 1 },
      },
      isPending: false,
      isError: false,
    };
    renderInbox("/app/persetujuan?tab=koreksi");

    expect(screen.getAllByText("Menunggu Top Management").length).toBeGreaterThan(0);
    await user.click(screen.getAllByRole("button", { name: "Setujui" })[0]);
    expect(decideCorrectionMock).toHaveBeenCalledWith({ id: "correction-1", keputusan: "setujui" });
  });

  it("surfaces a decision error inline without losing the queue", async () => {
    decideCorrectionMock.mockRejectedValue({ message: "Layanan tidak tersedia." });
    const user = userEvent.setup();
    authState.current = { role: "hr", user: { id: "user-1" } };
    correctionState.current = {
      data: {
        items: [correction({ status: "menunggu_hr" })],
        meta: { page: 1, limit: 10, total_data: 1, total_page: 1 },
      },
      isPending: false,
      isError: false,
    };
    renderInbox("/app/persetujuan?tab=koreksi");

    await user.click(screen.getAllByRole("button", { name: "Tolak" })[0]);
    expect(await screen.findByText("Layanan tidak tersedia.")).toBeInTheDocument();
  });
});
