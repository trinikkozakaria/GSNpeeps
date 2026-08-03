import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { PersonalMetricsPage } from "../pages/PersonalMetricsPage";

const { metricsState } = vi.hoisted(() => ({ metricsState: { current: {} } }));

vi.mock("../../auth/hooks/useAuth", () => ({
  useAuth: () => ({ role: "karyawan", user: { id: "user-1" } }),
}));

vi.mock("../hooks/useProfile", () => ({
  usePersonalMetrics: () => metricsState.current,
}));

const emptyMetrics = {
  periode: "2026-08",
  hadir: 0,
  terlambat: 0,
  izin: 0,
  total_lembur_jam: 0,
  riwayat_absensi: [],
};

describe("PersonalMetricsPage", () => {
  beforeEach(() => {
    metricsState.current = { data: emptyMetrics, isPending: false, isError: false };
  });

  // Modul Kehadiran belum aktif; halaman harus jujur menampilkan nol, bukan angka contoh.
  it("shows honest zeroes and an explanation when attendance has no data yet", () => {
    render(<PersonalMetricsPage />);

    expect(screen.getByText("Periode Agustus 2026.")).toBeInTheDocument();
    expect(screen.getAllByText("0").length).toBeGreaterThanOrEqual(3);
    expect(screen.getByText(/belum ada aktivitas kehadiran yang tercatat/i)).toBeInTheDocument();
    expect(screen.getByText("Belum ada riwayat check-in pada periode ini.")).toBeInTheDocument();
  });

  it("renders real clock history when the API returns it", () => {
    metricsState.current = {
      data: {
        ...emptyMetrics,
        hadir: 2,
        terlambat: 1,
        riwayat_absensi: [
          { tanggal: "2026-08-03", check_in: "08:55", check_out: "18:02", status: "tepat_waktu" },
          { tanggal: "2026-08-04", check_in: "09:15", check_out: null, status: "terlambat" },
        ],
      },
      isPending: false,
      isError: false,
    };
    render(<PersonalMetricsPage />);

    expect(screen.queryByText(/belum ada aktivitas kehadiran/i)).not.toBeInTheDocument();
    expect(screen.getAllByText("08:55").length).toBeGreaterThan(0);
    expect(screen.getAllByText("terlambat").length).toBeGreaterThan(0);
  });

  it("shows the loading state", () => {
    metricsState.current = { data: undefined, isPending: true, isError: false };
    render(<PersonalMetricsPage />);

    expect(screen.getByRole("status")).toHaveTextContent("Memuat metrik personal…");
  });

  it("shows a retryable error state", () => {
    metricsState.current = {
      data: undefined,
      isPending: false,
      isError: true,
      error: { message: "Layanan tidak tersedia." },
      refetch: vi.fn(),
    };
    render(<PersonalMetricsPage />);

    expect(screen.getByRole("alert")).toHaveTextContent("Layanan tidak tersedia.");
  });
});
