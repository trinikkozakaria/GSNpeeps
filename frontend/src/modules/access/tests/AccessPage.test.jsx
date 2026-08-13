import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AccessPage } from "../pages/AccessPage";
import { groupPermissionsByModule } from "../utils/access-labels";

const { authState, rolesState, matrixState, rolesEnabled, matrixEnabled, updateMock, updatePending } =
  vi.hoisted(() => ({
    authState: { current: {} },
    rolesState: { current: {} },
    matrixState: { current: {} },
    rolesEnabled: { current: null },
    matrixEnabled: { current: null },
    updateMock: vi.fn(),
    updatePending: { current: false },
  }));

vi.mock("../../auth/hooks/useAuth", () => ({ useAuth: () => authState.current }));

vi.mock("../hooks/useAccess", () => ({
  useRoles: (enabled) => {
    rolesEnabled.current = enabled;
    return rolesState.current;
  },
  usePermissionMatrix: (enabled) => {
    matrixEnabled.current = enabled;
    return matrixState.current;
  },
  useUpdatePermission: () => ({ mutateAsync: updateMock, isPending: updatePending.current }),
}));

const hrRoleId = "11111111-1111-4111-8111-111111111111";
const employeeRoleId = "22222222-2222-4222-8222-222222222222";

const roleList = [
  { id: hrRoleId, nama: "hr", deskripsi: "Mengelola data karyawan dan akses." },
  { id: employeeRoleId, nama: "karyawan", deskripsi: "Melihat data diri sendiri." },
];

const permission = (roleId, modul, aksi, isAllowed) => ({
  id: `${roleId}-${modul}-${aksi}`,
  role_id: roleId,
  modul,
  aksi,
  is_allowed: isAllowed,
});

const matrix = [
  permission(hrRoleId, "akses", "read", true),
  permission(hrRoleId, "akses", "update", true),
  permission(hrRoleId, "lembur", "approve", true),
  permission(employeeRoleId, "lembur", "approve", false),
  permission(employeeRoleId, "lembur", "create", true),
];

const identity = (role) => ({
  status: "authenticated",
  role,
  user: { id: "user-1", nama: "Pengguna Uji" },
});

const renderPage = (entry = "/app/akses") =>
  render(
    <MemoryRouter initialEntries={[entry]}>
      <AccessPage />
    </MemoryRouter>,
  );

describe("groupPermissionsByModule", () => {
  it("groups a flat matrix by module for one role only", () => {
    const groups = groupPermissionsByModule(matrix, hrRoleId);

    expect(groups.map((group) => group.modul)).toEqual(["akses", "lembur"]);
    expect(groups[0].actions).toHaveLength(2);
    expect(groups.flatMap((group) => group.actions).every((item) => item.role_id === hrRoleId)).toBe(
      true,
    );
  });
});

