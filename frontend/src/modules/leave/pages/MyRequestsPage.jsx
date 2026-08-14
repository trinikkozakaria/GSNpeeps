import { useMemo } from "react";
import { useSearchParams } from "react-router-dom";

import { DataTable } from "../../../components/data-table/DataTable";
import { Pagination } from "../../../components/data-table/Pagination";
import { Button } from "../../../components/ui/Button";
import { formatDate } from "../../../lib/format";
import { useAuth } from "../../auth/hooks/useAuth";
import { RequestStatusBadge } from "../../approvals/components/RequestStatusBadge";
import { useMyOvertimeRequests } from "../../overtime/hooks/useOvertime";
import { useMyLeaveRequests } from "../hooks/useLeave";

/**
 * Histori pengajuan milik pemohon sendiri. Halaman ini tidak pernah menampilkan kontrol
 * keputusan karena pemohon bukan approver atas pengajuannya sendiri.
 */
export const MyRequestsPage = () => {
  document.title = "Pengajuan Saya — GSNpeeps";
  const auth = useAuth();
  const [params, setParams] = useSearchParams();
  const tab = params.get("tab") === "lembur" ? "lembur" : "ketidakhadiran";
  const page = Number.parseInt(params.get("page") ?? "1", 10) || 1;
  const filters = useMemo(() => ({ page, limit: 10 }), [page]);

  const leaves = useMyLeaveRequests(auth.user?.id, filters);
  const overtimes = useMyOvertimeRequests(auth.user?.id, filters, tab === "lembur");
  const active = tab === "lembur" ? overtimes : leaves;

  const setTab = (nextTab) => {
    const next = new URLSearchParams(params);
    next.set("tab", nextTab);
    next.delete("page");
    setParams(next);
  };

  const leaveColumns = [
    { key: "jenis", header: "Jenis izin", render: (row) => row.jenis_izin },
    {
      key: "periode",
      header: "Periode",
      render: (row) =>
        `${formatDate(row.tanggal_mulai)} — ${formatDate(row.tanggal_selesai)} (${row.jumlah_hari} hari)`,
    },
    { key: "status", header: "Status", render: (row) => <RequestStatusBadge status={row.status} /> },
  ];

  const overtimeColumns = [
    { key: "tanggal", header: "Tanggal", render: (row) => formatDate(row.tanggal) },
    {
      key: "waktu",
      header: "Waktu",
      render: (row) => `${row.waktu_mulai}–${row.waktu_selesai} (${row.total_jam} jam)`,
    },
    { key: "status", header: "Status", render: (row) => <RequestStatusBadge status={row.status} /> },
  ];

  return (
    <section aria-labelledby="my-requests-title">
      <p className="text-sm font-semibold uppercase tracking-widest text-cyan-700">Absensi</p>
      <h1 id="my-requests-title" className="mt-2 text-3xl font-bold">
        Pengajuan Saya
      </h1>

      <div role="tablist" aria-label="Jenis pengajuan" className="mt-6 flex flex-wrap gap-2">
        {[
          { id: "ketidakhadiran", label: "Ketidakhadiran" },
          { id: "lembur", label: "Lembur" },
        ].map((item) => (
          <Button
            key={item.id}
            role="tab"
            aria-selected={tab === item.id}
            onClick={() => setTab(item.id)}
            variant={tab === item.id ? "primary" : "secondary"}
          >
            {item.label}
          </Button>
        ))}
      </div>

      <div className="mt-6" aria-live="polite">
        {active.isPending && <p role="status" className="text-slate-600">Memuat pengajuan…</p>}
        {active.isError && (
          <div role="alert" className="rounded-xl border border-red-400/30 bg-red-400/10 p-4 text-red-700">
            <p>Pengajuan belum dapat dimuat. {active.error.message}</p>
            <Button className="mt-3" variant="secondary" onClick={() => active.refetch()}>
              Coba lagi
            </Button>
          </div>
        )}
        {active.data && (
          <>
            <DataTable
              caption={tab === "lembur" ? "Pengajuan lembur saya" : "Pengajuan ketidakhadiran saya"}
              columns={tab === "lembur" ? overtimeColumns : leaveColumns}
              rows={active.data.items}
              rowKey={(row) => row.id}
              emptyMessage="Belum ada pengajuan."
            />
            {active.data.items.length > 0 && (
              <Pagination
                meta={active.data.meta}
                label="Navigasi halaman pengajuan saya"
                onPageChange={(nextPage) => {
                  const next = new URLSearchParams(params);
                  next.set("page", String(nextPage));
                  setParams(next);
                }}
              />
            )}
          </>
        )}
      </div>
    </section>
  );
};
