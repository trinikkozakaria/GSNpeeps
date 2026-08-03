import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { dashboardMetricsFixture } from "../../employees/tests/employee-fixtures";
import { DashboardPage } from "../pages/DashboardPage";

const { authState, metricsState, receivedFilters } = vi.hoisted(() => ({
  authState: { current: { role: "hr", user: { id: "user-1" } } },
  metricsState: { current: {} },
  receivedFilters: { current: null },
}));

vi.mock("../../auth/hooks/useAuth", () => ({
  useAuth: () => authState.current,
}));

vi.mock("../hooks/useDashboard", () => ({
  useDashboardMetrics: (scope, filters) => {
    receivedFilters.current = filters;
    return metricsState.current;
  },
}));

const renderPage = (initialEntry = "/app/dashboard") =>
  render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <DashboardPage />
    </MemoryRouter>,
  );

describe("DashboardPage", () => {
  beforeEach(() => {
    authState.current = { role: "hr", user: { id: "user-1" } };
    metricsState.current = {
      data: dashboardMetricsFixture,
      isPending: false,
      isError: false,
    };
    receivedFilters.current = null;
  });

  it("defaults to the monthly period", () => {
    renderPage();

    expect(receivedFilters.current).toEqual({ periode: "bulanan", tanggalAcuan: "" });
  });

  it("reads the period and anchor date from the URL", () => {
    renderPage("/app/dashboard?periode=mingguan&tanggal_acuan=2026-08-12");

    expect(receivedFilters.current).toEqual({
      periode: "mingguan",
      tanggalAcuan: "2026-08-12",
    });
  });

  it("ignores a period outside the contract enum", () => {
    renderPage("/app/dashboard?periode=triwulan");

    expect(receivedFilters.current.periode).toBe("bulanan");
  });

  it("offers exactly the four contract periods", async () => {
    const user = userEvent.setup();
    renderPage();
    const select = screen.getByLabelText("Periode");

    expect(within(select).getAllByRole("option")).toHaveLength(4);
    expect(screen.getByRole("option", { name: "Mingguan (Senin–Minggu)" })).toBeInTheDocument();

    await user.selectOptions(select, "tahunan");
    expect(receivedFilters.current.periode).toBe("tahunan");
  });

  it("shows the resolved period range and timezone", () => {
    renderPage();

    expect(
      screen.getByText(/1 Agustus 2026 — 31 Agustus 2026 \(Asia\/Jakarta\)/),
    ).toBeInTheDocument();
  });

  it("separates active and inactive department composition", () => {
    renderPage();

    expect(
      screen.getByRole("heading", { name: "Komposisi departemen — aktif" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Komposisi departemen — nonaktif" }),
    ).toBeInTheDocument();
  });

  it("shows the belum_diisi gender category explicitly", () => {
    renderPage();

    expect(screen.getByRole("rowheader", { name: "Belum diisi" })).toBeInTheDocument();
    expect(screen.getByRole("rowheader", { name: "Laki-laki" })).toBeInTheDocument();
    expect(screen.getByRole("rowheader", { name: "Perempuan" })).toBeInTheDocument();
  });

  it("renders charts as accessible tables rather than image-only graphics", () => {
    renderPage();

    const genderTable = screen.getByRole("table", { name: /populasi aktif menurut gender/i });
    expect(within(genderTable).getByRole("row", { name: /Belum diisi/ })).toHaveTextContent("1");
  });

  it("renders the org chart hierarchy", () => {
    renderPage();

    expect(screen.getByText("Anita Sintetis")).toBeInTheDocument();
    expect(screen.getByText("Budi Sintetis")).toBeInTheDocument();
    expect(screen.getByText(/2 karyawan aktif dalam 1 jalur pelaporan teratas/)).toBeInTheDocument();
  });

  it("formats currency and percentage in Indonesian locale", () => {
    renderPage();

    expect(screen.getByText("28,57%")).toBeInTheDocument();
    expect(screen.getByText(/Rp\s?21\.000\.000/)).toBeInTheDocument();
  });

  it("shows only the three approved Coming Soon modules", () => {
    renderPage();

    const comingSoon = screen.getAllByText("Coming Soon");
    expect(comingSoon).toHaveLength(3);
    expect(screen.getByRole("heading", { name: "Hiring Progress" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Recruitment Cost" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Benefit" })).toBeInTheDocument();
  });

  it("gives Top Management the same read-only view without mutation controls", () => {
    authState.current = { role: "top_management", user: { id: "user-2" } };
    renderPage();

    expect(screen.getByText(/akses Anda bersifat pemantauan saja/i)).toBeInTheDocument();
    expect(screen.getByText("Anita Sintetis")).toBeInTheDocument();
    const buttons = screen.queryAllByRole("button");
    buttons.forEach((button) => {
      expect(button).not.toHaveAccessibleName(/simpan|hapus|nonaktifkan|tambah|unggah/i);
    });
  });

  it("shows loading and retryable error states", () => {
    metricsState.current = { data: undefined, isPending: true, isError: false };
    const { unmount } = renderPage();
    expect(screen.getByRole("status")).toHaveTextContent("Memuat metrik dashboard…");
    unmount();

    metricsState.current = {
      data: undefined,
      isPending: false,
      isError: true,
      error: { message: "Layanan tidak tersedia." },
      refetch: vi.fn(),
    };
    renderPage();
    expect(screen.getByRole("alert")).toHaveTextContent("Layanan tidak tersedia.");
    expect(screen.getByRole("button", { name: "Coba lagi" })).toBeInTheDocument();
  });
});
