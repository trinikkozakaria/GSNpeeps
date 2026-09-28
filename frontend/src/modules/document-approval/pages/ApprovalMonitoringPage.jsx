import { useMemo } from "react";
import { Link, useSearchParams } from "react-router-dom";

import { DataTable } from "../../../components/data-table/DataTable";
import { Pagination } from "../../../components/data-table/Pagination";
import { Button } from "../../../components/ui/Button";
import { formatDate } from "../../../lib/format";
import { useApprovalMonitoring, useWorkflowTemplates } from "../hooks/useDocumentApproval";
import { documentApprovalStatuses } from "../schemas/document-approval-schema";

const statusLabel = {
  berjalan: "Berjalan",
  disetujui: "Disetujui",
  ditolak: "Ditolak",
  dibatalkan: "Dibatalkan",
  bermasalah: "Bermasalah",
};

const statusTone = {
  berjalan: "bg-amber-400/15 text-amber-800",
  disetujui: "bg-emerald-400/15 text-emerald-700",
  ditolak: "bg-rose-400/15 text-rose-700",
  dibatalkan: "bg-slate-200 text-slate-600",
  bermasalah: "bg-orange-400/15 text-orange-800",
};

export const ApprovalMonitoringPage = () => {
  document.title = "Monitoring Persetujuan Dokumen — GSNpeeps";
  const [params, setParams] = useSearchParams();
  const page = Number.parseInt(params.get("page") ?? "1", 10) || 1;
  const status = params.get("status") || "";
  const templateId = params.get("template_id") || "";

  const filters = useMemo(
    () => ({
      page,
      limit: 10,
      ...(status ? { status } : {}),
      ...(templateId ? { template_id: templateId } : {}),
    }),
    [page, status, templateId],
  );

  const monitoring = useApprovalMonitoring(filters);
  const templates = useWorkflowTemplates();

  const updateFilter = (key, value) => {
    const next = new URLSearchParams(params);
    if (value) {
      next.set(key, value);
    } else {
      next.delete(key);
    }
    next.delete("page");
    setParams(next);
  };

  const columns = [
    { key: "judul", header: "Dokumen", render: (row) => <span className="font-semibold text-slate-900">{row.judul}</span> },
    { key: "tahap", header: "Tahap", render: (row) => `Tahap ${row.stage_index + 1}` },
    {
      key: "status",
      header: "Status",
      render: (row) => (
        <span className={`inline-flex rounded-full px-2.5 py-1 text-xs font-semibold ${statusTone[row.status] ?? "bg-slate-200 text-slate-600"}`}>
          {statusLabel[row.status] ?? row.status}
        </span>
      ),
    },
    { key: "created_at", header: "Diajukan", render: (row) => formatDate(row.created_at) },
    {
      key: "aksi",
      srHeader: "Aksi",
      cellClassName: "text-right",
      render: (row) => (
        <Link to={`/app/persetujuan-dokumen/${row.id}`} className="font-semibold text-cyan-700 hover:text-cyan-900">
          Lihat
          <span className="sr-only"> {row.judul}</span>
        </Link>
      ),
    },
  ];

  return (
    <section aria-labelledby="monitoring-title">
      <p className="text-sm font-semibold uppercase tracking-widest text-cyan-700">Monitoring</p>
      <h1 id="monitoring-title" className="mt-2 text-3xl font-bold">
        Monitoring Persetujuan Dokumen
      </h1>
      <p className="mt-2 max-w-2xl text-slate-600">
        Seluruh pengajuan persetujuan dokumen di organisasi (read-only).
      </p>

      <div className="mt-6 flex flex-wrap gap-4">
        <div>
          <label htmlFor="filter-status" className="block text-sm font-medium">Status</label>
          <select
            id="filter-status"
            value={status}
            onChange={(event) => updateFilter("status", event.target.value)}
            className="mt-1 rounded-lg border border-slate-300 p-2"
          >
            <option value="">Semua status</option>
            {documentApprovalStatuses.map((value) => (
              <option key={value} value={value}>{statusLabel[value]}</option>
            ))}
          </select>
        </div>
        <div>
          <label htmlFor="filter-template" className="block text-sm font-medium">Alur</label>
          <select
            id="filter-template"
            value={templateId}
            onChange={(event) => updateFilter("template_id", event.target.value)}
            className="mt-1 rounded-lg border border-slate-300 p-2"
          >
            <option value="">Semua alur</option>
            {(templates.data ?? []).map((template) => (
              <option key={template.id} value={template.id}>{template.nama}</option>
            ))}
          </select>
        </div>
      </div>

      <div className="mt-6" aria-live="polite">
        {monitoring.isPending && <p role="status" className="text-slate-600">Memuat data…</p>}
        {monitoring.isError && (
          <div role="alert" className="rounded-xl border border-red-400/30 bg-red-400/10 p-4 text-red-700">
            <p>Data belum dapat dimuat. {monitoring.error.message}</p>
            <Button className="mt-3" variant="secondary" onClick={() => monitoring.refetch()}>
              Coba lagi
            </Button>
          </div>
        )}
        {monitoring.data && (
          <>
            <DataTable
              caption="Seluruh pengajuan persetujuan dokumen"
              columns={columns}
              rows={monitoring.data.items}
              rowKey={(row) => row.id}
              emptyMessage="Tidak ada pengajuan yang cocok dengan filter."
            />
            {monitoring.data.items.length > 0 && (
              <Pagination
                meta={monitoring.data.meta}
                label="Navigasi halaman monitoring"
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
