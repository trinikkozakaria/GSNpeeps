import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DocumentTypesPage } from "../pages/DocumentTypesPage";

const {
  typesState,
  mutationFlags,
  createRequest,
  updateRequest,
  deleteRequest,
  invalidateQueries,
} = vi.hoisted(() => ({
  typesState: { current: {} },
  // Flag per-mutation ("create" | "update" | "deactivate") supaya tombol yang di-render
  // dapat dibuat pending/error secara terpisah.
  mutationFlags: { current: {} },
  createRequest: vi.fn(),
  updateRequest: vi.fn(),
  deleteRequest: vi.fn(),
  invalidateQueries: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("../api/uat-api", () => ({
  documentTypesRequest: vi.fn(),
  createDocumentTypeRequest: (...args) => createRequest(...args),
  updateDocumentTypeRequest: (...args) => updateRequest(...args),
  deleteDocumentTypeRequest: (...args) => deleteRequest(...args),
}));

vi.mock("../../../lib/query/query-client", () => ({
  queryClient: { invalidateQueries },
}));

// useMutation nyata butuh provider; di sini kita jalankan mutationFn lalu onSuccess
// sinkron, dan menandai identitas mutation lewat mutationKey yang di-set halaman.
vi.mock("@tanstack/react-query", async (importOriginal) => ({
  ...(await importOriginal()),
  useQuery: () => typesState.current,
  useMutation: (options) => {
    const key = options.mutationKey?.[0];
    return {
      mutate: async (vars) => {
        await options.mutationFn(vars);
        await options.onSuccess?.();
      },
      isPending: Boolean(mutationFlags.current[key]?.isPending),
      isError: Boolean(mutationFlags.current[key]?.isError),
    };
  },
}));

const rows = [
  { id: "type-1", kode: "KTP", nama: "Foto KTP", wajib: true, is_active: true },
  { id: "type-2", kode: "NPWP", nama: "NPWP", wajib: false, is_active: true },
  { id: "type-3", kode: "SIM", nama: "SIM Lama", wajib: false, is_active: false },
];

describe("DocumentTypesPage", () => {
  beforeEach(() => {
    createRequest.mockReset().mockResolvedValue({});
    updateRequest.mockReset().mockResolvedValue({});
    deleteRequest.mockReset().mockResolvedValue({});
    invalidateQueries.mockClear();
    mutationFlags.current = {};
    typesState.current = { data: [], isPending: false, isError: false };
  });

  it("shows loading, error, and empty list states", () => {
    typesState.current = { isPending: true, isError: false };
    const { rerender } = render(<DocumentTypesPage />);
    expect(screen.getByRole("status")).toHaveTextContent("Memuat jenis dokumen");

    typesState.current = { isPending: false, isError: true };
    rerender(<DocumentTypesPage />);
    expect(screen.getByRole("alert")).toHaveTextContent("belum dapat dimuat");

    typesState.current = { data: [], isPending: false, isError: false };
    rerender(<DocumentTypesPage />);
    expect(screen.getByText("Belum ada jenis dokumen.")).toBeInTheDocument();
  });

  it("trims fields and sends the required flag on create", async () => {
    const user = userEvent.setup();
    render(<DocumentTypesPage />);

    const submit = screen.getByRole("button", { name: "Tambah" });
    expect(submit).toBeDisabled();

    await user.type(screen.getByLabelText("Kode"), "  KTP  ");
    await user.type(screen.getByLabelText("Nama"), "  Foto KTP  ");
    await user.click(screen.getByLabelText("Wajib")); // default true -> false
    await user.click(submit);

    expect(createRequest).toHaveBeenCalledWith({ kode: "KTP", nama: "Foto KTP", wajib: false });
    expect(updateRequest).not.toHaveBeenCalled();
    expect(deleteRequest).not.toHaveBeenCalled();
  });

  it("disables submission and explains a create failure", () => {
    mutationFlags.current = { create: { isPending: true, isError: true } };
    render(<DocumentTypesPage />);

    expect(screen.getByRole("button", { name: "Menambahkan…" })).toBeDisabled();
    expect(screen.getByRole("alert")).toHaveTextContent("kode atau nama sudah digunakan");
  });

  it("renders required, optional, and deactivated document types", () => {
    typesState.current = { isPending: false, isError: false, data: rows };
    render(<DocumentTypesPage />);

    const list = screen.getByRole("list");
    expect(within(list).getByText("Foto KTP")).toBeInTheDocument();

    const ktpRow = within(list).getByText("Foto KTP").closest("li");
    expect(within(ktpRow).getByText("Wajib")).toBeInTheDocument();

    const npwpRow = within(list).getByText("NPWP").closest("li");
    expect(within(npwpRow).getByText("Opsional")).toBeInTheDocument();

    const simRow = within(list).getByText("SIM Lama").closest("li");
    expect(within(simRow).getByText(/Nonaktif/)).toBeInTheDocument();
  });

  it("edits a row and calls the update endpoint with id and payload", async () => {
    const user = userEvent.setup();
    typesState.current = { isPending: false, isError: false, data: rows };
    render(<DocumentTypesPage />);

    const ktpRow = screen.getByText("Foto KTP").closest("li");
    await user.click(within(ktpRow).getByRole("button", { name: "Edit" }));

    const nameField = within(ktpRow).getByLabelText("Nama");
    await user.clear(nameField);
    await user.type(nameField, "Kartu Tanda Penduduk");
    await user.click(within(ktpRow).getByRole("button", { name: "Simpan" }));

    expect(updateRequest).toHaveBeenCalledTimes(1);
    expect(updateRequest).toHaveBeenCalledWith("type-1", {
      kode: "KTP",
      nama: "Kartu Tanda Penduduk",
      wajib: true,
      is_active: true,
    });
    expect(createRequest).not.toHaveBeenCalled();
  });

  it("deactivates an active row through the delete endpoint", async () => {
    const user = userEvent.setup();
    typesState.current = { isPending: false, isError: false, data: rows };
    render(<DocumentTypesPage />);

    const npwpRow = screen.getByText("NPWP").closest("li");
    await user.click(within(npwpRow).getByRole("button", { name: "Nonaktifkan" }));

    expect(deleteRequest).toHaveBeenCalledWith("type-2");
    expect(updateRequest).not.toHaveBeenCalled();
  });

  it("offers no deactivate control on an already inactive row", () => {
    typesState.current = { isPending: false, isError: false, data: rows };
    render(<DocumentTypesPage />);

    const simRow = screen.getByText("SIM Lama").closest("li");
    expect(within(simRow).queryByRole("button", { name: "Nonaktifkan" })).not.toBeInTheDocument();
    // Edit tetap tersedia untuk mengaktifkan kembali.
    expect(within(simRow).getByRole("button", { name: "Edit" })).toBeInTheDocument();
  });
});
