import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";

import { DataTable } from "../../../components/data-table/DataTable";
import { Pagination } from "../../../components/data-table/Pagination";
import { CheckboxMultiSelect } from "../../../components/form/CheckboxMultiSelect";
import { Button } from "../../../components/ui/Button";
import { formatNumber } from "../../../lib/format";
import { useAuth } from "../../auth/hooks/useAuth";
import { useDepartments } from "../../employees/hooks/useEmployees";
import { ReportExportMenu } from "../components/ReportExportMenu";
import { useAttendanceReport } from "../hooks/useReports";

const monitoringRoles = ["hr", "top_management"];

export const AttendanceReportPage = () => {
  document.title = "Laporan Kehadiran — GSNpeeps";
  const auth = useAuth();
  const [params, setParams] = useSearchParams();
  const canRead = monitoringRoles.includes(auth.role);
  const [nameInput, setNameInput] = useState(params.get("nama") ?? "");

  const filters = useMemo(
    () => ({
      nama: params.get("nama") || undefined,
      periode: params.get("periode") || undefined,
      tanggal_mulai: params.get("tanggal_mulai") || undefined,
      tanggal_selesai: params.get("tanggal_selesai") || undefined,
      department_id: params.getAll("department_id"),
      page: Number.parseInt(params.get("page") ?? "1", 10) || 1,
      limit: 10,
    }),
    [params],
  );

  const departments = useDepartments();
  const report = useAttendanceReport(auth.role, filters, canRead);

  const setFilter = (key, value) => {
    const next = new URLSearchParams(params);
    if (value) next.set(key, value);
    else next.delete(key);
    if (key !== "page") next.delete("page");
    setParams(next);
  };

  const setDepartments = (ids) => {
    const next = new URLSearchParams(params);
    next.delete("department_id");
    ids.forEach((id) => next.append("department_id", id));
    next.delete("page");
    setParams(next);
  };

  // Debounce input nama agar tidak memicu fetch per ketikan; state page direset.
  useEffect(() => {
    const normalized = nameInput.trim();
    const timer = window.setTimeout(() => {
      setParams((current) => {
        if (normalized === (current.get("nama") ?? "")) return current;
        const next = new URLSearchParams(current);
        if (normalized) next.set("nama", normalized);
        else next.delete("nama");
        next.delete("page");
        return next;
      }, { replace: true });
    }, 350);
    return () => window.clearTimeout(timer);
  }, [nameInput, setParams]);

  const columns = [
    { key: "nama", header: "Karyawan", render: (row) => row.nama_karyawan },
    { key: "departemen", header: "Departemen", render: (row) => row.departemen || "—" },
    { key: "hadir", header: "Hadir", render: (row) => formatNumber(row.hadir) },
    { key: "terlambat", header: "Terlambat", render: (row) => formatNumber(row.terlambat) },
    { key: "izin", header: "Izin", render: (row) => formatNumber(row.izin) },
  ];

  return (
    <section aria-labelledby="report-title">
      <p className="text-sm font-semibold uppercase tracking-widest text-cyan-700">Monitoring</p>
      <h1 id="report-title" className="mt-2 text-3xl font-bold">Laporan Kehadiran</h1>
      <p className="mt-2 text-slate-600">
        Rekap dihitung server pada zona waktu Asia/Jakarta.
        {auth.role === "top_management" && " Akses Anda bersifat pemantauan saja."}
      </p>

      <div className="mt-7 grid gap-4 rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-4 sm:grid-cols-2 lg:grid-cols-4">
        <label className="text-sm font-medium text-slate-700">
          Cari nama karyawan
          <input
            type="search"
            value={nameInput}
            onChange={(event) => setNameInput(event.target.value)}
            placeholder="mis. Budi"
            className="mt-2 min-h-10 w-full rounded-lg border border-slate-900/15 bg-white px-3 text-slate-900 outline-none focus:border-cyan-300"
          />
        </label>
        <label className="text-sm font-medium text-slate-700">
          Periode (YYYY-MM)
          <input
            type="month"
            value={filters.periode ?? ""}
            onChange={(event) => setFilter("periode", event.target.value)}
            className="mt-2 min-h-10 w-full rounded-lg border border-slate-900/15 bg-white px-3 text-slate-900 outline-none focus:border-cyan-300"
          />
        </label>
        <label className="text-sm font-medium text-slate-700">
          Tanggal mulai
          <input
            type="date"
            value={filters.tanggal_mulai ?? ""}
            onChange={(event) => setFilter("tanggal_mulai", event.target.value)}
            className="mt-2 min-h-10 w-full rounded-lg border border-slate-900/15 bg-white px-3 text-slate-900 outline-none focus:border-cyan-300"
          />
        </label>
        <label className="text-sm font-medium text-slate-700">
          Tanggal selesai
          <input
            type="date"
            value={filters.tanggal_selesai ?? ""}
            onChange={(event) => setFilter("tanggal_selesai", event.target.value)}
            className="mt-2 min-h-10 w-full rounded-lg border border-slate-900/15 bg-white px-3 text-slate-900 outline-none focus:border-cyan-300"
          />
        </label>
        <CheckboxMultiSelect
          legend="Departemen"
          allLabel="Semua departemen"
          options={(departments.data ?? []).map((item) => ({
            value: item.id,
            label: item.nama,
          }))}
          selected={filters.department_id}
          onChange={setDepartments}
          emptyLabel="Memuat departemen…"
        />
      </div>

      {/* Export hanya tersedia bagi HR sesuai API Contract. */}
      {auth.role === "hr" && (
        <div className="mt-5">
          <ReportExportMenu filters={filters} />
        </div>
      )}

      <div className="mt-6" aria-live="polite">
        {report.isPending && <p role="status" className="text-slate-600">Memuat laporan…</p>}
        {report.isError && (
          <div role="alert" className="rounded-xl border border-red-400/30 bg-red-400/10 p-4 text-red-700">
            <p>Laporan belum dapat dimuat. {report.error.message}</p>
            <Button className="mt-3" variant="secondary" onClick={() => report.refetch()}>Coba lagi</Button>
          </div>
        )}
        {report.data && (
          <>
            <DataTable
              caption="Rekap kehadiran karyawan"
              columns={columns}
              rows={report.data.items}
              rowKey={(row) => row.employee_id}
              emptyMessage="Tidak ada data kehadiran pada filter ini."
            />
            {report.data.items.length > 0 && (
              <Pagination
                meta={report.data.meta}
                label="Navigasi halaman laporan"
                onPageChange={(page) => setFilter("page", String(page))}
              />
            )}
          </>
        )}
      </div>
    </section>
  );
};
