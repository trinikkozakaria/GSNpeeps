import { Link } from "react-router-dom";

import { DataTable } from "../../../components/data-table/DataTable";
import { Button } from "../../../components/ui/Button";
import { formatDate } from "../../../lib/format";
import { useAuth } from "../../auth/hooks/useAuth";
import { useWorkflowTemplates } from "../hooks/useDocumentApproval";

export const WorkflowTemplatesPage = () => {
  document.title = "Alur Persetujuan Dokumen — GSNpeeps";
  const auth = useAuth();
  const isHR = auth.role === "hr";

  const templates = useWorkflowTemplates();

  const columns = [
    {
      key: "nama",
      header: "Nama Alur",
      render: (row) => <span className="font-semibold text-slate-900">{row.nama}</span>,
    },
    {
      key: "status",
      header: "Status",
      render: (row) => (
        <span
          className={`inline-flex rounded-full px-2.5 py-1 text-xs font-semibold ${
            row.aktif ? "bg-emerald-400/15 text-emerald-700" : "bg-slate-200 text-slate-600"
          }`}
        >
          {row.aktif ? "Aktif" : "Nonaktif"}
        </span>
      ),
    },
    {
      key: "created_at",
      header: "Dibuat",
      render: (row) => formatDate(row.created_at),
    },
    ...(isHR
      ? [
          {
            key: "aksi",
            srHeader: "Aksi",
            cellClassName: "text-right",
            render: (row) => (
              <Link
                to={`/app/master/alur-persetujuan-dokumen/${row.id}/edit`}
                className="font-semibold text-cyan-700 hover:text-cyan-900"
              >
                Edit
                <span className="sr-only"> alur {row.nama}</span>
              </Link>
            ),
          },
        ]
      : []),
  ];

  return (
    <section aria-labelledby="workflow-templates-title">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <p className="text-sm font-semibold uppercase tracking-widest text-cyan-700">
            Master Data
          </p>
          <h1 id="workflow-templates-title" className="mt-2 text-3xl font-bold">
            Alur Persetujuan Dokumen
          </h1>
        </div>
        {isHR && (
          <Link to="/app/master/alur-persetujuan-dokumen/baru">
            <Button>Tambah Alur</Button>
          </Link>
        )}
      </div>

      <div className="mt-6" aria-live="polite">
        {templates.isPending && (
          <p role="status" className="text-slate-600">
            Memuat alur persetujuan…
          </p>
        )}
        {templates.isError && (
          <div role="alert" className="rounded-xl border border-red-400/30 bg-red-400/10 p-4 text-red-700">
            <p>Alur persetujuan belum dapat dimuat. {templates.error.message}</p>
            <Button className="mt-3" variant="secondary" onClick={() => templates.refetch()}>
              Coba lagi
            </Button>
          </div>
        )}
        {templates.data && (
          <DataTable
            caption="Daftar alur persetujuan dokumen"
            columns={columns}
            rows={templates.data}
            rowKey={(row) => row.id}
            emptyMessage="Belum ada alur persetujuan dokumen. Buat alur pertama untuk memulai."
          />
        )}
      </div>
    </section>
  );
};
