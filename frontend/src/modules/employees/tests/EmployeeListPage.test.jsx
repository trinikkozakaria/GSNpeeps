import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { EmployeeListPage } from "../pages/EmployeeListPage";

const { authState, employeesState, receivedFilters } = vi.hoisted(() => ({
  authState: { current: { role: "hr", user: { id: "user-1" } } },
  employeesState: { current: {} },
  receivedFilters: { current: null },
}));

vi.mock("../../auth/hooks/useAuth", () => ({
  useAuth: () => authState.current,
}));

vi.mock("../hooks/useEmployees", () => ({
  useDepartments: () => ({ data: [{ id: "dept-1", nama: "Teknologi" }] }),
  useEmployees: (scope, filters) => {
    receivedFilters.current = filters;
    return employeesState.current;
  },
  useBulkEmployees: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("../api/employee-api", () => ({ exportEmployeesRequest: vi.fn() }));

const page = (items, meta) => ({
  data: { items, meta },
  isPending: false,
  isError: false,
});

const employeeRow = {
  id: "11111111-1111-4111-8111-111111111111",
  nip: "UJI-001",
  nama: "Anita Sintetis",
  email: "anita@example.test",
  departemen: "Teknologi",
  jabatan: "Staff",
  status: "aktif",
};

const renderPage = (entry = "/app/karyawan") =>
  render(
    <MemoryRouter initialEntries={[entry]}>
      <EmployeeListPage />
    </MemoryRouter>,
  );

describe("EmployeeListPage", () => {
  beforeEach(() => {
    authState.current = { role: "hr", user: { id: "user-1" } };
    employeesState.current = page([employeeRow], {
      page: 1,
      limit: 10,
      total_data: 1,
      total_page: 1,
    });
    receivedFilters.current = null;
  });

  it("reads filters and page from the URL", () => {
    renderPage("/app/karyawan?search=anita&department_id=dept-1&status=aktif&page=3");

    expect(receivedFilters.current).toMatchObject({
      search: "anita",
      department_id: "dept-1",
      status: "aktif",
      page: 3,
    });
  });

  it("resets the page when a filter changes", async () => {
    const user = userEvent.setup();
    renderPage("/app/karyawan?page=4");

    await user.selectOptions(screen.getByLabelText("Status"), "nonaktif");

    await waitFor(() => expect(receivedFilters.current.status).toBe("nonaktif"));
    expect(receivedFilters.current.page).toBe(1);
  });

  it("labels search with the fields the contract actually searches", () => {
    renderPage();

    expect(screen.getByLabelText("Cari nama atau NIP")).toBeInTheDocument();
  });

  it("uses the server pagination meta instead of counting client-side", () => {
    employeesState.current = page([employeeRow], {
      page: 2,
      limit: 10,
      total_data: 134,
      total_page: 14,
    });
    renderPage();

    expect(screen.getByText("134 karyawan ditemukan")).toBeInTheDocument();
    expect(screen.getByText(/Halaman 2 dari 14/)).toBeInTheDocument();
  });

  it("gives HR the create and export controls", () => {
    renderPage();

    expect(screen.getByRole("link", { name: "Tambah karyawan" })).toBeInTheDocument();
    expect(screen.getByRole("group", { name: "Export hasil filter" })).toBeInTheDocument();
  });

  it("hides mutation and export controls from Top Management", () => {
    authState.current = { role: "top_management", user: { id: "user-2" } };
    renderPage();

    expect(screen.queryByRole("link", { name: "Tambah karyawan" })).not.toBeInTheDocument();
    expect(screen.queryByRole("group", { name: /export/i })).not.toBeInTheDocument();
    // Daftar tetap dapat dibaca.
    expect(screen.getAllByText("Anita Sintetis").length).toBeGreaterThan(0);
  });

  it("distinguishes an empty result from a filtered empty result", () => {
    employeesState.current = page([], { page: 1, limit: 10, total_data: 0, total_page: 0 });
    const { unmount } = renderPage();
    expect(screen.getByText("Belum ada data karyawan.")).toBeInTheDocument();
    unmount();

    renderPage("/app/karyawan?search=tidak-ada");
    expect(screen.getByText("Tidak ada karyawan yang cocok dengan filter.")).toBeInTheDocument();
  });

  it("shows a retryable error state", () => {
    employeesState.current = {
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
