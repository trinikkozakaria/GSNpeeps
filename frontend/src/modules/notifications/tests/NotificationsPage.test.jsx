import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { NotificationsPage } from "../pages/NotificationsPage";
import {
  notificationDeepLink,
  notificationTargetForRole,
} from "../utils/notification-presentation";

const { authState, listState, listParams, markMock, dismissMock, markPending, dismissPending } =
  vi.hoisted(() => ({
    authState: { current: {} },
    listState: { current: {} },
    listParams: { current: null },
    markMock: vi.fn(),
    dismissMock: vi.fn(),
    markPending: { current: false },
    dismissPending: { current: false },
  }));

vi.mock("../../auth/hooks/useAuth", () => ({ useAuth: () => authState.current }));

vi.mock("../hooks/useNotifications", () => ({
  useNotifications: (_userId, params) => {
    listParams.current = params;
    return listState.current;
  },
  useMarkNotificationRead: () => ({ mutateAsync: markMock, mutate: markMock, isPending: markPending.current }),
  useDismissNotification: () => ({ mutateAsync: dismissMock, isPending: dismissPending.current }),
}));

const notification = (overrides = {}) => ({
  id: "22222222-2222-4222-8222-222222222222",
  user_id: "11111111-1111-4111-8111-111111111111",
  tipe: "ketidakhadiran_baru",
  judul: "Pengajuan ketidakhadiran baru",
  pesan: "Ada pengajuan ketidakhadiran yang menunggu keputusan Anda.",
  reference_id: "33333333-3333-4333-8333-333333333333",
  reference_type: "ketidakhadiran",
  is_read: false,
  read_at: null,
  created_at: "2026-08-03T02:00:00Z",
  ...overrides,
});

const pageOf = (items) => ({
  data: {
    items,
    meta: { page: 1, limit: 10, total_data: items.length, total_page: 1 },
  },
  isPending: false,
  isError: false,
  refetch: vi.fn(),
});

const renderPage = (entry = "/app/notifikasi") =>
  render(
    <MemoryRouter initialEntries={[entry]}>
      <NotificationsPage />
    </MemoryRouter>,
  );

describe("notificationDeepLink", () => {
  it("maps only known reference types with a UUID identifier", () => {
    expect(
      notificationDeepLink({
        reference_type: "ketidakhadiran",
        reference_id: "33333333-3333-4333-8333-333333333333",
      }),
    ).toBe("/app/persetujuan/ketidakhadiran/33333333-3333-4333-8333-333333333333");
    expect(
      notificationDeepLink({
        reference_type: "lembur",
        reference_id: "33333333-3333-4333-8333-333333333333",
      }),
    ).toBe("/app/persetujuan/lembur/33333333-3333-4333-8333-333333333333");
  });

  // Server tidak pernah mengirim URL; nilai di luar katalog tidak boleh menjadi tautan.
  it("refuses unknown types, external values, and traversal attempts", () => {
    expect(notificationDeepLink({ reference_type: "gaji", reference_id: "33333333-3333-4333-8333-333333333333" })).toBeNull();
    expect(
      notificationDeepLink({ reference_type: "ketidakhadiran", reference_id: "https://jahat.test" }),
    ).toBeNull();
    expect(
      notificationDeepLink({ reference_type: "ketidakhadiran", reference_id: "../../admin" }),
    ).toBeNull();
    expect(notificationDeepLink({ reference_type: null, reference_id: null })).toBeNull();
  });

  // Karyawan tidak memiliki halaman detail karyawan, sehingga tautannya ditiadakan.
  it("keeps role-forbidden targets out of the link", () => {
    const contractNotice = {
      tipe: "kontrak_akan_habis",
      reference_type: "karyawan",
      reference_id: "44444444-4444-4444-8444-444444444444",
    };
    expect(notificationTargetForRole(contractNotice, "karyawan")).toBeNull();
    expect(notificationTargetForRole(contractNotice, "hr")).toBe(
      "/app/karyawan/44444444-4444-4444-8444-444444444444",
    );
  });

  it("sends decision notifications to the requester's own list", () => {
    const decision = {
      tipe: "keputusan_approve",
      reference_type: "ketidakhadiran",
      reference_id: "33333333-3333-4333-8333-333333333333",
    };
    expect(notificationTargetForRole(decision, "karyawan")).toBe("/app/pengajuan");
    expect(notificationTargetForRole(decision, "top_management")).toBeNull();
  });
});

