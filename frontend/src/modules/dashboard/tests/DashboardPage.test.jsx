import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  dashboardMetricsFixture,
  employeeDetailFixture,
} from "../../employees/tests/employee-fixtures";
import { DashboardPage } from "../pages/DashboardPage";

const { authState, detailState, metricsState, receivedFilters } = vi.hoisted(() => ({
  authState: { current: { role: "hr", user: { id: "user-1" } } },
  detailState: { current: {} },
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

vi.mock("../../employees/hooks/useEmployees", () => ({
  useEmployeeDetail: () => detailState.current,
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
    detailState.current = {
      data: employeeDetailFixture,
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

  it("shows active department composition without the inactive breakdown", () => {
    renderPage();

    expect(
      screen.getByRole("heading", { name: "Komposisi departemen — aktif" }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("heading", { name: "Komposisi departemen — nonaktif" }),
    ).not.toBeInTheDocument();
  });

  it("shows the belum_diisi gender category explicitly", () => {
    renderPage();

    const genderList = screen.getByRole("list", { name: /populasi aktif menurut gender/i });
    expect(within(genderList).getByText("Belum diisi")).toBeInTheDocument();
    expect(within(genderList).getByText("Laki-laki")).toBeInTheDocument();
    expect(within(genderList).getByText("Perempuan")).toBeInTheDocument();
  });

  it("renders gender icons with an accessible text alternative", () => {
    renderPage();

    const genderList = screen.getByRole("list", { name: /populasi aktif menurut gender/i });
    expect(within(genderList).getByRole("listitem", { name: /Belum diisi: 1/ })).toBeInTheDocument();
  });

  it("renders the org chart hierarchy", () => {
    renderPage();

    const chartContent = document.getElementById("organization-chart-content");
    expect(chartContent).toHaveClass("organization-tree", "overflow-x-auto");
    expect(chartContent.querySelector(".organization-tree__roots")).toBeInTheDocument();
    expect(screen.getByText("Anita Sintetis")).toBeInTheDocument();
    expect(screen.getByText("Budi Sintetis")).toBeInTheDocument();
    expect(screen.getByText("Anita Sintetis").closest(".organization-tree__card")).toHaveStyle({
      "--organization-department-color": "hsl(214 68% 45%)",
    });
    expect(screen.getByText(/2 karyawan aktif dalam 1 jalur pelaporan teratas/)).toBeInTheDocument();
  });

  it("orders the official CEO branches and separates coordination lines", async () => {
    const employee = (id, nama, departemen, jabatan, bawahan = []) => ({
      employee_id: id,
      nama,
      departemen,
      jabatan,
      bawahan,
    });
    metricsState.current = {
      ...metricsState.current,
      data: {
        ...dashboardMetricsFixture,
        organization_chart: [
          employee(
            "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
            "Akun Uji Terpisah",
            "Pengujian",
            "Staff",
          ),
          employee(
            "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
            "Anthony Hamonangan Sihombing",
            "Director",
            "CEO",
            [
              employee("11111111-1111-4111-8111-111111111111", "Beniqno Joe Prasetyo Siilitonga", "Human Resources", "Head HR"),
              employee("22222222-2222-4222-8222-222222222222", "Dewa Pambudhi", "Operations", "Head Operations"),
              employee("33333333-3333-4333-8333-333333333333", "Mesianti Puspawardhany", "Legal & Governance", "Head Legal"),
              employee("44444444-4444-4444-8444-444444444444", "Nadya Theresia Sihombing", "CEO Office", "Head CEO Office"),
              employee("55555555-5555-4555-8555-555555555555", "Rachmat Efendi", "Finance", "Head Finance"),
            ],
          ),
        ],
      },
    };

    renderPage();

    const directReports = document.querySelector(
      ".organization-tree__roots > .organization-tree__member > .organization-tree__children",
    );
    expect(
      [...directReports.children].map((item) =>
        item.querySelector(":scope > .organization-tree__card p").textContent,
      ),
    ).toEqual([
      "Rachmat Efendi",
      "Nadya Theresia Sihombing",
      "Mesianti Puspawardhany",
      "Dewa Pambudhi",
      "Beniqno Joe Prasetyo Siilitonga",
    ]);
    expect(screen.queryByText("Akun Uji Terpisah")).not.toBeInTheDocument();
    await waitFor(() => {
      expect(document.querySelectorAll(".organization-tree__coordination path")).toHaveLength(1);
    });
    expect(screen.getByText("Garis putus-putus menunjukkan koordinasi.")).toBeInTheDocument();
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

  it("explains that valid attendance requires both clock-in and clock-out", () => {
    renderPage();

    expect(screen.getByText("Memiliki clock-in dan clock-out")).toBeInTheDocument();
  });

  it("opens an employee detail dialog from every organization card", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "Lihat detail Anita Sintetis" }));

    const dialog = screen.getByRole("dialog", { name: "Anita Sintetis" });
    expect(within(dialog).getByText("Identitas dan pekerjaan")).toBeInTheDocument();
    expect(within(dialog).getByText("UJI-001")).toBeInTheDocument();
    expect(within(dialog).getByText("anita@example.test")).toBeInTheDocument();
    expect(within(dialog).getByText("Perempuan")).toBeInTheDocument();
    expect(within(dialog).getByText("Teknologi")).toBeInTheDocument();

    await user.click(within(dialog).getByRole("button", { name: "Tutup detail karyawan" }));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
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
