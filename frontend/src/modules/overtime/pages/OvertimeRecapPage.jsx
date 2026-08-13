import { useMemo } from "react";
import { useSearchParams } from "react-router-dom";

import { DataTable } from "../../../components/data-table/DataTable";
import { Button } from "../../../components/ui/Button";
import { formatNumber } from "../../../lib/format";
import { useAuth } from "../../auth/hooks/useAuth";
import { useDepartments } from "../../employees/hooks/useEmployees";
import { useOvertimeRecap } from "../hooks/useOvertime";

/**
 * Rekap lembur hanya untuk HR. Sistem tidak menghitung kompensasi maupun uang lembur;
 * perhitungan tersebut dilakukan manual di luar sistem sesuai PRD.
 */
export const OvertimeRecapPage = () => {
  document.title = "Rekap Lembur — GSNpeeps";
  const auth = useAuth();
  const [params, setParams] = useSearchParams();
  const isHR = auth.role === "hr";

  const filters = useMemo(
    () => ({
      tanggal_mulai: params.get("tanggal_mulai") || undefined,
      tanggal_selesai: params.get("tanggal_selesai") || undefined,
      department_id: params.get("department_id") || undefined,
    }),
    [params],
  );

  const departments = useDepartments();
  const recap = useOvertimeRecap(auth.role, filters, isHR);

  const setFilter = (key, value) => {
    const next = new URLSearchParams(params);
    if (value) next.set(key, value);
    else next.delete(key);
    setParams(next);
  };

  if (!isHR) {
    return (
      <section aria-labelledby="recap-title">
        <h1 id="recap-title" className="text-3xl font-bold">Rekap Lembur</h1>
        <p role="status" className="mt-3 text-slate-600">Rekap lembur hanya dapat diakses HR.</p>
      </section>
    );
  }

  const columns = [
    { key: "nama", header: "Karyawan", render: (row) => row.nama_karyawan },
    { key: "departemen", header: "Departemen", render: (row) => row.departemen || "—" },
    { key: "pengajuan", header: "Total pengajuan", render: (row) => formatNumber(row.total_pengajuan) },
    { key: "jam", header: "Total jam", render: (row) => `${row.total_jam} jam` },
  ];

  return (
    <section aria-labelledby="recap-title">
      <p className="text-sm font-semibold uppercase tracking-widest text-cyan-700">Monitoring</p>
      <h1 id="recap-title" className="mt-2 text-3xl font-bold">Rekap Lembur</h1>
      <p className="mt-2 max-w-2xl text-slate-600">
        Menampilkan durasi lembur yang telah disetujui. Perhitungan kompensasi dilakukan di
        luar sistem.
      </p>

      <div className="mt-7 grid gap-4 rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-4 sm:grid-cols-3">
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
        <label className="text-sm font-medium text-slate-700">
          Departemen
          <select
            value={filters.department_id ?? ""}
            onChange={(event) => setFilter("department_id", event.target.value)}
            className="mt-2 min-h-10 w-full rounded-lg border border-slate-900/15 bg-white px-3 text-slate-900 outline-none focus:border-cyan-300"
          >
            <option value="">Semua departemen</option>
            {(departments.data ?? []).map((item) => (
              <option key={item.id} value={item.id}>{item.nama}</option>
            ))}
          </select>
        </label>
      </div>

      <div className="mt-6" aria-live="polite">
        {recap.isPending && <p role="status" className="text-slate-600">Memuat rekap…</p>}
        {recap.isError && (
          <div role="alert" className="rounded-xl border border-red-400/30 bg-red-400/10 p-4 text-red-700">
            <p>Rekap belum dapat dimuat. {recap.error.message}</p>
            <Button className="mt-3" variant="secondary" onClick={() => recap.refetch()}>Coba lagi</Button>
          </div>
        )}
        {recap.data && (
          <DataTable
            caption="Rekap lembur disetujui"
            columns={columns}
            rows={recap.data}
            rowKey={(row) => row.employee_id}
            emptyMessage="Belum ada lembur disetujui pada filter ini."
          />
        )}
      </div>
    </section>
  );
};
