import { useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";

import { DataTable } from "../../../components/data-table/DataTable";
import { Pagination } from "../../../components/data-table/Pagination";
import { Button } from "../../../components/ui/Button";
import { formatDateTime } from "../../../lib/format";
import { useAuth } from "../../auth/hooks/useAuth";
import { useAuditLogs } from "../hooks/useAuditLogs";
import { formatAuditDetail } from "../utils/redact-detail";

const monitoringRoles = ["hr", "top_management"];

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

export const AuditLogPage = () => {
  document.title = "Audit Log — GSNpeeps";
  const auth = useAuth();
  const [params, setParams] = useSearchParams();
  const [expandedId, setExpandedId] = useState(null);

  // Karyawan dan Atasan tidak memicu permintaan audit sekalipun membuka URL langsung.
  const canRead = monitoringRoles.includes(auth.role);

  const rawUserId = params.get("user_id") ?? "";
  const isUserFilterValid = rawUserId === "" || uuidPattern.test(rawUserId);

  const filters = useMemo(
    () => ({
      tanggal_mulai: params.get("tanggal_mulai") || undefined,
      tanggal_selesai: params.get("tanggal_selesai") || undefined,
      modul: params.get("modul") || undefined,
      aksi: params.get("aksi") || undefined,
      user_id: uuidPattern.test(rawUserId) ? rawUserId : undefined,
      page: Number.parseInt(params.get("page") ?? "1", 10) || 1,
      limit: 10,
    }),
    [params, rawUserId],
  );

  const logs = useAuditLogs(auth.role, filters, canRead);

  const setFilter = (key, value) => {
    const next = new URLSearchParams(params);
    if (value) next.set(key, value);
    else next.delete(key);
    if (key !== "page") next.delete("page");
    setParams(next);
  };

  if (!canRead) {
    return (
      <section aria-labelledby="audit-title">
        <h1 id="audit-title" className="text-3xl font-bold">
          Audit Log
        </h1>
        <p role="alert" className="mt-4 text-slate-300">
          Modul ini hanya tersedia untuk HR dan Top Management.
        </p>
      </section>
    );
  }

  const columns = [
    {
      key: "waktu",
      header: "Waktu",
      render: (row) => <time dateTime={row.created_at}>{formatDateTime(row.created_at)}</time>,
    },
    {
      key: "aktor",
      header: "Aktor",
      render: (row) =>
        row.user_id ? (
          <span className="flex flex-wrap items-center gap-2">
            <span>{row.nama_user || "Pengguna"}</span>
            <Button variant="secondary" onClick={() => setFilter("user_id", row.user_id)}>
              Filter aktor
            </Button>
          </span>
        ) : (
          // Aktor sistem tidak memiliki user; jangan menampilkan pengguna palsu.
          <span className="text-slate-300">Sistem</span>
        ),
    },
    { key: "aksi", header: "Aksi", render: (row) => row.aksi },
    { key: "modul", header: "Modul", render: (row) => row.modul },
    { key: "ip", header: "IP", render: (row) => row.ip_address || "—" },
    {
      key: "detail",
      srHeader: "Detail",
      render: (row) => {
        const detail = formatAuditDetail(row.detail);
        if (!detail) return <span className="text-slate-400">Tanpa detail</span>;
        const isOpen = expandedId === row.id;
        return (
          <div>
            <Button
              variant="secondary"
              aria-expanded={isOpen}
              aria-controls={`audit-detail-${row.id}`}
              onClick={() => setExpandedId(isOpen ? null : row.id)}
            >
              {isOpen ? "Sembunyikan detail" : "Lihat detail"}
            </Button>
            {isOpen && (
              /* Detail dirender sebagai teks JSON, bukan HTML. */
              <pre
                id={`audit-detail-${row.id}`}
                className="mt-2 max-w-full overflow-x-auto rounded-lg border border-white/10 bg-slate-950 p-3 text-xs text-slate-200"
              >
                {detail}
              </pre>
            )}
          </div>
        );
      },
    },
  ];

  return (
    <section aria-labelledby="audit-title">
      <p className="text-sm font-semibold uppercase tracking-widest text-cyan-300">Keamanan</p>
      <h1 id="audit-title" className="mt-2 text-3xl font-bold">
        Audit Log
      </h1>
      <p className="mt-2 text-slate-300">
        Riwayat aktivitas bersifat append-only dan tidak dapat diubah maupun dihapus dari
        aplikasi. Nilai sensitif sudah diredaksi.
      </p>

      <div className="mt-7 grid gap-4 rounded-xl border border-white/10 bg-white/[0.03] p-4 sm:grid-cols-2 lg:grid-cols-5">
        <label className="text-sm font-medium text-slate-200">
          Tanggal mulai
          <input
            type="date"
            value={filters.tanggal_mulai ?? ""}
            onChange={(event) => setFilter("tanggal_mulai", event.target.value)}
            className="mt-2 min-h-11 w-full rounded-lg border border-white/15 bg-slate-950 px-3 text-white outline-none focus:border-cyan-300"
          />
        </label>
        <label className="text-sm font-medium text-slate-200">
          Tanggal selesai
          <input
            type="date"
            value={filters.tanggal_selesai ?? ""}
            onChange={(event) => setFilter("tanggal_selesai", event.target.value)}
            className="mt-2 min-h-11 w-full rounded-lg border border-white/15 bg-slate-950 px-3 text-white outline-none focus:border-cyan-300"
          />
        </label>
        <label className="text-sm font-medium text-slate-200">
          Modul
          <input
            type="text"
            value={filters.modul ?? ""}
            placeholder="ketidakhadiran"
            onChange={(event) => setFilter("modul", event.target.value)}
            className="mt-2 min-h-11 w-full rounded-lg border border-white/15 bg-slate-950 px-3 text-white outline-none focus:border-cyan-300"
          />
        </label>
        <label className="text-sm font-medium text-slate-200">
          Aksi
          <input
            type="text"
            value={filters.aksi ?? ""}
            placeholder="APPROVE"
            onChange={(event) => setFilter("aksi", event.target.value)}
            className="mt-2 min-h-11 w-full rounded-lg border border-white/15 bg-slate-950 px-3 text-white outline-none focus:border-cyan-300"
          />
        </label>
        <label className="text-sm font-medium text-slate-200">
          ID pengguna
          <input
            type="text"
            value={rawUserId}
            placeholder="UUID pengguna"
            aria-invalid={!isUserFilterValid}
            aria-describedby={isUserFilterValid ? undefined : "audit-user-error"}
            onChange={(event) => setFilter("user_id", event.target.value)}
            className="mt-2 min-h-11 w-full rounded-lg border border-white/15 bg-slate-950 px-3 text-white outline-none focus:border-cyan-300"
          />
        </label>
      </div>

      {!isUserFilterValid && (
        <p
          id="audit-user-error"
          role="alert"
          className="mt-3 text-sm text-amber-200"
        >
          ID pengguna harus berupa UUID. Filter aktor diabaikan sampai formatnya benar.
        </p>
      )}

      <div className="mt-6" aria-live="polite">
        {logs.isPending && (
          <p role="status" className="text-slate-300">
            Memuat audit log…
          </p>
        )}
        {logs.isError && (
          <div
            role="alert"
            className="rounded-xl border border-red-400/30 bg-red-400/10 p-4 text-red-100"
          >
            <p>Audit log belum dapat dimuat. {logs.error?.message}</p>
            <Button className="mt-3" variant="secondary" onClick={() => logs.refetch()}>
              Coba lagi
            </Button>
          </div>
        )}
        {logs.data && (
          <>
            <DataTable
              caption="Riwayat aktivitas terkontrol"
              columns={columns}
              rows={logs.data.items}
              rowKey={(row) => row.id}
              emptyMessage="Tidak ada aktivitas pada filter ini."
            />
            {logs.data.items.length > 0 && (
              <Pagination
                meta={logs.data.meta}
                label="Navigasi halaman audit log"
                onPageChange={(page) => setFilter("page", String(page))}
              />
            )}
          </>
        )}
      </div>
    </section>
  );
};
