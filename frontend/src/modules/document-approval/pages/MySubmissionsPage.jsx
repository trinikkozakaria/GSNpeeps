import { useMemo } from "react";
import { Link, useSearchParams } from "react-router-dom";

import { DataTable } from "../../../components/data-table/DataTable";
import { Pagination } from "../../../components/data-table/Pagination";
import { Button } from "../../../components/ui/Button";
import { formatDate } from "../../../lib/format";
import { useMyApprovalRequests } from "../hooks/useDocumentApproval";

const statusTone = {
  berjalan: "bg-amber-400/15 text-amber-800",
  disetujui: "bg-emerald-400/15 text-emerald-700",
  ditolak: "bg-rose-400/15 text-rose-700",
  dibatalkan: "bg-slate-200 text-slate-600",
  bermasalah: "bg-orange-400/15 text-orange-800",
};

const statusLabel = {
  berjalan: "Berjalan",
  disetujui: "Disetujui",
  ditolak: "Ditolak",
  dibatalkan: "Dibatalkan",
  bermasalah: "Bermasalah",
};

const ApprovalStatusBadge = ({ status }) => (
  <span className={`inline-flex rounded-full px-2.5 py-1 text-xs font-semibold ${statusTone[status] ?? "bg-slate-200 text-slate-600"}`}>
    {statusLabel[status] ?? status}
  </span>
);

export const MySubmissionsPage = () => {
  document.title = "Pengajuan Saya — GSNpeeps";
  const [params, setParams] = useSearchParams();
  const page = Number.parseInt(params.get("page") ?? "1", 10) || 1;
  const filters = useMemo(() => ({ page, limit: 10 }), [page]);
  const requests = useMyApprovalRequests(filters);

  const columns = [
    {
      key: "judul",
      header: "Dokumen",
      render: (row) => (
        <a href={row.document_url} target="_blank" rel="noreferrer" className="font-semibold text-cyan-700 hover:text-cyan-900">
          {row.judul}
        </a>
      ),
    },
    { key: "tahap", header: "Tahap", render: (row) => `Tahap ${row.stage_index + 1}` },
    { key: "status", header: "Status", render: (row) => <ApprovalStatusBadge status={row.status} /> },
    { key: "created_at", header: "Diajukan", render: (row) => formatDate(row.created_at) },
    {
      key: "aksi",
      srHeader: "Aksi",
      cellClassName: "text-right",
      render: (row) => (
        <Link to={`/app/persetujuan-dokumen/${row.id}`} className="font-semibold text-cyan-700 hover:text-cyan-900">
          Lihat
          <span className="sr-only"> pengajuan {row.judul}</span>
        </Link>
      ),
    },
  ];

  return (
    <section aria-labelledby="my-submissions-title">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <p className="text-sm font-semibold uppercase tracking-widest text-cyan-700">
            Persetujuan Dokumen
          </p>
          <h1 id="my-submissions-title" className="mt-2 text-3xl font-bold">
            Pengajuan Saya
          </h1>
        </div>
        <Link to="/app/persetujuan-dokumen/ajukan">
          <Button>Ajukan Dokumen</Button>
        </Link>
      </div>

      <div className="mt-6" aria-live="polite">
        {requests.isPending && <p role="status" className="text-slate-600">Memuat pengajuan…</p>}
        {requests.isError && (
          <div role="alert" className="rounded-xl border border-red-400/30 bg-red-400/10 p-4 text-red-700">
            <p>Pengajuan belum dapat dimuat. {requests.error.message}</p>
            <Button className="mt-3" variant="secondary" onClick={() => requests.refetch()}>
              Coba lagi
            </Button>
          </div>
        )}
        {requests.data && (
          <>
            <DataTable
              caption="Daftar pengajuan persetujuan dokumen saya"
              columns={columns}
              rows={requests.data.items}
              rowKey={(row) => row.id}
              emptyMessage="Anda belum pernah mengajukan dokumen untuk persetujuan."
            />
            {requests.data.items.length > 0 && (
              <Pagination
                meta={requests.data.meta}
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
