import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { NotificationBell } from "../components/NotificationBell";

const { authState, countState, countEnabled, countUserId } = vi.hoisted(() => ({
  authState: { current: {} },
  countState: { current: {} },
  countEnabled: { current: null },
  countUserId: { current: null },
}));

vi.mock("../../auth/hooks/useAuth", () => ({ useAuth: () => authState.current }));

vi.mock("../hooks/useNotifications", () => ({
  useUnreadNotificationCount: (userId, enabled) => {
    countUserId.current = userId;
    countEnabled.current = enabled;
    return countState.current;
  },
}));

const authenticated = {
  status: "authenticated",
  role: "karyawan",
  user: { id: "11111111-1111-4111-8111-111111111111", nama: "Karyawan Uji" },
};

const renderBell = () =>
  render(
    <MemoryRouter>
      <NotificationBell />
    </MemoryRouter>,
  );

describe("NotificationBell", () => {
  beforeEach(() => {
    authState.current = authenticated;
    countEnabled.current = null;
    countUserId.current = null;
    countState.current = { data: 3, isPending: false, isError: false, isSuccess: true };
  });

  it("announces the unread count in the accessible name", () => {
    renderBell();

    expect(screen.getByRole("link", { name: "Notifikasi, 3 belum dibaca" })).toBeInTheDocument();
    expect(screen.getByRole("link")).toHaveAttribute("href", "/app/notifikasi");
  });

  // Angka besar tidak boleh merusak lebar badge.
  it("caps very large counts", () => {
    countState.current = { data: 1200, isPending: false, isError: false, isSuccess: true };
    renderBell();

    expect(screen.getByText("99+")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /1\.?200 belum dibaca/ })).toBeInTheDocument();
  });

  // Selama memuat, badge tidak boleh menampilkan angka apa pun.
  it("shows no number while loading", () => {
    countState.current = { data: undefined, isPending: true, isError: false, isSuccess: false };
    renderBell();

    expect(screen.getByRole("link", { name: /sedang dimuat/i })).toBeInTheDocument();
    expect(screen.queryByText("0")).not.toBeInTheDocument();
  });

  it("stays usable when the count cannot be loaded", () => {
    countState.current = { data: undefined, isPending: false, isError: true, isSuccess: false };
    renderBell();

    expect(screen.getByRole("link", { name: /tidak dapat dimuat/i })).toBeInTheDocument();
  });

  it("hides the badge when nothing is unread", () => {
    countState.current = { data: 0, isPending: false, isError: false, isSuccess: true };
    renderBell();

    expect(
      screen.getByRole("link", { name: "Notifikasi, tidak ada yang belum dibaca" }),
    ).toBeInTheDocument();
  });

  // Inbox tidak boleh diminta sebelum sesi terautentikasi.
  it("does not fetch before the session is authenticated", () => {
    authState.current = { status: "initializing", role: null, user: null };
    renderBell();

    expect(countEnabled.current).toBe(false);
    expect(countUserId.current).toBeUndefined();
  });

  it("scopes the request to the authenticated user", () => {
    renderBell();

    expect(countEnabled.current).toBe(true);
    expect(countUserId.current).toBe(authenticated.user.id);
  });
});
