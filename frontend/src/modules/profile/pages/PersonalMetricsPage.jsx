import { DataTable } from "../../../components/data-table/DataTable";
import { Button } from "../../../components/ui/Button";
import { formatDate, formatHours, formatNumber, formatPeriod } from "../../../lib/format";
import { useAuth } from "../../auth/hooks/useAuth";
import { usePersonalMetrics } from "../hooks/useProfile";

const MetricCard = ({ label, value, hint }) => (
  <div className="rounded-xl border border-white/10 bg-white/[0.03] p-5">
    <p className="text-xs font-semibold uppercase tracking-wider text-slate-500">{label}</p>
    <p className="mt-2 text-3xl font-bold tabular-nums text-white">{value}</p>
    {hint && <p className="mt-1 text-xs text-slate-400">{hint}</p>}
  </div>
);

/**
 * Metrik Personal hanya untuk Karyawan, Atasan, dan HR. Nilai selalu berasal dari response
 * API; ketika modul Kehadiran belum aktif, nilai nol ditampilkan apa adanya dengan
 * penjelasan, bukan diganti angka contoh.
 */
export const PersonalMetricsPage = () => {
  document.title = "Metrik Personal — GSNpeeps";
  const auth = useAuth();
  const metrics = usePersonalMetrics(auth.user?.id);
  const data = metrics.data;
  const hasClockHistory = (data?.riwayat_absensi.length ?? 0) > 0;
  const hasActivity =
    Boolean(data) &&
    (data.hadir > 0 || data.terlambat > 0 || data.izin > 0 || data.total_lembur_jam > 0 || hasClockHistory);

  return (
    <section aria-labelledby="personal-metrics-title">
      <p className="text-sm font-semibold uppercase tracking-widest text-cyan-300">Ringkasan pribadi</p>
      <h1 id="personal-metrics-title" className="mt-2 text-3xl font-bold">
        Metrik Personal
      </h1>
      {data && (
        <p className="mt-2 text-slate-300">Periode {formatPeriod(data.periode)}.</p>
      )}

      <div className="mt-6" aria-live="polite">
        {metrics.isPending && (
          <p role="status" className="text-slate-300">
            Memuat metrik personal…
          </p>
        )}
        {metrics.isError && (
          <div role="alert" className="rounded-xl border border-red-400/30 bg-red-400/10 p-4 text-red-100">
            <p>Metrik belum dapat dimuat. {metrics.error.message}</p>
            <Button className="mt-3" variant="secondary" onClick={() => metrics.refetch()}>
              Coba lagi
            </Button>
          </div>
        )}
        {data && (
          <>
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <MetricCard label="Hari hadir" value={formatNumber(data.hadir)} />
              <MetricCard label="Terlambat" value={formatNumber(data.terlambat)} hint="Check-in setelah 09.00 WIB" />
              <MetricCard label="Hari izin" value={formatNumber(data.izin)} />
              <MetricCard label="Total lembur" value={formatHours(data.total_lembur_jam)} />
            </div>

            {!hasActivity && (
              <p className="mt-6 rounded-xl border border-white/10 bg-white/[0.03] p-5 text-sm text-slate-300">
                Belum ada aktivitas kehadiran yang tercatat pada periode ini. Angka akan terisi
                setelah modul Kehadiran dan Pengajuan aktif dan Anda mulai melakukan check-in.
              </p>
            )}

            <section className="mt-8" aria-labelledby="clock-history-title">
              <h2 id="clock-history-title" className="text-lg font-bold">
                Riwayat check-in dan check-out
              </h2>
              <div className="mt-4">
                <DataTable
                  caption="Riwayat check-in dan check-out"
                  columns={[
                    {
                      key: "tanggal",
                      header: "Tanggal",
                      render: (row) => formatDate(row.tanggal),
                    },
                    {
                      key: "check_in",
                      header: "Check-in",
                      render: (row) => row.check_in ?? "—",
                    },
                    {
                      key: "check_out",
                      header: "Check-out",
                      render: (row) => row.check_out ?? "—",
                    },
                    { key: "status", header: "Status", render: (row) => row.status },
                  ]}
                  rows={data.riwayat_absensi}
                  rowKey={(row) => row.tanggal}
                  emptyMessage="Belum ada riwayat check-in pada periode ini."
                />
              </div>
            </section>
          </>
        )}
      </div>
    </section>
  );
};
