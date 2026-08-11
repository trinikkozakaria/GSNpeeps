import { useState } from "react";

import { FileField } from "../../../components/form/FileField";
import { Button } from "../../../components/ui/Button";
import { formatDate } from "../../../lib/format";
import { useEmployeeDocuments, useUploadEmployeeDocument } from "../hooks/useEmployees";
import { DetailSection, EmptyNote } from "./DetailSection";

// Format yang disetujui kontrak. Arsip ZIP dan RAR sengaja tidak termasuk; validasi client
// hanya untuk UX karena backend tetap menjadi otoritas.
const allowedExtensions = [
  ".pdf",
  ".jpg",
  ".jpeg",
  ".png",
  ".doc",
  ".docx",
  ".xls",
  ".xlsx",
  ".ppt",
  ".pptx",
];

export const maxDocumentBytes = 5 * 1024 * 1024;

export const validateDocumentFile = (file) => {
  if (!file) return "Berkas dokumen wajib dipilih.";
  const extension = file.name.slice(file.name.lastIndexOf(".")).toLowerCase();
  if (!file.name.includes(".") || !allowedExtensions.includes(extension)) {
    return "Format berkas tidak didukung. Gunakan PDF, JPG, PNG, DOC, DOCX, XLS, XLSX, PPT, atau PPTX.";
  }
  if (file.size > maxDocumentBytes) {
    return "Ukuran berkas melebihi batas 5 MB.";
  }
  if (file.size === 0) {
    return "Berkas dokumen kosong.";
  }
  return "";
};

const uploadErrorMessage = (error) => {
  if (error?.status === 413) return "Ukuran berkas melebihi batas 5 MB.";
  if (error?.status === 415) {
    return "Format berkas ditolak server. Gunakan PDF, JPG, PNG, DOC, DOCX, XLS, XLSX, PPT, atau PPTX.";
  }
  if (error?.status === 403) return "Anda tidak memiliki akses untuk mengunggah dokumen.";
  if (error?.status === 404) return "Karyawan tidak ditemukan atau sudah dinonaktifkan.";
  return error?.message ?? "Dokumen belum dapat diunggah.";
};

export const EmployeeDocuments = ({ scope, employeeId, canUpload }) => {
  const documents = useEmployeeDocuments(scope, employeeId);
  const upload = useUploadEmployeeDocument(scope, employeeId);
  const [documentType, setDocumentType] = useState("");
  const [file, setFile] = useState(null);
  const [fieldError, setFieldError] = useState("");
  const [typeError, setTypeError] = useState("");
  const [formError, setFormError] = useState("");
  const [successMessage, setSuccessMessage] = useState("");

  const handleSubmit = async (event) => {
    event.preventDefault();
    setFormError("");
    setSuccessMessage("");

    const trimmedType = documentType.trim();
    const nextTypeError = trimmedType ? "" : "Jenis dokumen wajib diisi.";
    const nextFileError = validateDocumentFile(file);
    setTypeError(nextTypeError);
    setFieldError(nextFileError);
    if (nextTypeError || nextFileError) return;

    try {
      await upload.mutateAsync({ jenisDokumen: trimmedType, file });
      setDocumentType("");
      setFile(null);
      setSuccessMessage("Dokumen berhasil diunggah.");
    } catch (error) {
      setFormError(uploadErrorMessage(error));
    }
  };

  return (
    <DetailSection
      title="Dokumen karyawan"
      description="Dokumen tersimpan di penyimpanan berkas terkontrol; sistem hanya menyimpan tautannya."
    >
      {documents.isPending && <p role="status">Memuat dokumen…</p>}
      {documents.isError && (
        <div role="alert" className="rounded-lg border border-rose-300/30 bg-rose-300/10 p-3 text-rose-700">
          <p>Dokumen belum dapat dimuat. {documents.error.message}</p>
          <Button className="mt-3" variant="secondary" onClick={() => documents.refetch()}>
            Coba lagi
          </Button>
        </div>
      )}
      {documents.data && documents.data.length === 0 && (
        <EmptyNote>Belum ada dokumen yang diunggah.</EmptyNote>
      )}
      {documents.data && documents.data.length > 0 && (
        <ul className="divide-y divide-slate-900/10">
          {documents.data.map((document) => (
            <li key={document.id} className="grid gap-2 py-4 sm:grid-cols-3">
              <span className="font-semibold text-slate-900">{document.jenis_dokumen}</span>
              <a
                href={document.file_url}
                target="_blank"
                rel="noopener noreferrer"
                className="font-medium text-cyan-700 underline hover:text-cyan-900"
              >
                {document.nama_file}
                <span className="sr-only"> (buka di tab baru)</span>
              </a>
              <span className="text-slate-600">{formatDate(document.created_at.slice(0, 10))}</span>
            </li>
          ))}
        </ul>
      )}

      {canUpload && (
        <form onSubmit={handleSubmit} noValidate className="mt-6 grid gap-4 border-t border-slate-900/10 pt-6">
          <h3 className="text-base font-semibold text-slate-900">Unggah dokumen baru</h3>
          {formError && (
            <p role="alert" className="rounded-lg border border-rose-300/30 bg-rose-300/10 p-3 text-sm text-rose-700">
              {formError}
            </p>
          )}
          {successMessage && (
            <p role="status" className="rounded-lg border border-emerald-300/30 bg-emerald-300/10 p-3 text-sm text-emerald-700">
              {successMessage}
            </p>
          )}
          <div>
            <label htmlFor="document-type" className="mb-2 block text-sm font-medium text-slate-700">
              Jenis dokumen
            </label>
            <input
              id="document-type"
              value={documentType}
              maxLength={100}
              aria-invalid={Boolean(typeError)}
              aria-describedby={typeError ? "document-type-error" : undefined}
              onChange={(event) => setDocumentType(event.target.value)}
              disabled={upload.isPending}
              className="min-h-11 w-full rounded-lg border border-slate-900/15 bg-white px-3 text-slate-900 outline-none focus:border-cyan-300 disabled:opacity-60"
            />
            {typeError && (
              <p id="document-type-error" role="alert" className="mt-2 text-sm text-rose-700">
                {typeError}
              </p>
            )}
          </div>
          <FileField
            id="document-file"
            label="Berkas dokumen"
            accept={allowedExtensions.join(",")}
            description="Maksimal 5 MB. Format: PDF, JPG, PNG, DOC, DOCX, XLS, XLSX, PPT, PPTX."
            file={file}
            error={fieldError}
            disabled={upload.isPending}
            onFileChange={(selected) => {
              setFile(selected);
              setFieldError(selected ? validateDocumentFile(selected) : "");
            }}
          />
          <div>
            <Button type="submit" disabled={upload.isPending}>
              {upload.isPending ? "Mengunggah…" : "Unggah dokumen"}
            </Button>
          </div>
        </form>
      )}
    </DetailSection>
  );
};
