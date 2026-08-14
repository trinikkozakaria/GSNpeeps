import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DocumentTypesPage } from "../pages/DocumentTypesPage";

const { createMock, createState, typesState } = vi.hoisted(() => ({
  createMock: vi.fn(),
  createState: { current: {} },
  typesState: { current: {} },
}));

vi.mock("@tanstack/react-query", async (importOriginal) => ({
  ...(await importOriginal()),
  useQuery: () => typesState.current,
  useMutation: () => ({ mutate: createMock, ...createState.current }),
}));

describe("DocumentTypesPage", () => {
  beforeEach(() => {
    createMock.mockReset();
    createState.current = { isPending: false, isError: false };
    typesState.current = { data: [], isPending: false, isError: false };
  });

  it("shows loading, error, and empty list states", () => {
    typesState.current = { isPending: true, isError: false };
    const { rerender } = render(<DocumentTypesPage />);
    expect(screen.getByRole("status", { name: "" })).toHaveTextContent("Memuat jenis dokumen");

    typesState.current = { isPending: false, isError: true };
    rerender(<DocumentTypesPage />);
    expect(screen.getByRole("alert")).toHaveTextContent("belum dapat dimuat");

    typesState.current = { data: [], isPending: false, isError: false };
    rerender(<DocumentTypesPage />);
    expect(screen.getByText("Belum ada jenis dokumen.")).toBeInTheDocument();
  });

  it("trims fields and sends the required flag", async () => {
    const user = userEvent.setup();
    render(<DocumentTypesPage />);

    const submit = screen.getByRole("button", { name: "Tambah" });
    expect(submit).toBeDisabled();

    await user.type(screen.getByLabelText("Kode"), "  KTP  ");
    await user.type(screen.getByLabelText("Nama"), "  Foto KTP  ");
    await user.click(screen.getByLabelText("Wajib"));
    await user.click(submit);

    expect(createMock).toHaveBeenCalledWith({
      kode: "KTP",
      nama: "Foto KTP",
      wajib: false,
    });
  });

  it("disables submission and explains a create failure", () => {
    createState.current = { isPending: true, isError: true };
    render(<DocumentTypesPage />);

    expect(screen.getByRole("button", { name: "Menambahkan…" })).toBeDisabled();
    expect(screen.getByRole("alert")).toHaveTextContent("kode atau nama sudah digunakan");
  });

  it("renders required and optional document types", () => {
    typesState.current = {
      isPending: false,
      isError: false,
      data: [
        { id: "type-1", kode: "KTP", nama: "Foto KTP", wajib: true },
        { id: "type-2", kode: "NPWP", nama: "NPWP", wajib: false },
      ],
    };
    render(<DocumentTypesPage />);

    expect(screen.getByText("Foto KTP")).toBeInTheDocument();
    expect(screen.getAllByText("Wajib")).toHaveLength(2);
    expect(screen.getByText("Opsional")).toBeInTheDocument();
  });
});
