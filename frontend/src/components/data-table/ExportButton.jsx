import { useState } from "react";

import { Button } from "../ui/Button";
import { saveBlob } from "../../lib/api/download";

const defaultErrorMessage = (error, emptyMessage, forbiddenMessage) => {
  if (error?.status === 403) return forbiddenMessage;
  if (error?.status === 404) return emptyMessage;
  return error?.message ?? "Export belum dapat diproses.";
};

/**
 * Tombol download report tunggal (mis. XLSX) untuk halaman yang hanya menyediakan satu
 * format export, berbeda dari ReportExportMenu/EmployeeExportMenu yang menawarkan pilihan
 * format. `exportRequest` mengembalikan `{ blob, fileName }` seperti helper `downloadFile`.
 */
export const ExportButton = ({
  label = "Download Excel",
  exportRequest,
  emptyMessage = "Tidak ada data untuk diunduh pada filter saat ini.",
  forbiddenMessage = "Anda tidak memiliki akses untuk mengunduh berkas ini.",
}) => {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [status, setStatus] = useState("");

  const runExport = async () => {
    setBusy(true);
    setError("");
    setStatus("");
    try {
      const { blob, fileName } = await exportRequest();
      saveBlob(blob, fileName);
      setStatus(`Berkas ${fileName} berhasil diunduh.`);
    } catch (exportError) {
      setError(defaultErrorMessage(exportError, emptyMessage, forbiddenMessage));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div>
      <Button variant="secondary" disabled={busy} onClick={runExport}>
        {busy ? "Menyiapkan berkas…" : label}
      </Button>
      <p aria-live="polite" className="mt-2 text-sm text-slate-500">{status}</p>
      {error && <p role="alert" className="mt-2 text-sm text-rose-700">{error}</p>}
    </div>
  );
};
