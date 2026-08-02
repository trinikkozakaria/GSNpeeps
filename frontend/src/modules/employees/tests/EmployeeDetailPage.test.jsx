import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { EmployeeDetailPage } from "../pages/EmployeeDetailPage";
import { employeeDetailFixture } from "./employee-fixtures";

const { authState, detailState, deactivateMock, documentsState, uploadMock } = vi.hoisted(() => ({
  authState: { current: { role: "hr", user: { id: "user-1" } } },
  detailState: { current: {} },
  deactivateMock: vi.fn(),
  documentsState: { current: {} },
  uploadMock: vi.fn(),
}));

vi.mock("../../auth/hooks/useAuth", () => ({
  useAuth: () => authState.current,
}));

vi.mock("../hooks/useEmployees", () => ({
  useEmployeeDetail: () => detailState.current,
  useDeactivateEmployee: () => ({ mutateAsync: deactivateMock, isPending: false }),
  useEmployeeDocuments: () => documentsState.current,
  useUploadEmployeeDocument: () => ({ mutateAsync: uploadMock, isPending: false }),
}));

const renderPage = () =>
  render(
    <MemoryRouter initialEntries={["/app/karyawan/employee-1"]}>
      <Routes>
        <Route path="/app/karyawan/:id" element={<EmployeeDetailPage />} />
      </Routes>
    </MemoryRouter>,
  );

describe("EmployeeDetailPage", () => {
  beforeEach(() => {
    deactivateMock.mockReset();
    deactivateMock.mockResolvedValue({});
    authState.current = { role: "hr", user: { id: "user-1" } };
    detailState.current = {
      data: employeeDetailFixture,
      isPending: false,
      isError: false,
    };
    documentsState.current = { data: [], isPending: false, isError: false };
  });

  it("renders every detail section from the contract response", () => {
    renderPage();

    expect(screen.getByRole("heading", { name: "Identitas dan pekerjaan" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Alamat" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Kontrak" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "BPJS dan NPWP" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Kontak darurat" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Pendidikan" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Riwayat jabatan" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Gaji bulan berjalan" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Dokumen karyawan" })).toBeInTheDocument();
  });

  it("shows only the current month salary period", () => {
    renderPage();

    expect(screen.getByText("Agustus 2026")).toBeInTheDocument();
    expect(
      screen.getByText(/riwayat gaji penuh tidak tersedia di sini/i),
    ).toBeInTheDocument();
  });

  it("gives HR the mutation and export controls", () => {
    renderPage();

    expect(screen.getByRole("link", { name: "Edit" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Nonaktifkan" })).toBeInTheDocument();
    expect(screen.getByRole("group", { name: "Export karyawan ini" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Unggah dokumen" })).toBeInTheDocument();
  });

  it("hides every mutation control from Top Management", () => {
    authState.current = { role: "top_management", user: { id: "user-2" } };
    renderPage();

    expect(screen.queryByRole("link", { name: "Edit" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Nonaktifkan" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Unggah dokumen" })).not.toBeInTheDocument();
    expect(screen.queryByRole("group", { name: /export/i })).not.toBeInTheDocument();
    // Bagian dokumen tetap dapat dibaca sesuai akses read-only.
    expect(screen.getByRole("heading", { name: "Dokumen karyawan" })).toBeInTheDocument();
  });

  it("explains that deactivation is a soft delete before confirming", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "Nonaktifkan" }));

    const dialog = await screen.findByRole("alertdialog");
    expect(dialog).toHaveTextContent(/bukan dihapus permanen/i);
    expect(dialog).toHaveTextContent(/data dan riwayat tetap tersimpan/i);
    expect(deactivateMock).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Nonaktifkan karyawan" }));
    expect(deactivateMock).toHaveBeenCalledTimes(1);
  });

  it("cancels deactivation without calling the API", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "Nonaktifkan" }));
    await user.click(screen.getByRole("button", { name: "Batal" }));

    expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
    expect(deactivateMock).not.toHaveBeenCalled();
  });

  it("does not leak record existence on a forbidden response", () => {
    detailState.current = {
      data: undefined,
      isPending: false,
      isError: true,
      error: { status: 403, message: "Anda tidak memiliki akses" },
    };
    renderPage();

    expect(screen.getByRole("alert")).toHaveTextContent(
      "Data karyawan tidak ditemukan atau tidak dapat diakses dengan hak akses Anda.",
    );
  });
});
