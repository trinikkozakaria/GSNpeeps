import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AuditLogPage } from "../pages/AuditLogPage";
import { formatAuditDetail, redactAuditDetail, redactedPlaceholder } from "../utils/redact-detail";

const { authState, logsState, logsParams, logsEnabled } = vi.hoisted(() => ({
  authState: { current: {} },
  logsState: { current: {} },
  logsParams: { current: null },
  logsEnabled: { current: null },
}));

vi.mock("../../auth/hooks/useAuth", () => ({ useAuth: () => authState.current }));

vi.mock("../hooks/useAuditLogs", () => ({
  useAuditLogs: (_scope, params, enabled) => {
    logsParams.current = params;
    logsEnabled.current = enabled;
    return logsState.current;
  },
}));

const actorId = "11111111-1111-4111-8111-111111111111";

const entry = (overrides = {}) => ({
  id: "22222222-2222-4222-8222-222222222222",
  user_id: actorId,
  nama_user: "HR Uji",
  aksi: "APPROVE",
  modul: "ketidakhadiran",
  resource_id: "33333333-3333-4333-8333-333333333333",
  detail: { status_baru: "disetujui" },
  ip_address: "203.0.113.10",
  created_at: "2026-08-03T02:00:00Z",
  ...overrides,
});

const pageOf = (items) => ({
  data: { items, meta: { page: 1, limit: 10, total_data: items.length, total_page: 1 } },
  isPending: false,
  isError: false,
  refetch: vi.fn(),
});

const identity = (role) => ({ status: "authenticated", role, user: { id: "user-1" } });

const renderPage = (path = "/app/audit") =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <AuditLogPage />
    </MemoryRouter>,
  );

describe("redactAuditDetail", () => {
  it("hides sensitive values even if the server sends them", () => {
    const redacted = redactAuditDetail({
      password_hash: "$argon2id$",
      access_token: "abc.def",
      gaji_pokok: 9_500_000,
      nomor_rekening: "1234567890",
      status_baru: "disetujui",
      sebelum: { npwp: "12.345", modul: "akses" },
    });

    expect(redacted.password_hash).toBe(redactedPlaceholder);
    expect(redacted.access_token).toBe(redactedPlaceholder);
    expect(redacted.gaji_pokok).toBe(redactedPlaceholder);
    expect(redacted.nomor_rekening).toBe(redactedPlaceholder);
    expect(redacted.sebelum.npwp).toBe(redactedPlaceholder);

    // Field operasional tetap terbaca agar audit tetap berguna.
    expect(redacted.status_baru).toBe("disetujui");
    expect(redacted.sebelum.modul).toBe("akses");
  });

  it("formats detail as a JSON string, never as markup", () => {
    const formatted = formatAuditDetail({ pesan: "<script>alert(1)</script>" });

    expect(typeof formatted).toBe("string");
    // Nilainya tetap ada apa adanya; yang menjadikannya aman adalah dirender di dalam `pre`.
    expect(formatted).toContain("<script>alert(1)</script>");
    expect(formatAuditDetail(null)).toBe("");
    expect(formatAuditDetail({})).toBe("");
  });
});

describe("AuditLogPage", () => {
  beforeEach(() => {
    authState.current = identity("hr");
    logsState.current = pageOf([entry()]);
    logsParams.current = null;
    logsEnabled.current = null;
  });

  it("shows the log for HR and Top Management", () => {
    for (const role of ["hr", "top_management"]) {
      authState.current = identity(role);
      const view = renderPage();

      expect(logsEnabled.current).toBe(true);
      // DataTable merender tabel dan daftar kartu dari data yang sama untuk perilaku responsif.
      expect(screen.getAllByText("APPROVE").length).toBeGreaterThan(0);
      expect(screen.queryByRole("button", { name: /hapus|ubah/i })).not.toBeInTheDocument();
      view.unmount();
    }
  });

  // Karyawan dan Atasan tidak boleh memicu permintaan audit.
  it("does not fetch for Karyawan or Atasan", () => {
    for (const role of ["karyawan", "atasan"]) {
      authState.current = identity(role);
      const view = renderPage();

      expect(logsEnabled.current).toBe(false);
      expect(screen.getByRole("alert")).toHaveTextContent(/hanya tersedia untuk HR/i);
      view.unmount();
    }
  });

  it("reads filters and pagination from the URL", () => {
    renderPage(
      `/app/audit?tanggal_mulai=2026-08-01&tanggal_selesai=2026-08-31&modul=akses&aksi=PERMISSION_UPDATE&user_id=${actorId}&page=3`,
    );

    expect(logsParams.current).toMatchObject({
      tanggal_mulai: "2026-08-01",
      tanggal_selesai: "2026-08-31",
      modul: "akses",
      aksi: "PERMISSION_UPDATE",
      user_id: actorId,
      page: 3,
    });
  });

  // Nilai user_id yang bukan UUID tidak boleh dikirim sebagai filter.
  it("ignores a malformed actor filter and says so", () => {
    renderPage("/app/audit?user_id=bukan-uuid");

    expect(logsParams.current?.user_id).toBeUndefined();
    expect(screen.getByRole("alert")).toHaveTextContent(/harus berupa UUID/i);
  });

  it("writes a changed filter into the URL and resets the page", async () => {
    const user = userEvent.setup();
    renderPage("/app/audit?page=4");

    await user.type(screen.getByLabelText("Modul"), "akses");

    await waitFor(() => expect(logsParams.current?.modul).toBe("akses"));
    expect(logsParams.current?.page).toBe(1);
  });

  it("filters by the actor of a row", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getAllByRole("button", { name: "Filter aktor" })[0]);

    await waitFor(() => expect(logsParams.current?.user_id).toBe(actorId));
  });

  // Aktor sistem tidak boleh ditampilkan sebagai pengguna.
  it("labels a system actor without inventing a user", () => {
    logsState.current = pageOf([entry({ user_id: null, nama_user: null, aksi: "AUTO_ESCALATE" })]);
    renderPage();

    expect(screen.getAllByText("Sistem").length).toBeGreaterThan(0);
    expect(screen.queryByRole("button", { name: "Filter aktor" })).not.toBeInTheDocument();
  });

  it("renders detail as text inside a disclosure", async () => {
    logsState.current = pageOf([
      entry({ detail: { password_hash: "rahasia", status_baru: "disetujui" } }),
    ]);
    const user = userEvent.setup();
    renderPage();

    const toggle = screen.getAllByRole("button", { name: "Lihat detail" })[0];
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    await user.click(toggle);

    const detail = (await screen.findAllByText(/status_baru/))[0];
    expect(detail.tagName).toBe("PRE");
    // Redaksi client tetap berlaku meski backend keliru mengirim nilai sensitif.
    expect(detail.textContent).toContain(redactedPlaceholder);
    expect(detail.textContent).not.toContain("rahasia");
  });

  it("shows loading, empty, and error states", () => {
    logsState.current = { data: undefined, isPending: true, isError: false, refetch: vi.fn() };
    const loading = renderPage();
    expect(screen.getByRole("status")).toHaveTextContent(/memuat audit log/i);
    loading.unmount();

    logsState.current = pageOf([]);
    const empty = renderPage();
    expect(screen.getByText(/tidak ada aktivitas/i)).toBeInTheDocument();
    empty.unmount();

    logsState.current = {
      data: undefined,
      isPending: false,
      isError: true,
      error: { message: "Layanan tidak tersedia." },
      refetch: vi.fn(),
    };
    renderPage();
    expect(screen.getByRole("alert")).toHaveTextContent("Layanan tidak tersedia.");
  });
});
