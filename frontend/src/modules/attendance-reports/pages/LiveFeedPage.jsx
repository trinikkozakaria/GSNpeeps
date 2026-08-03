import { useSearchParams } from "react-router-dom";

import { DataTable } from "../../../components/data-table/DataTable";
import { Button } from "../../../components/ui/Button";
import { attendanceStatusLabel, workModeLabel } from "../../../lib/request-status";
import { useAuth } from "../../auth/hooks/useAuth";
import { useLiveFeed } from "../../attendance/hooks/useAttendance";

const monitoringRoles = ["hr", "top_management"];

export const LiveFeedPage = () => {
  document.title = "Live Feed Absensi — GSNpeeps";
  const auth = useAuth();
  const [params, setParams] = useSearchParams();
  const canRead = monitoringRoles.includes(auth.role);
  const tanggal = params.get("tanggal") || "";

  // Karyawan dan Atasan tidak pernah memicu fetch live feed.
  const feed = useLiveFeed(auth.role, tanggal, canRead);

  const columns = [
    {
      key: "karyawan",
      header: "Karyawan",
      render: (row) => (
        <>
          <span className="block font-semibold text-white">{row.nama_karyawan}</span>
          <span className="mt-1 block text-xs text-slate-400">{row.departemen || "—"}</span>
        </>
      ),
    },
    {
      key: "waktu",
      header: "Waktu server",
      render: (row) => new Date(row.waktu).toLocaleTimeString("id-ID"),
    },
    {
      key: "tipe",
      header: "Jenis",
      render: (row) => (row.tipe === "check_in" ? "Check-in" : "Check-out"),
    },
    { key: "mode", header: "Mode", render: (row) => workModeLabel[row.mode_kerja] },
    {
      key: "status",
      header: "Status",
      render: (row) => attendanceStatusLabel[row.status] ?? row.status,
    },
    {
      key: "foto",
      header: "Foto",
      render: (row) =>
        row.foto_url ? (
          <img
            src={row.foto_url}
            loading="lazy"
            alt={`Foto absensi ${row.nama_karyawan}`}
            className="h-12 w-12 rounded object-cover"
          />
        ) : (
          <span className="text-xs text-slate-400">Tidak tersedia</span>
        ),
    },
  ];

  return (
    <section aria-labelledby="live-feed-title">
      <p className="text-sm font-semibold uppercase tracking-widest text-cyan-300">Monitoring</p>
      <h1 id="live-feed-title" className="mt-2 text-3xl font-bold">Live Feed Absensi</h1>
      <p className="mt-2 text-slate-300">
        Menampilkan absensi seluruh karyawan pada tanggal yang dipilih.
      </p>

      <label className="mt-6 block max-w-xs text-sm font-medium text-slate-200">
        Tanggal
        <input
          type="date"
          value={tanggal}
          onChange={(event) => {
            const next = new URLSearchParams(params);
            if (event.target.value) next.set("tanggal", event.target.value);
            else next.delete("tanggal");
            setParams(next);
          }}
          className="mt-2 min-h-11 w-full rounded-lg border border-white/15 bg-slate-950 px-3 text-white outline-none focus:border-cyan-300"
        />
        <span className="mt-2 block text-xs text-slate-400">
          Kosongkan untuk memakai tanggal hari ini menurut server.
        </span>
      </label>

      <div className="mt-6" aria-live="polite">
        {feed.isPending && <p role="status" className="text-slate-300">Memuat live feed…</p>}
        {feed.isError && (
          <div role="alert" className="rounded-xl border border-red-400/30 bg-red-400/10 p-4 text-red-100">
            <p>Live feed belum dapat dimuat. {feed.error.message}</p>
            <Button className="mt-3" variant="secondary" onClick={() => feed.refetch()}>Coba lagi</Button>
          </div>
        )}
        {feed.data && (
          <DataTable
            caption="Live feed absensi"
            columns={columns}
            rows={feed.data}
            rowKey={(row) => row.id}
            emptyMessage="Belum ada absensi tercatat pada tanggal ini."
          />
        )}
      </div>
    </section>
  );
};
