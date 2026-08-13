import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApprovalInboxPage } from "../pages/ApprovalInboxPage";

const { authState, leaveEnabled, overtimeEnabled } = vi.hoisted(() => ({
  authState: { current: { role: "atasan", user: { id: "user-1" } } },
  leaveEnabled: { current: null },
  overtimeEnabled: { current: null },
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
});
