import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { AttendanceReportPage } from "../pages/AttendanceReportPage";
import { LiveFeedPage } from "../pages/LiveFeedPage";

const { authState, reportState, feedState, exportMock, reportEnabled, feedEnabled } = vi.hoisted(
  () => ({
    authState: { current: { role: "hr", user: { id: "user-1" } } },
    reportState: { current: {} },
    feedState: { current: {} },
    exportMock: vi.fn(),
    reportEnabled: { current: null },
    feedEnabled: { current: null },
  }),
);

vi.mock("../../auth/hooks/useAuth", () => ({ useAuth: () => authState.current }));
vi.mock("../../employees/hooks/useEmployees", () => ({
  useDepartments: () => ({ data: [] }),
}));
vi.mock("../hooks/useReports", () => ({
  useAttendanceReport: (_scope, _params, enabled) => {
    reportEnabled.current = enabled;
    return reportState.current;
  },
}));
vi.mock("../../attendance/hooks/useAttendance", () => ({
  useLiveFeed: (_scope, _tanggal, enabled) => {
    feedEnabled.current = enabled;
    return feedState.current;
  },
}));
vi.mock("../api/report-api", () => ({ exportAttendanceReportRequest: exportMock }));

const createObjectURL = vi.fn(() => "blob:mock-url");
const revokeObjectURL = vi.fn();

const reportRow = {
  employee_id: "11111111-1111-4111-8111-111111111111",
  nama_karyawan: "Karyawan Uji",
  departemen: "Teknologi",
  hadir: 18,
  terlambat: 2,
  izin: 1,
  alpha: 0,
};

describe("AttendanceReportPage", () => {
  beforeEach(() => {
    authState.current = { role: "hr", user: { id: "user-1" } };
    reportState.current = {
      data: { items: [reportRow], meta: { page: 1, limit: 10, total_data: 1, total_page: 1 } },
      isPending: false,
      isError: false,
    };
    exportMock.mockReset();
    exportMock.mockResolvedValue({
      blob: new Blob(["x"]),
      fileName: "laporan-kehadiran.xlsx",
    });
    createObjectURL.mockClear();
    revokeObjectURL.mockClear();
    vi.stubGlobal("URL", { ...URL, createObjectURL, revokeObjectURL });
  });

  afterEach(() => vi.unstubAllGlobals());

  const renderPage = (entry = "/app/laporan-kehadiran") =>
    render(
      <MemoryRouter initialEntries={[entry]}>
        <AttendanceReportPage />
      </MemoryRouter>,
    );

  it("shows the export menu to HR only", () => {
    renderPage();
    expect(screen.getByRole("group", { name: "Export laporan" })).toBeInTheDocument();
  });

  it("hides the export menu from Top Management", () => {
    authState.current = { role: "top_management", user: { id: "user-2" } };
    renderPage();

    expect(screen.queryByRole("group", { name: "Export laporan" })).not.toBeInTheDocument();
    expect(screen.getByText(/pemantauan saja/i)).toBeInTheDocument();
    // Data laporan tetap terbaca oleh Top Management.
    expect(screen.getAllByText("Karyawan Uji").length).toBeGreaterThan(0);
  });

  it("revokes the object URL after the export download", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "XLSX" }));

    await waitFor(() => expect(createObjectURL).toHaveBeenCalledTimes(1));
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:mock-url");
    expect(document.querySelectorAll("a[download]")).toHaveLength(0);
  });

  it("reports an empty export result without a raw error", async () => {
    exportMock.mockRejectedValue({ status: 404, message: "raw" });
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "PDF" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(/tidak ada data laporan/i);
  });

  it("reads filters from the URL", () => {
    renderPage("/app/laporan-kehadiran?periode=2026-07&page=2");

    expect(screen.getByLabelText(/periode/i)).toHaveValue("2026-07");
  });
});

describe("LiveFeedPage", () => {
  beforeEach(() => {
    feedEnabled.current = null;
    feedState.current = { data: [], isPending: false, isError: false };
  });

  const renderFeed = () =>
    render(
      <MemoryRouter initialEntries={["/app/live-feed"]}>
        <LiveFeedPage />
      </MemoryRouter>,
    );

  it("fetches for HR and Top Management", () => {
    authState.current = { role: "hr", user: { id: "user-1" } };
    renderFeed();
    expect(feedEnabled.current).toBe(true);
  });

  // Karyawan dan Atasan tidak boleh memicu permintaan data seluruh organisasi.
  it("does not fetch for employees or supervisors", () => {
    for (const role of ["karyawan", "atasan"]) {
      authState.current = { role, user: { id: "user-9" } };
      const view = renderFeed();
      expect(feedEnabled.current).toBe(false);
      view.unmount();
    }
  });

  it("lazy loads attendance photos with a descriptive alt text", () => {
    authState.current = { role: "hr", user: { id: "user-1" } };
    feedState.current = {
      data: [
        {
          id: "22222222-2222-4222-8222-222222222222",
          employee_id: "33333333-3333-4333-8333-333333333333",
          nama_karyawan: "Karyawan Uji",
          departemen: "Teknologi",
          tanggal: "2026-08-03",
          tipe: "check_in",
          mode_kerja: "WFO",
          waktu: "2026-08-03T02:00:00Z",
          status: "tepat_waktu",
          foto_url: "https://files.example.test/absensi.jpg",
        },
      ],
      isPending: false,
      isError: false,
    };
    renderFeed();

    const photo = screen.getAllByAltText("Foto absensi Karyawan Uji")[0];
    expect(photo).toHaveAttribute("loading", "lazy");
  });
});
