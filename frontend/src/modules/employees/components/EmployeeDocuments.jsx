import { useEffect, useState } from "react";

import { FileField } from "../../../components/form/FileField";
import { Button } from "../../../components/ui/Button";
import { formatDate } from "../../../lib/format";
import { useEmployeeDocuments, useUploadEmployeeDocument } from "../hooks/useEmployees";
import { DetailSection, EmptyNote } from "./DetailSection";
import { documentTypesRequest } from "../../uat/api/uat-api";
import { ProtectedDocumentPreview } from "../../../components/media/ProtectedImage";

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
  const [documentTypes, setDocumentTypes] = useState([]);
  const [documentTypesStatus, setDocumentTypesStatus] = useState("idle");
  useEffect(() => {
    if (!canUpload) return undefined;
    const controller = new AbortController();
    setDocumentTypesStatus("loading");
    documentTypesRequest(controller.signal)
      .then((items) => {
        setDocumentTypes(items.filter((item) => item.is_active));
        setDocumentTypesStatus("ready");
      })
      .catch((error) => {
        if (error?.code === "ERR_CANCELED") return;
        setDocumentTypes([]);
        setDocumentTypesStatus("error");
      });
    return () => controller.abort();
  }, [canUpload]);
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
    const typeIsActive = documentTypes.some((item) => item.nama === trimmedType);
    const nextTypeError = typeIsActive ? "" : "Pilih jenis dokumen aktif dari master data.";
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
      {documents.data && documentTypes.length > 0 && (
        <div className="mb-4 rounded-lg bg-slate-50 p-3">
          <p className="text-sm font-semibold">Kelengkapan dokumen wajib</p>
          {documentTypes.filter((item) => item.wajib).map((item) => {
            const uploaded = documents.data.some((document) => document.jenis_dokumen === item.nama);
            return <p key={item.id} className={`text-sm ${uploaded ? "text-emerald-700" : "text-red-700"}`}>{uploaded ? "✓" : "!"} {item.nama}</p>;
          })}
        </div>
      )}
      {documents.data && documents.data.length > 0 && (
        <ul className="divide-y divide-slate-900/10">
          {documents.data.map((document) => (
            <li key={document.id} className="grid gap-3 py-4 sm:grid-cols-[minmax(0,1fr)_minmax(0,2fr)_auto]">
              <span className="font-semibold text-slate-900">{document.jenis_dokumen}</span>
              <ProtectedDocumentPreview
                path={document.file_url}
                fileName={document.nama_file}
              />
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
            <select
              id="document-type"
              value={documentType}
              aria-invalid={Boolean(typeError)}
              aria-describedby={typeError ? "document-type-error" : undefined}
              onChange={(event) => setDocumentType(event.target.value)}
              disabled={upload.isPending || documentTypesStatus !== "ready"}
              className="min-h-10 w-full rounded-lg border border-slate-900/15 bg-white px-3 text-slate-900 outline-none focus:border-cyan-300 disabled:opacity-60"
            >
              <option value="">
                {documentTypesStatus === "loading" ? "Memuat jenis dokumen…" : "Pilih jenis dokumen"}
              </option>
              {documentTypes.map((item) => <option key={item.id} value={item.nama}>{item.nama}</option>)}
            </select>
            {documentTypesStatus === "error" && (
              <p role="alert" className="mt-2 text-sm text-rose-700">
                Master jenis dokumen belum dapat dimuat. Coba muat ulang halaman.
              </p>
            )}
            {documentTypesStatus === "ready" && documentTypes.length === 0 && (
              <p role="alert" className="mt-2 text-sm text-amber-700">
                Belum ada jenis dokumen aktif. Tambahkan melalui Master Jenis Dokumen.
              </p>
            )}
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
            <Button type="submit" disabled={upload.isPending || documentTypesStatus !== "ready" || documentTypes.length === 0}>
              {upload.isPending ? "Mengunggah…" : "Unggah dokumen"}
            </Button>
          </div>
        </form>
      )}
    </DetailSection>
  );
};