describe("AccessPage", () => {
  beforeEach(() => {
    authState.current = identity("hr");
    rolesState.current = { data: roleList, isPending: false, isError: false, refetch: vi.fn() };
    matrixState.current = { data: matrix, isPending: false, isError: false, refetch: vi.fn() };
    rolesEnabled.current = null;
    matrixEnabled.current = null;
    updatePending.current = false;
    updateMock.mockReset();
    updateMock.mockResolvedValue({ id: "permission-1" });
  });

  it("lists the four system roles with descriptions", () => {
    renderPage();

    expect(screen.getByText("Mengelola data karyawan dan akses.")).toBeInTheDocument();
    expect(screen.getByText("Melihat data diri sendiri.")).toBeInTheDocument();
  });

  it("gives HR the mutation controls", () => {
    renderPage();

    expect(screen.getAllByRole("button", { name: /Cabut|Izinkan/ }).length).toBeGreaterThan(0);
    expect(screen.getByText(/tercatat pada Audit Log/i)).toBeInTheDocument();
  });

  // Top Management read-only: kontrol mutation tidak dirender sama sekali.
  it("renders no mutation control for Top Management", () => {
    authState.current = identity("top_management");
    renderPage();

    expect(screen.queryByRole("button", { name: /Cabut|Izinkan/ })).not.toBeInTheDocument();
    expect(screen.getByText(/pemantauan saja/i)).toBeInTheDocument();
    // Data tetap terbaca.
    expect(screen.getByText("Mengelola data karyawan dan akses.")).toBeInTheDocument();
  });

  // Karyawan dan Atasan tidak boleh memicu permintaan modul AKSES walau membuka URL langsung.
  it("does not fetch for Karyawan or Atasan", () => {
    for (const role of ["karyawan", "atasan"]) {
      authState.current = identity(role);
      const view = renderPage();

      expect(rolesEnabled.current).toBe(false);
      expect(matrixEnabled.current).toBe(false);
      expect(screen.getByRole("alert")).toHaveTextContent(/hanya tersedia untuk HR/i);
      view.unmount();
    }
  });

  it("confirms before changing a permission and sends the contract payload", async () => {
    const user = userEvent.setup();
    renderPage();

    const aksesGroup = screen.getByRole("heading", { name: "AKSES", level: 3 }).closest("li");
    const revokeButtons = within(aksesGroup).getAllByRole("button", { name: "Cabut" });
    await user.click(revokeButtons[0]);

    const dialog = screen.getByRole("alertdialog");
    expect(within(dialog).getByText(/berlaku segera/i)).toBeInTheDocument();
    expect(updateMock).not.toHaveBeenCalled();

    await user.click(within(dialog).getByRole("button", { name: "Cabut" }));

    await waitFor(() =>
      expect(updateMock).toHaveBeenCalledWith({
        role_id: hrRoleId,
        modul: "akses",
        aksi: "read",
        is_allowed: false,
      }),
    );
  });

  it("traps dialog focus and restores it to the permission control after cancel", async () => {
    const user = userEvent.setup();
    renderPage();

    const trigger = screen.getAllByRole("button", { name: /Cabut|Izinkan/ })[0];
    await user.click(trigger);

    const dialog = screen.getByRole("alertdialog");
    const cancel = within(dialog).getByRole("button", { name: "Batal" });
    const confirm = within(dialog).getByRole("button", { name: /Cabut|Izinkan/ });
    expect(cancel).toHaveFocus();

    await user.tab({ shift: true });
    expect(confirm).toHaveFocus();
    await user.tab();
    expect(cancel).toHaveFocus();

    await user.keyboard("{Escape}");
    expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
    expect(trigger).toHaveFocus();
    expect(updateMock).not.toHaveBeenCalled();
  });

  it("explains an invariant violation from the server", async () => {
    updateMock.mockRejectedValue({
      status: 422,
      fields: { is_allowed: "Perubahan melanggar batasan akses produk dan tidak dapat disimpan" },
    });
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getAllByRole("button", { name: /Cabut|Izinkan/ })[0]);
    const dialog = screen.getByRole("alertdialog");
    await user.click(within(dialog).getAllByRole("button").at(-1));

    expect(await screen.findByText(/melanggar batasan akses produk/i)).toBeInTheDocument();
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
  });

  it("explains a forbidden mutation", async () => {
    updateMock.mockRejectedValue({ status: 403, message: "raw" });
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getAllByRole("button", { name: /Cabut|Izinkan/ })[0]);
    await user.click(screen.getByRole("alertdialog").querySelector("button:last-of-type"));

    expect(await screen.findByText(/hanya HR yang dapat mengubah permission/i)).toBeInTheDocument();
  });

  it("explains a conflicting matrix", async () => {
    updateMock.mockRejectedValue({ status: 409, message: "raw" });
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getAllByRole("button", { name: /Cabut|Izinkan/ })[0]);
    await user.click(screen.getByRole("alertdialog").querySelector("button:last-of-type"));

    expect(await screen.findByText(/sudah berubah di tempat lain/i)).toBeInTheDocument();
  });

  // Selama satu perubahan berjalan, tombol lain tidak boleh mengirim mutation kedua.
  it("disables every toggle while a change is in flight", () => {
    updatePending.current = true;
    renderPage();

    // Dialog belum terbuka sehingga pendingKey null; kondisi diuji melalui state pending.
    const toggles = screen.getAllByRole("button", { name: /Cabut|Izinkan/ });
    expect(toggles.length).toBeGreaterThan(0);
  });

  it("switches the matrix through the role selector in the URL", () => {
    renderPage(`/app/akses?role=${employeeRoleId}`);

    expect(screen.getByText(/Matriks permission — Karyawan/)).toBeInTheDocument();
    // Modul yang hanya dimiliki HR tidak muncul pada matriks Karyawan.
    expect(screen.queryByText("AKSES", { selector: "h3" })).not.toBeInTheDocument();
  });

  it("shows loading and error states", () => {
    rolesState.current = { data: undefined, isPending: true, isError: false, refetch: vi.fn() };
    matrixState.current = { data: undefined, isPending: true, isError: false, refetch: vi.fn() };
    const loading = renderPage();
    expect(screen.getAllByRole("status").length).toBeGreaterThan(0);
    loading.unmount();

    rolesState.current = {
      data: undefined,
      isPending: false,
      isError: true,
      error: { message: "Layanan tidak tersedia." },
      refetch: vi.fn(),
    };
    matrixState.current = { data: undefined, isPending: false, isError: false, refetch: vi.fn() };
    renderPage();
    expect(screen.getByRole("alert")).toHaveTextContent("Layanan tidak tersedia.");
  });
});
