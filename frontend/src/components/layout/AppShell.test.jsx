import { fireEvent, render, screen, within } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AppShell } from "./AppShell";

const logout = vi.fn();

vi.mock("../../modules/auth/hooks/useAuth", () => ({
  useAuth: () => ({
    role: "hr",
    user: { nama: "HR Sintetis", foto_profil_url: null },
    logout,
  }),
}));

vi.mock("../../modules/notifications/components/NotificationBell", () => ({
  NotificationBell: () => <span>Notifikasi</span>,
}));

const renderShell = () =>
  render(
    <MemoryRouter initialEntries={["/app"]}>
      <Routes>
        <Route path="/app" element={<AppShell />}>
          <Route index element={<p>Konten halaman</p>} />
        </Route>
      </Routes>
    </MemoryRouter>,
  );

describe("AppShell sidebar", () => {
  beforeEach(() => {
    logout.mockReset();
  });

  it("keeps compact menu buttons usable when the desktop sidebar is collapsed", () => {
    renderShell();

    const navigation = screen.getByRole("navigation", { name: "Navigasi utama" });
    const menu = screen.getByRole("list", { name: "Menu utama" });
    const main = screen.getByRole("main");

    expect(main).toHaveClass("lg:ml-60");
    expect(navigation).toHaveClass("lg:w-60");
    expect(menu).toHaveClass("lg:block");
    expect(screen.queryByRole("button", { name: "Buka sidebar" })).not.toBeInTheDocument();

    fireEvent.pointerDown(main);
    expect(main).toHaveClass("lg:ml-20");
    expect(navigation).toHaveClass("lg:w-20");
    expect(menu).toHaveClass("lg:hidden");
    const hamburger = screen.getByRole("button", { name: "Buka sidebar" });
    expect(hamburger).toHaveAttribute("aria-expanded", "false");
    expect(hamburger).toHaveClass("h-12", "w-12");
    const compactMenu = screen.getByRole("list", { name: "Menu ringkas" });
    expect(compactMenu).toHaveClass("lg:block");
    expect(compactMenu).not.toHaveTextContent(/BR|PR|PG|PS|OR|MN|IN|MD|AD|AK/);
    expect(within(compactMenu).getByRole("link", { name: "Beranda" })).toBeInTheDocument();
    expect(within(compactMenu).getByRole("button", { name: "Pribadi" })).toHaveClass("h-12", "w-12");

    fireEvent.mouseEnter(navigation);
    expect(main).toHaveClass("lg:ml-20");
    expect(menu).toHaveClass("lg:hidden");

    fireEvent.click(within(compactMenu).getByRole("button", { name: "Pribadi" }));
    const flyout = screen.getByRole("group", { name: "Submenu Pribadi" });
    expect(within(flyout).getByRole("link", { name: "Profil Saya" })).toBeInTheDocument();
    expect(within(flyout).getByRole("link", { name: "Kehadiran Saya" })).toBeInTheDocument();

    fireEvent.pointerDown(main);
    expect(screen.queryByRole("group", { name: "Submenu Pribadi" })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Buka sidebar" }));
    expect(main).toHaveClass("lg:ml-60");
    expect(menu).toHaveClass("lg:block");
    expect(screen.queryByRole("button", { name: "Buka sidebar" })).not.toBeInTheDocument();
  });

  it("groups module links and reveals their children from the parent control", () => {
    renderShell();

    const group = screen.getByRole("button", { name: "Pribadi" });
    expect(group).toHaveAttribute("aria-expanded", "false");

    fireEvent.click(group);

    expect(group).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByRole("link", { name: "Profil Saya" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Koreksi Absensi" })).toBeInTheDocument();
  });
});