describe("NotificationsPage", () => {
  beforeEach(() => {
    authState.current = {
      status: "authenticated",
      role: "atasan",
      user: { id: "11111111-1111-4111-8111-111111111111" },
    };
    listState.current = pageOf([notification()]);
    listParams.current = null;
    markPending.current = false;
    dismissPending.current = false;
    markMock.mockReset();
    markMock.mockResolvedValue({ id: "22222222-2222-4222-8222-222222222222" });
    dismissMock.mockReset();
    dismissMock.mockResolvedValue("22222222-2222-4222-8222-222222222222");
  });

  it("renders type, message, and read state as text", () => {
    renderPage();

    const list = screen.getByRole("list", { name: "Daftar notifikasi" });
    expect(within(list).getByText("Pengajuan ketidakhadiran")).toBeInTheDocument();
    expect(within(list).getByText("Pengajuan ketidakhadiran baru")).toBeInTheDocument();
    expect(within(list).getByText(/Belum dibaca/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Buka detail" })).toHaveAttribute(
      "href",
      "/app/persetujuan/ketidakhadiran/33333333-3333-4333-8333-333333333333",
    );
  });

  // Payload berisi markup harus tampil sebagai teks, tidak pernah dieksekusi.
  it("renders an HTML payload as plain text", () => {
    listState.current = pageOf([
      notification({
        judul: "<img src=x onerror=alert(1)>",
        pesan: "<script>window.__xss = true;</script>",
      }),
    ]);
    renderPage();

    expect(screen.getByText("<img src=x onerror=alert(1)>")).toBeInTheDocument();
    expect(screen.getByText("<script>window.__xss = true;</script>")).toBeInTheDocument();
    expect(document.querySelector("script")).toBeNull();
    expect(document.querySelector("img")).toBeNull();
    expect(window.__xss).toBeUndefined();
  });

  it("marks a notification read through the endpoint", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "Tandai dibaca" }));

    await waitFor(() => expect(markMock).toHaveBeenCalledWith("22222222-2222-4222-8222-222222222222"));
  });

  it("marks read when the deep link is opened", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("link", { name: "Buka detail" }));

    expect(markMock).toHaveBeenCalledWith("22222222-2222-4222-8222-222222222222");
  });

  it("does not offer mark read for an already read notification", () => {
    listState.current = pageOf([
      notification({ is_read: true, read_at: "2026-08-03T03:00:00Z" }),
    ]);
    renderPage();

    expect(screen.queryByRole("button", { name: "Tandai dibaca" })).not.toBeInTheDocument();
    const list = screen.getByRole("list", { name: "Daftar notifikasi" });
    expect(within(list).getByText(/Sudah dibaca/)).toBeInTheDocument();
  });

  // Dismiss permanen, sehingga memerlukan konfirmasi eksplisit.
  it("confirms before dismissing", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "Hapus" }));

    const dialog = screen.getByRole("alertdialog");
    expect(within(dialog).getByText(/tidak dapat dikembalikan/i)).toBeInTheDocument();
    expect(dismissMock).not.toHaveBeenCalled();

    await user.click(within(dialog).getByRole("button", { name: "Hapus" }));
    await waitFor(() =>
      expect(dismissMock).toHaveBeenCalledWith("22222222-2222-4222-8222-222222222222"),
    );
  });

  it("cancelling the dialog leaves the notification untouched", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "Hapus" }));
    await user.click(screen.getByRole("button", { name: "Batal" }));

    expect(dismissMock).not.toHaveBeenCalled();
    expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
  });

  // Setelah dismiss server tidak lagi mengembalikan baris tersebut; UI harus mengikuti data.
  it("keeps a dismissed notification out of the list after refetch", async () => {
    const user = userEvent.setup();
    const view = renderPage();

    await user.click(screen.getByRole("button", { name: "Hapus" }));
    await user.click(screen.getByRole("alertdialog").querySelector("button:last-of-type"));
    await waitFor(() => expect(dismissMock).toHaveBeenCalled());

    listState.current = pageOf([]);
    view.rerender(
      <MemoryRouter initialEntries={["/app/notifikasi"]}>
        <NotificationsPage />
      </MemoryRouter>,
    );

    expect(screen.queryByText("Pengajuan ketidakhadiran baru")).not.toBeInTheDocument();
    expect(screen.getByText(/belum ada notifikasi/i)).toBeInTheDocument();
  });

  // Pesan kegagalan muncul sekali, di dalam dialog yang tetap terbuka.
  it("explains a forbidden dismiss without leaving the row in a fake state", async () => {
    dismissMock.mockRejectedValue({ status: 403, message: "raw" });
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "Hapus" }));
    await user.click(screen.getByRole("alertdialog").querySelector("button:last-of-type"));

    const dialog = await screen.findByRole("alertdialog");
    expect(within(dialog).getByText(/tidak dapat diakses lagi/i)).toBeInTheDocument();
    expect(screen.getAllByText(/tidak dapat diakses lagi/i)).toHaveLength(1);
    const list = screen.getByRole("list", { name: "Daftar notifikasi" });
    expect(within(list).getByText("Pengajuan ketidakhadiran baru")).toBeInTheDocument();
  });

  it("filters unread through the URL", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "Belum dibaca" }));

    await waitFor(() => expect(listParams.current?.is_read).toBe(false));
  });

  it("reads the active filter from the URL", () => {
    renderPage("/app/notifikasi?status=sudah");

    expect(listParams.current?.is_read).toBe(true);
    expect(screen.getByRole("button", { name: "Sudah dibaca" })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
  });

  it("shows loading, empty, and error states", () => {
    listState.current = { data: undefined, isPending: true, isError: false };
    const loading = renderPage();
    expect(screen.getByRole("status")).toHaveTextContent(/memuat notifikasi/i);
    loading.unmount();

    listState.current = pageOf([]);
    const empty = renderPage();
    expect(screen.getByText(/belum ada notifikasi/i)).toBeInTheDocument();
    empty.unmount();

    listState.current = {
      data: undefined,
      isPending: false,
      isError: true,
      error: { message: "Layanan tidak tersedia." },
      refetch: vi.fn(),
    };
    renderPage();
    expect(screen.getByRole("alert")).toHaveTextContent("Layanan tidak tersedia.");
  });

  it("shows a safe note when a notification has no reachable target", () => {
    listState.current = pageOf([
      notification({ tipe: "kontrak_akan_habis", reference_type: "karyawan" }),
    ]);
    renderPage();

    expect(screen.getByText(/tidak ada halaman detail/i)).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Buka detail" })).not.toBeInTheDocument();
  });
});
