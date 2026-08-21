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

const renderShell = (entry = "/app") =>
  render(
    <MemoryRouter initialEntries={[entry]}>
      <Routes>
        <Route path="/app" element={<AppShell />}>
          <Route index element={<p>Konten halaman</p>} />
          <Route path="profil" element={<p>Konten profil</p>} />
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

    expect(within(navigation).queryByText("GSNpeeps")).not.toBeInTheDocument();
    expect(within(navigation).queryByRole("img")).not.toBeInTheDocument();
    expect(main).toHaveClass("lg:ml-60");
    expect(navigation).toHaveClass("lg:w-60");
    expect(menu).toHaveClass("lg:block");
    expect(screen.queryByRole("button", { name: "Buka sidebar" })).not.toBeInTheDocument();

    // Klik di luar sidebar tidak boleh mengubah collapsed/expanded state; hanya tombol
    // dedicated di pojok kanan atas sidebar yang boleh melakukannya.
    fireEvent.pointerDown(main);
    expect(main).toHaveClass("lg:ml-60");
    expect(navigation).toHaveClass("lg:w-60");
    expect(menu).toHaveClass("lg:block");

    fireEvent.click(screen.getByRole("button", { name: "Ciutkan sidebar" }));
    expect(main).toHaveClass("lg:ml-20");
    expect(navigation).toHaveClass("lg:w-20");
    expect(menu).toHaveClass("lg:hidden");
    const hamburger = screen.getByRole("button", { name: "Buka sidebar" });
    expect(hamburger).toHaveAttribute("aria-expanded", "false");
    expect(hamburger).toHaveClass("h-8", "w-8", "rounded-md", "focus-visible:outline-2");
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

  it("opens the mobile navbar menu with the same compact icon list as the collapsed sidebar", () => {
    renderShell();
    const main = screen.getByRole("main");

    expect(screen.queryByRole("list", { name: "Menu ringkas" })).not.toBeInTheDocument();

    const toggle = screen.getByRole("button", { name: "Buka menu navigasi" });
    fireEvent.click(toggle);

    expect(toggle).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByRole("button", { name: "Tutup menu navigasi" })).toBe(toggle);
    const compactMenu = screen.getByRole("list", { name: "Menu ringkas" });
    expect(within(compactMenu).getByRole("link", { name: "Beranda" })).toBeInTheDocument();
    expect(within(compactMenu).getByRole("button", { name: "Pribadi" })).toBeInTheDocument();

    // Klik di luar navbar/menu menutup menu mobile, sama seperti flyout desktop.
    fireEvent.pointerDown(main);
    expect(screen.queryByRole("list", { name: "Menu ringkas" })).not.toBeInTheDocument();

    fireEvent.click(toggle);
    expect(screen.getByRole("list", { name: "Menu ringkas" })).toBeInTheDocument();
    fireEvent.keyDown(window, { key: "Escape" });
    expect(screen.queryByRole("list", { name: "Menu ringkas" })).not.toBeInTheDocument();

    // Tombol toggle sendiri harus tetap membuka/menutup menu, bukan diperlakukan sebagai
    // klik di luar yang langsung menutupnya kembali.
    fireEvent.click(toggle);
    expect(screen.getByRole("list", { name: "Menu ringkas" })).toBeInTheDocument();
    fireEvent.click(toggle);
    expect(screen.queryByRole("list", { name: "Menu ringkas" })).not.toBeInTheDocument();
  });

  it("groups module links and reveals their children from the parent control", () => {
    renderShell();

    const menu = screen.getByRole("list", { name: "Menu utama" });
    const topLevelItems = [
      ["link", "Beranda"],
      ...["Pribadi", "Pengajuan", "Persetujuan", "Organisasi", "Monitoring", "Informasi", "Master Data", "Administrasi", "Akun"]
        .map((name) => ["button", name]),
    ];
    topLevelItems.forEach(([role, name]) => {
      expect(within(menu).getByRole(role, { name }).querySelector("svg")).toBeInTheDocument();
    });

    const group = screen.getByRole("button", { name: "Pribadi" });
    expect(group).toHaveAttribute("aria-expanded", "false");
    const chevron = group.querySelector("span[aria-hidden='true']");
    expect(chevron).toHaveClass("h-6", "w-6", "items-center", "justify-center");
    expect(chevron.querySelector("path")).toHaveAttribute("d", "m8 5 5 5-5 5");

    fireEvent.click(group);

    expect(group).toHaveAttribute("aria-expanded", "true");
    expect(chevron.querySelector("path")).toHaveAttribute("d", "m5 8 5 5 5-5");
    const profileLink = screen.getByRole("link", { name: "Profil Saya" });
    expect(profileLink).toBeInTheDocument();
    expect(profileLink).toHaveClass("py-1.5");
    expect(profileLink.closest("ul")).toHaveClass("lg:mt-1", "lg:space-y-0.5");
    expect(screen.getByRole("link", { name: "Koreksi Absensi" })).toBeInTheDocument();
  });

  it("shows the active submenu as bold blue text", () => {
    renderShell("/app/profil");

    const profileLink = screen.getByRole("link", { name: "Profil Saya" });
    expect(profileLink).toHaveClass("font-bold", "text-cyan-700");
    expect(profileLink).not.toHaveClass("bg-cyan-50");
    expect(screen.getByRole("button", { name: "Pribadi" })).toHaveAttribute("aria-expanded", "true");
  });

  it("opens account settings and sign out from the profile dropdown", () => {
    renderShell();

    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
    const profileButton = screen.getByRole("button", { name: "Buka menu profil" });
    expect(profileButton).toHaveClass("rounded-xl", "border", "max-w-72");
    expect(profileButton).not.toHaveClass("sm:min-w-64");
    expect(profileButton.querySelector("svg")).toHaveClass("text-slate-950");
    fireEvent.click(profileButton);

    const menu = screen.getByRole("menu");
    expect(within(menu).getByRole("menuitem", { name: "Pengaturan Akun" }))
      .toHaveAttribute("href", "/app/keamanan");
    expect(within(menu).getByRole("menuitem", { name: "Keluar" })).toBeInTheDocument();
  });
});
