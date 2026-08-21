import { useState } from "react";

import { Button } from "../../../components/ui/Button";
import { saveBlob } from "../../../lib/api/download";
import { exportEmployeesRequest } from "../api/employee-api";

const exportErrorMessage = (error) => {
  if (error?.status === 403) return "Anda tidak memiliki akses untuk mengekspor data karyawan.";
  if (error?.status === 404) return "Tidak ada data karyawan yang cocok dengan filter saat ini.";
  return error?.message ?? "Export belum dapat diproses.";
};

export const EmployeeExportMenu = ({
  filters,
  employeeId,
  label = "Export data",
  buttonSize = "default",
  labelClassName = "",
}) => {
  const [busyFormat, setBusyFormat] = useState("");
  const [error, setError] = useState("");
  const [status, setStatus] = useState("");

  const runExport = async (format) => {
    setBusyFormat(format);
    setError("");
    setStatus("");
    try {
      const { blob, fileName } = await exportEmployeesRequest({
        format,
        id: employeeId,
        search: filters?.search,
        department_id: filters?.department_id,
        status: filters?.status,
      });
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
      <div className="flex flex-wrap items-center gap-2" role="group" aria-label={label}>
        <span className={`text-sm text-slate-500 ${labelClassName}`}>{label}:</span>
        <Button size={buttonSize} variant="secondary" disabled={Boolean(busyFormat)} onClick={() => runExport("xlsx")}>
          {busyFormat === "xlsx" ? "Menyiapkan XLSX…" : "XLSX"}
        </Button>
        <Button size={buttonSize} variant="secondary" disabled={Boolean(busyFormat)} onClick={() => runExport("pdf")}>
          {busyFormat === "pdf" ? "Menyiapkan PDF…" : "PDF"}
        </Button>
      </div>
      {status && <p aria-live="polite" className="mt-2 text-sm text-slate-500">{status}</p>}
      {error && (
        <p role="alert" className="mt-2 text-sm text-rose-700">
          {error}
        </p>
      )}
    </div>
  );
};
