import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApprovalInboxPage } from "../pages/ApprovalInboxPage";

const { authState, correctionEnabled, leaveEnabled, overtimeEnabled } = vi.hoisted(() => ({
  authState: { current: { role: "atasan", user: { id: "user-1" } } },
  correctionEnabled: { current: null },
  leaveEnabled: { current: null },
  overtimeEnabled: { current: null },
}));

vi.mock("../../attendance/hooks/useAttendanceCorrections", () => ({
  useAttendanceCorrections: (_scope, enabled) => {
    correctionEnabled.current = enabled;
    return { data: [], isPending: false, isError: false };
  },
}));

vi.mock("../../auth/hooks/useAuth", () => ({ useAuth: () => authState.current }));

const emptyPage = {
  data: { items: [], meta: { page: 1, limit: 10, total_data: 0, total_page: 0 } },
  isPending: false,
  isError: false,
};

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

  it("integrates attendance corrections for supervisors and HR only", () => {
    authState.current = { role: "atasan", user: { id: "user-1" } };
    renderInbox("/app/persetujuan?tab=koreksi");

    expect(correctionEnabled.current).toBe(true);
    expect(leaveEnabled.current).toBe(false);
    expect(screen.getByRole("tab", { name: "Koreksi Absensi" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByText("Tidak ada koreksi absensi yang menunggu keputusan Anda.")).toBeInTheDocument();
  });

  it("does not expose attendance corrections to Top Management", () => {
    authState.current = { role: "top_management", user: { id: "user-1" } };
    renderInbox("/app/persetujuan?tab=koreksi");

    expect(correctionEnabled.current).toBe(false);
    expect(screen.queryByRole("tab", { name: "Koreksi Absensi" })).not.toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Ketidakhadiran" })).toHaveAttribute("aria-selected", "true");
  });
});
