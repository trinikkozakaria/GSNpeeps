import { useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";

import { DataTable } from "../../../components/data-table/DataTable";
import { Button } from "../../../components/ui/Button";
import { attendanceStatusLabel, workModeLabel } from "../../../lib/request-status";
import { formatTime } from "../../../lib/format";
import { useAuth } from "../../auth/hooks/useAuth";
import { useLiveFeed } from "../../attendance/hooks/useAttendance";
import { PhotoPreviewModal } from "../components/PhotoPreviewModal";

const monitoringRoles = ["hr"];

// Live feed API mengembalikan satu baris per event (check-in dan check-out terpisah).
// Satu hari kehadiran karyawan tetap dihitung satu baris di tabel; check-in dan check-out
// pada tanggal yang sama digabung, bukan ditampilkan sebagai dua kehadiran terpisah.
const groupByEmployeeDay = (items) => {
  const rows = new Map();
  for (const item of items) {
    const key = `${item.employee_id}_${item.tanggal}`;
    const existing = rows.get(key) ?? {
      key,
      employeeId: item.employee_id,
      namaKaryawan: item.nama_karyawan,
      departemen: item.departemen,
      modeKerja: item.mode_kerja,
      checkIn: null,
      checkOut: null,
    };
    if (item.tipe === "check_in") existing.checkIn = item;
    else existing.checkOut = item;
    existing.modeKerja = item.mode_kerja ?? existing.modeKerja;
    rows.set(key, existing);
  }
  return [...rows.values()].sort((a, b) => {
    const latest = (row) => row.checkOut?.waktu ?? row.checkIn?.waktu ?? "";
    return latest(b).localeCompare(latest(a));
  });
};

export const LiveFeedPage = () => {
  document.title = "Live Feed Absensi — GSNpeeps";
  const auth = useAuth();
  const [params, setParams] = useSearchParams();
  const canRead = monitoringRoles.includes(auth.role);
  const tanggal = params.get("tanggal") || "";
  const [preview, setPreview] = useState(null);

  // Karyawan dan Atasan tidak pernah memicu fetch live feed.
  const feed = useLiveFeed(auth.role, tanggal, canRead);
  const rows = useMemo(() => groupByEmployeeDay(feed.data ?? []), [feed.data]);

  const columns = [
    {
      key: "karyawan",
      header: "Karyawan",
      render: (row) => (
        <>
          <span className="block font-semibold text-slate-900">{row.namaKaryawan}</span>
          <span className="mt-1 block text-xs text-slate-500">{row.departemen || "—"}</span>
        </>
      ),
    },
    {
      key: "check_in",
      header: "Check-in",
      render: (row) =>
        row.checkIn ? formatTime(row.checkIn.waktu) : "—",
    },
    {
      key: "check_out",
      header: "Check-out",
      render: (row) =>
        row.checkOut ? formatTime(row.checkOut.waktu) : "—",
    },
    { key: "mode", header: "Mode", render: (row) => workModeLabel[row.modeKerja] },
    {
      key: "status",
      header: "Status",
      render: (row) => {
        const statuses = [row.checkIn && ["Masuk", row.checkIn.status], row.checkOut && ["Pulang", row.checkOut.status]].filter(Boolean);
        if (statuses.length === 0) return "—";
        return <div className="flex flex-wrap gap-1">{statuses.map(([label,status]) => { const danger=status==="terlambat"||status==="pulang_cepat"; return <span key={label} className={`inline-flex rounded-full px-2.5 py-1 text-xs font-bold ${danger?"bg-red-100 text-red-800":"bg-emerald-100 text-emerald-800"}`}>{label}: {attendanceStatusLabel[status]??status}</span>; })}</div>;
      },
    },
    {
      key: "foto",
      header: "Foto",
      render: (row) => (
        <div className="flex flex-wrap gap-2">
          {row.checkIn?.foto_url && (
            <Button variant="secondary" onClick={() => setPreview({ row, event: row.checkIn, label: "masuk" })}>
              Foto masuk
            </Button>
          )}
          {row.checkOut?.foto_url && (
            <Button variant="secondary" onClick={() => setPreview({ row, event: row.checkOut, label: "keluar" })}>
              Foto keluar
            </Button>
          )}
          {!row.checkIn?.foto_url && !row.checkOut?.foto_url && (
            <span className="text-xs text-slate-500">Tidak tersedia</span>
          )}
        </div>
      ),
    },
  ];

  return (
    <section aria-labelledby="live-feed-title">
      <p className="text-sm font-semibold uppercase tracking-widest text-cyan-700">Monitoring</p>
      <h1 id="live-feed-title" className="mt-2 text-3xl font-bold">Live Feed Absensi</h1>
      <p className="mt-2 text-slate-600">
        Menampilkan absensi seluruh karyawan pada tanggal yang dipilih.
      </p>

      <label className="mt-6 block max-w-xs text-sm font-medium text-slate-700">
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
          className="mt-2 min-h-10 w-full rounded-lg border border-slate-900/15 bg-white px-3 text-slate-900 outline-none focus:border-cyan-300"
        />
        <span className="mt-2 block text-xs text-slate-500">
          Kosongkan untuk memakai tanggal hari ini menurut server.
        </span>
      </label>

      <div className="mt-6" aria-live="polite">
        {feed.isPending && <p role="status" className="text-slate-600">Memuat live feed…</p>}
        {feed.isError && (
          <div role="alert" className="rounded-xl border border-red-400/30 bg-red-400/10 p-4 text-red-700">
            <p>Live feed belum dapat dimuat. {feed.error.message}</p>
            <Button className="mt-3" variant="secondary" onClick={() => feed.refetch()}>Coba lagi</Button>
          </div>
        )}
        {feed.data && (
          <DataTable
            caption="Live feed absensi"
            columns={columns}
            rows={rows}
            rowKey={(row) => row.key}
            emptyMessage="Belum ada absensi tercatat pada tanggal ini."
          />
        )}
      </div>

      <PhotoPreviewModal
        open={Boolean(preview)}
        photoUrl={preview?.event?.foto_url}
        title={`Foto ${preview?.label ?? ""} ${preview?.row?.namaKaryawan ?? ""}`}
        onClose={() => setPreview(null)}
      />
    </section>
  );
};
