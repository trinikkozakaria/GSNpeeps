import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { EmployeeDocuments, validateDocumentFile } from "../components/EmployeeDocuments";

const { documentsState, uploadMock, uploadPending } = vi.hoisted(() => ({
  documentsState: { current: {} },
  uploadMock: vi.fn(),
  uploadPending: { current: false },
}));

vi.mock("../hooks/useEmployees", () => ({
  useEmployeeDocuments: () => documentsState.current,
  useUploadEmployeeDocument: () => ({ mutateAsync: uploadMock, isPending: uploadPending.current }),
}));

const file = (name, sizeBytes, type = "application/pdf") => {
  const created = new File(["konten-sintetis"], name, { type });
  Object.defineProperty(created, "size", { value: sizeBytes });
  return created;
};

const renderDocuments = (canUpload = true) =>
  render(<EmployeeDocuments scope="hr" employeeId="employee-1" canUpload={canUpload} />);

describe("validateDocumentFile", () => {
  it("accepts every approved document format", () => {
    const approved = [
      "a.pdf", "a.jpg", "a.jpeg", "a.png",
      "a.doc", "a.docx", "a.xls", "a.xlsx", "a.ppt", "a.pptx",
    ];
    approved.forEach((name) => {
      expect(validateDocumentFile(file(name, 1024))).toBe("");
    });
  });

  it("rejects archives and unknown extensions", () => {
    expect(validateDocumentFile(file("arsip.zip", 1024))).toMatch(/format berkas tidak didukung/i);
    expect(validateDocumentFile(file("arsip.rar", 1024))).toMatch(/format berkas tidak didukung/i);
    expect(validateDocumentFile(file("skrip.sh", 1024))).toMatch(/format berkas tidak didukung/i);
    expect(validateDocumentFile(file("tanpaekstensi", 1024))).toMatch(/format berkas tidak didukung/i);
  });

  it("rejects files above the 5 MB contract limit", () => {
    expect(validateDocumentFile(file("besar.pdf", 5 * 1024 * 1024 + 1))).toMatch(/5 MB/);
    expect(validateDocumentFile(file("pas.pdf", 5 * 1024 * 1024))).toBe("");
  });
});

describe("EmployeeDocuments", () => {
  beforeEach(() => {
    uploadMock.mockReset();
    uploadMock.mockResolvedValue({});
    uploadPending.current = false;
    documentsState.current = { data: [], isPending: false, isError: false };
  });

  it("hides the upload form when the role cannot upload", () => {
    renderDocuments(false);

    expect(screen.queryByRole("button", { name: "Unggah dokumen" })).not.toBeInTheDocument();
  });

  it("requires a document type before uploading", async () => {
    const user = userEvent.setup();
    renderDocuments();

    await user.upload(screen.getByLabelText("Berkas dokumen"), file("ijazah.pdf", 2048));
    await user.click(screen.getByRole("button", { name: "Unggah dokumen" }));

    expect(await screen.findByText("Jenis dokumen wajib diisi.")).toBeInTheDocument();
    expect(uploadMock).not.toHaveBeenCalled();
  });

  it("blocks an unsupported file before sending it to the server", async () => {
    const user = userEvent.setup();
    renderDocuments();

    await user.type(screen.getByLabelText("Jenis dokumen"), "Ijazah");
    // Atribut accept hanya penyaring dialog; pengguna dapat memilih "All files" dan tetap
    // mengirim arsip. fireEvent dipakai agar penyaring tersebut terlewati sehingga yang
    // benar-benar diuji adalah validasi JavaScript sebagai penjaga terakhir di client.
    fireEvent.change(screen.getByLabelText("Berkas dokumen"), {
      target: { files: [file("arsip.zip", 2048, "application/zip")] },
    });
    await user.click(screen.getByRole("button", { name: "Unggah dokumen" }));

    expect(await screen.findByText(/format berkas tidak didukung/i)).toBeInTheDocument();
    expect(uploadMock).not.toHaveBeenCalled();
  });

  it("sends the contract multipart fields on a valid upload", async () => {
    const user = userEvent.setup();
    renderDocuments();
    const selected = file("ijazah.pdf", 2048);

    await user.type(screen.getByLabelText("Jenis dokumen"), "Ijazah");
    await user.upload(screen.getByLabelText("Berkas dokumen"), selected);
    await user.click(screen.getByRole("button", { name: "Unggah dokumen" }));

    expect(uploadMock).toHaveBeenCalledTimes(1);
    expect(uploadMock).toHaveBeenCalledWith({ jenisDokumen: "Ijazah", file: selected });
    expect(await screen.findByText("Dokumen berhasil diunggah.")).toBeInTheDocument();
  });

  it("maps 413 to a size message", async () => {
    uploadMock.mockRejectedValue({ status: 413, message: "raw" });
    const user = userEvent.setup();
    renderDocuments();

    await user.type(screen.getByLabelText("Jenis dokumen"), "Ijazah");
    await user.upload(screen.getByLabelText("Berkas dokumen"), file("ijazah.pdf", 2048));
    await user.click(screen.getByRole("button", { name: "Unggah dokumen" }));

    expect(await screen.findByText("Ukuran berkas melebihi batas 5 MB.")).toBeInTheDocument();
  });

  it("maps 415 to a format message", async () => {
    uploadMock.mockRejectedValue({ status: 415, message: "raw" });
    const user = userEvent.setup();
    renderDocuments();

    await user.type(screen.getByLabelText("Jenis dokumen"), "Ijazah");
    await user.upload(screen.getByLabelText("Berkas dokumen"), file("ijazah.pdf", 2048));
    await user.click(screen.getByRole("button", { name: "Unggah dokumen" }));

    expect(await screen.findByText(/format berkas ditolak server/i)).toBeInTheDocument();
  });

  it("prevents a duplicate submit while the upload is in flight", () => {
    uploadPending.current = true;
    renderDocuments();

    expect(screen.getByRole("button", { name: "Mengunggah…" })).toBeDisabled();
  });

  it("opens document links safely in a new tab", () => {
    documentsState.current = {
      data: [
        {
          id: "doc-1",
          jenis_dokumen: "Ijazah",
          nama_file: "ijazah.pdf",
          file_url: "https://files.example.test/doc/ijazah.pdf",
          created_at: "2026-08-01T10:00:00Z",
        },
      ],
      isPending: false,
      isError: false,
    };
    renderDocuments();

    const link = screen.getByRole("link", { name: /ijazah\.pdf/ });
    expect(link).toHaveAttribute("target", "_blank");
    expect(link).toHaveAttribute("rel", expect.stringContaining("noopener"));
  });
});
