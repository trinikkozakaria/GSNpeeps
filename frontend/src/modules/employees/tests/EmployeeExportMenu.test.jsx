import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { EmployeeExportMenu } from "../components/EmployeeExportMenu";

const { exportMock } = vi.hoisted(() => ({ exportMock: vi.fn() }));

vi.mock("../api/employee-api", () => ({
  exportEmployeesRequest: exportMock,
}));

const createObjectURL = vi.fn(() => "blob:mock-url");
const revokeObjectURL = vi.fn();

describe("EmployeeExportMenu", () => {
  beforeEach(() => {
    exportMock.mockReset();
    createObjectURL.mockClear();
    revokeObjectURL.mockClear();
    vi.stubGlobal("URL", { ...URL, createObjectURL, revokeObjectURL });
    exportMock.mockResolvedValue({
      blob: new Blob(["konten"], { type: "application/pdf" }),
      fileName: "karyawan-20260801.xlsx",
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("sends the current filters and the requested format", async () => {
    const user = userEvent.setup();
    render(
      <EmployeeExportMenu
        filters={{ search: "anita", department_id: "dept-1", status: "aktif" }}
      />,
    );

    await user.click(screen.getByRole("button", { name: "XLSX" }));

    await waitFor(() => expect(exportMock).toHaveBeenCalledTimes(1));
    expect(exportMock).toHaveBeenCalledWith({
      format: "xlsx",
      id: undefined,
      search: "anita",
      department_id: "dept-1",
      status: "aktif",
    });
  });

  it("exports a single employee when an id is given", async () => {
    const user = userEvent.setup();
    render(<EmployeeExportMenu employeeId="employee-1" />);

    await user.click(screen.getByRole("button", { name: "PDF" }));

    await waitFor(() => expect(exportMock).toHaveBeenCalledTimes(1));
    expect(exportMock.mock.calls[0][0]).toMatchObject({ format: "pdf", id: "employee-1" });
  });

  it("revokes the object URL after the download is triggered", async () => {
    const user = userEvent.setup();
    render(<EmployeeExportMenu />);

    await user.click(screen.getByRole("button", { name: "XLSX" }));

    await waitFor(() => expect(createObjectURL).toHaveBeenCalledTimes(1));
    expect(revokeObjectURL).toHaveBeenCalledTimes(1);
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:mock-url");
    // Tidak ada anchor sementara yang tertinggal di dokumen.
    expect(document.querySelectorAll("a[download]")).toHaveLength(0);
  });

  it("reports an empty filter result without exposing a raw error", async () => {
    exportMock.mockRejectedValue({ status: 404, message: "Data tidak ditemukan" });
    const user = userEvent.setup();
    render(<EmployeeExportMenu filters={{ search: "tidak-ada" }} />);

    await user.click(screen.getByRole("button", { name: "XLSX" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Tidak ada data karyawan yang cocok dengan filter saat ini.",
    );
    expect(revokeObjectURL).not.toHaveBeenCalled();
  });

  it("reports a forbidden export attempt", async () => {
    exportMock.mockRejectedValue({ status: 403, message: "raw" });
    const user = userEvent.setup();
    render(<EmployeeExportMenu />);

    await user.click(screen.getByRole("button", { name: "PDF" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Anda tidak memiliki akses untuk mengekspor data karyawan.",
    );
  });
});
