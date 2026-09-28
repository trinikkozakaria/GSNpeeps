import { useMemo } from "react";
import { Link, useSearchParams } from "react-router-dom";

import { DataTable } from "../../../components/data-table/DataTable";
import { Pagination } from "../../../components/data-table/Pagination";
import { Button } from "../../../components/ui/Button";
import { formatDate } from "../../../lib/format";
import { useApprovalInbox } from "../hooks/useDocumentApproval";

export const ApprovalInboxDocumentPage = () => {
  document.title = "Inbox Persetujuan Dokumen — GSNpeeps";
  const [params, setParams] = useSearchParams();
  const page = Number.parseInt(params.get("page") ?? "1", 10) || 1;
  const filters = useMemo(() => ({ page, limit: 10 }), [page]);
  const inbox = useApprovalInbox(filters);

  const columns = [
    {
      key: "judul",
      header: "Dokumen",
      render: (row) => (
        <a href={row.document_url} target="_blank" rel="noreferrer" className="font-semibold text-slate-900 hover:text-cyan-700">
          {row.judul}
        </a>
      ),
    },
    { key: "tahap", header: "Tahap saat ini", render: (row) => `Tahap ${row.stage_index + 1}` },
    { key: "created_at", header: "Diajukan", render: (row) => formatDate(row.created_at) },
    {
      key: "aksi",
      srHeader: "Aksi",
      cellClassName: "text-right",
      render: (row) => (
        <Link to={`/app/persetujuan-dokumen/${row.id}`} className="font-semibold text-cyan-700 hover:text-cyan-900">
          Tinjau
          <span className="sr-only"> dokumen {row.judul}</span>
        </Link>
      ),
    },
  ];

  return (
    <section aria-labelledby="doc-approval-inbox-title">
      <p className="text-sm font-semibold uppercase tracking-widest text-cyan-700">Persetujuan</p>
      <h1 id="doc-approval-inbox-title" className="mt-2 text-3xl font-bold">
        Inbox Persetujuan Dokumen
      </h1>
      <p className="mt-2 max-w-2xl text-slate-600">
        Dokumen yang menunggu keputusan Anda pada posisi yang Anda pegang saat ini.
      </p>

      <div className="mt-6" aria-live="polite">
        {inbox.isPending && <p role="status" className="text-slate-600">Memuat antrean…</p>}
        {inbox.isError && (
          <div role="alert" className="rounded-xl border border-red-400/30 bg-red-400/10 p-4 text-red-700">
            <p>Antrean belum dapat dimuat. {inbox.error.message}</p>
            <Button className="mt-3" variant="secondary" onClick={() => inbox.refetch()}>
              Coba lagi
            </Button>
          </div>
        )}
        {inbox.data && (
          <>
            <DataTable
              caption="Antrean persetujuan dokumen"
              columns={columns}
              rows={inbox.data.items}
              rowKey={(row) => row.id}
              emptyMessage="Tidak ada dokumen yang menunggu keputusan Anda."
            />
            {inbox.data.items.length > 0 && (
              <Pagination
                meta={inbox.data.meta}
                label="Navigasi halaman inbox persetujuan dokumen"
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
