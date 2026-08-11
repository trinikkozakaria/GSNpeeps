import { useState } from "react";

import { Button } from "../../../components/ui/Button";
import { exportAttendanceReportRequest } from "../api/report-api";

const exportErrorMessage = (error) => {
  if (error?.status === 403) return "Anda tidak memiliki akses untuk mengekspor laporan.";
  if (error?.status === 404) return "Tidak ada data laporan pada filter saat ini.";
  return error?.message ?? "Export belum dapat diproses.";
};

// Object URL dicabut pada finally sehingga blob tidak bertahan setelah unduhan dipicu.
const saveBlob = (blob, fileName) => {
  const objectUrl = URL.createObjectURL(blob);
  try {
    const link = document.createElement("a");
    link.href = objectUrl;
    link.download = fileName;
    link.rel = "noopener";
    document.body.append(link);
    link.click();
    link.remove();
  } finally {
    URL.revokeObjectURL(objectUrl);
  }
};

export const ReportExportMenu = ({ filters }) => {
  const [busyFormat, setBusyFormat] = useState("");
  const [error, setError] = useState("");
  const [status, setStatus] = useState("");

  const runExport = async (format) => {
    setBusyFormat(format);
    setError("");
    setStatus("");
    try {
      const { blob, fileName } = await exportAttendanceReportRequest({ ...filters, format });
      saveBlob(blob, fileName);
      setStatus(`Berkas ${fileName} berhasil diunduh.`);
    } catch (exportError) {
      setError(exportErrorMessage(exportError));
    } finally {
      setBusyFormat("");
    }
  };

  return (
    <div>
      <div className="flex flex-wrap items-center gap-2" role="group" aria-label="Export laporan">
        <span className="text-sm text-slate-500">Export laporan:</span>
        <Button variant="secondary" disabled={Boolean(busyFormat)} onClick={() => runExport("xlsx")}>
          {busyFormat === "xlsx" ? "Menyiapkan XLSX…" : "XLSX"}
        </Button>
        <Button variant="secondary" disabled={Boolean(busyFormat)} onClick={() => runExport("pdf")}>
          {busyFormat === "pdf" ? "Menyiapkan PDF…" : "PDF"}
        </Button>
      </div>
      <p aria-live="polite" className="mt-2 text-sm text-slate-500">{status}</p>
      {error && <p role="alert" className="mt-2 text-sm text-rose-700">{error}</p>}
    </div>
  );
};
