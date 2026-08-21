import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { RoleLandingPage } from "../pages/RoleLandingPage";

const { summaryState } = vi.hoisted(() => ({ summaryState: { current: {} } }));

vi.mock("@tanstack/react-query", async (importOriginal) => ({
  ...(await importOriginal()),
  useQuery: () => summaryState.current,
}));

vi.mock("../hooks/useAuth", () => ({
  useAuth: () => ({
    role: "karyawan",
    user: { id: "user-1", nama: "Karyawan Sintetis", email: "karyawan@example.test" },
  }),
}));

vi.mock("../../uat/components/CompanyFeedInfiniteList", () => ({
  CompanyFeedInfiniteList: () => <p>Daftar company feed</p>,
}));

describe("RoleLandingPage", () => {
  beforeEach(() => {
    summaryState.current = {
      data: {
        pengajuan_perlu_disetujui: 2,
        pengajuan_ketidakhadiran_pribadi: 3,
        saldo_cuti: [{ jenis: "Cuti Tahunan", sisa: 8 }],
      },
      isPending: false,
      isError: false,
    };
  });

  it("renders every requested home summary and company feed", () => {
    render(<RoleLandingPage />);

    expect(screen.getByText("Perlu disetujui").parentElement).toHaveClass("bg-slate-900/[0.03]");
    expect(screen.getByText("Ketidakhadiran pribadi").parentElement).toHaveClass("bg-slate-900/[0.03]");
    expect(screen.getByText("8")).toHaveClass("text-3xl", "font-bold");
    expect(screen.getByText("8").parentElement).toHaveClass("mt-1", "items-baseline");
    expect(screen.getByText("hari · Cuti Tahunan")).toHaveClass("text-sm");
    expect(screen.getByText("Daftar company feed")).toBeInTheDocument();
  });

  it("keeps the summary visible while loading", () => {
    summaryState.current = { isPending: true, isError: false };
    render(<RoleLandingPage />);
    expect(screen.getByRole("status")).toHaveTextContent("Memuat ringkasan beranda");
  });

  it("shows a retry action when the summary fails", () => {
    summaryState.current = {
      isPending: false,
      isError: true,
      error: { message: "Layanan tidak tersedia." },
      refetch: vi.fn(),
    };
    render(<RoleLandingPage />);
    expect(screen.getByRole("alert")).toHaveTextContent("Layanan tidak tersedia");
    expect(screen.getByRole("button", { name: "Coba lagi" })).toBeInTheDocument();
  });
});
