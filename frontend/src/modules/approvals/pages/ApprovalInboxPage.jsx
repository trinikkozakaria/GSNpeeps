import { useMemo } from "react";
import { Link, useSearchParams } from "react-router-dom";

import { DataTable } from "../../../components/data-table/DataTable";
import { Pagination } from "../../../components/data-table/Pagination";
import { Button } from "../../../components/ui/Button";
import { formatDate } from "../../../lib/format";
import { useAuth } from "../../auth/hooks/useAuth";
import { useLeaveApprovalInbox } from "../../leave/hooks/useLeave";
import { useOvertimeList } from "../../overtime/hooks/useOvertime";
import { RequestStatusBadge } from "../components/RequestStatusBadge";

// Hanya role approver yang memiliki inbox. Karyawan tidak pernah memicu fetch.
const approverRoles = ["atasan", "hr", "top_management"];

const stageExplanation = {
  atasan: "Menampilkan pengajuan bawahan langsung Anda yang menunggu keputusan Atasan.",
  hr: "Menampilkan pengajuan yang menunggu keputusan HR.",
  top_management: "Menampilkan pengajuan milik HR yang menunggu keputusan Top Management.",
};

export const ApprovalInboxPage = () => {
  document.title = "Persetujuan — GSNpeeps";
  const auth = useAuth();
  const [params, setParams] = useSearchParams();
  const isApprover = approverRoles.includes(auth.role);

  const tab = params.get("tab") === "lembur" ? "lembur" : "ketidakhadiran";
  const page = Number.parseInt(params.get("page") ?? "1", 10) || 1;
  const filters = useMemo(() => ({ page, limit: 10 }), [page]);

  const leaveInbox = useLeaveApprovalInbox(
    auth.role,
    filters,
    isApprover && tab === "ketidakhadiran",
  );
  const overtimeInbox = useOvertimeList(auth.role, filters, isApprover && tab === "lembur");
  const active = tab === "lembur" ? overtimeInbox : leaveInbox;

  const setTab = (nextTab) => {
    const next = new URLSearchParams(params);
    next.set("tab", nextTab);
    next.delete("page");
    setParams(next);
  };

  if (!isApprover) {
    return (
      <section aria-labelledby="approval-title">
        <h1 id="approval-title" className="text-3xl font-bold">
          Persetujuan
        </h1>
        <p role="status" className="mt-3 text-slate-600">
          Peran Anda tidak memiliki antrean persetujuan.
        </p>
      </section>
    );
  }

  const leaveColumns = [
    {
      key: "karyawan",
      header: "Pemohon",
      render: (row) => (
        <>
          <span className="block font-semibold text-slate-900">{row.nama_karyawan}</span>
          <span className="mt-1 block text-xs text-slate-500">{row.jenis_izin}</span>
        </>
      ),
    },
    {
      key: "periode",
      header: "Periode",
      render: (row) =>
        `${formatDate(row.tanggal_mulai)} — ${formatDate(row.tanggal_selesai)} (${row.jumlah_hari} hari)`,
    },
    { key: "status", header: "Status", render: (row) => <RequestStatusBadge status={row.status} /> },
    {
      key: "aksi",
      srHeader: "Aksi",
      cellClassName: "text-right",
      render: (row) => (
        <Link
          to={`/app/persetujuan/ketidakhadiran/${row.id}`}
          className="font-semibold text-cyan-700 hover:text-cyan-900"
        >
          Tinjau
          <span className="sr-only"> pengajuan {row.nama_karyawan}</span>
        </Link>
      ),
    },
  ];

  const overtimeColumns = [
    {
      key: "karyawan",
      header: "Pemohon",
      render: (row) => <span className="font-semibold text-slate-900">{row.nama_karyawan}</span>,
    },
    {
      key: "waktu",
      header: "Waktu",
      render: (row) =>
        `${formatDate(row.tanggal)} · ${row.waktu_mulai}–${row.waktu_selesai} (${row.total_jam} jam)`,
    },
    { key: "status", header: "Status", render: (row) => <RequestStatusBadge status={row.status} /> },
    {
      key: "aksi",
      srHeader: "Aksi",
      cellClassName: "text-right",
      render: (row) => (
        <Link
          to={`/app/persetujuan/lembur/${row.id}`}
          className="font-semibold text-cyan-700 hover:text-cyan-900"
        >
          Tinjau
          <span className="sr-only"> lembur {row.nama_karyawan}</span>
        </Link>
      ),
    },
  ];

  return (
    <section aria-labelledby="approval-title">
      <p className="text-sm font-semibold uppercase tracking-widest text-cyan-700">Approval</p>
      <h1 id="approval-title" className="mt-2 text-3xl font-bold">
        Persetujuan
      </h1>
      <p className="mt-2 max-w-2xl text-slate-600">{stageExplanation[auth.role]}</p>

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
        {active.isPending && (
          <p role="status" className="text-slate-600">
            Memuat antrean persetujuan…
          </p>
        )}
        {active.isError && (
          <div role="alert" className="rounded-xl border border-red-400/30 bg-red-400/10 p-4 text-red-700">
            <p>Antrean belum dapat dimuat. {active.error.message}</p>
            <Button className="mt-3" variant="secondary" onClick={() => active.refetch()}>
              Coba lagi
            </Button>
          </div>
        )}
        {active.data && (
          <>
            <DataTable
              caption={tab === "lembur" ? "Antrean lembur" : "Antrean ketidakhadiran"}
              columns={tab === "lembur" ? overtimeColumns : leaveColumns}
              rows={active.data.items}
              rowKey={(row) => row.id}
              emptyMessage="Tidak ada pengajuan yang menunggu keputusan Anda."
            />
            {active.data.items.length > 0 && (
              <Pagination
                meta={active.data.meta}
                label="Navigasi halaman antrean persetujuan"
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
