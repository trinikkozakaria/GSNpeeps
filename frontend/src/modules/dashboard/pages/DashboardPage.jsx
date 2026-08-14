import { useMemo } from "react";
import { useSearchParams } from "react-router-dom";

import { GenderIcons, PieChart } from "../../../components/charts/PieChart";
import { Button } from "../../../components/ui/Button";
import { formatCurrency, formatDate, formatNumber, formatPercent } from "../../../lib/format";
import { useAuth } from "../../auth/hooks/useAuth";
import { ComingSoonCards } from "../components/ComingSoonCards";
import { OrganizationChart } from "../components/OrganizationChart";
import { useDashboardMetrics } from "../hooks/useDashboard";
import { dashboardPeriods } from "../schemas/dashboard-schema";

const periodLabel = {
  harian: "Harian",
  mingguan: "Mingguan (Senin–Minggu)",
  bulanan: "Bulanan",
  tahunan: "Tahunan",
};

// Label Bahasa Indonesia dipetakan di frontend; nilai enum wire tidak pernah diubah.
const genderLabel = {
  laki_laki: "Laki-laki",
  perempuan: "Perempuan",
  belum_diisi: "Belum diisi",
};

const MetricCard = ({ label, value, hint }) => (
  <div className="rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-5">
    <p className="text-xs font-semibold uppercase tracking-wider text-slate-500">{label}</p>
    <p className="mt-2 text-3xl font-bold tabular-nums text-slate-900">{value}</p>
    {hint && <p className="mt-1 text-xs text-slate-500">{hint}</p>}
  </div>
);

const Panel = ({ title, children, id }) => (
  <section aria-labelledby={id} className="rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-5">
    <h2 id={id} className="text-lg font-bold">
      {title}
    </h2>
    <div className="mt-4">{children}</div>
  </section>
);

export const DashboardPage = () => {
  document.title = "Dashboard HR — GSNpeeps";
  const auth = useAuth();
  const [params, setParams] = useSearchParams();
  const filters = useMemo(
    () => ({
      periode: dashboardPeriods.includes(params.get("periode") ?? "")
        ? params.get("periode")
        : "bulanan",
      tanggalAcuan: params.get("tanggal_acuan") || "",
    }),
    [params],
  );
  const metrics = useDashboardMetrics(auth.role, filters);
  const data = metrics.data;

  const setFilter = (key, value) => {
    const next = new URLSearchParams(params);
    if (value) next.set(key, value);
    else next.delete(key);
    setParams(next);
  };

  return (
    <section aria-labelledby="dashboard-title">
      <p className="text-sm font-semibold uppercase tracking-widest text-cyan-700">Monitoring</p>
      <h1 id="dashboard-title" className="mt-2 text-3xl font-bold">
        Dashboard HR
      </h1>
      <p className="mt-2 max-w-3xl text-slate-600">
        Seluruh angka dihitung server pada zona waktu Asia/Jakarta.
      </p>

      <div className="mt-7 grid gap-4 rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-4 sm:grid-cols-2 lg:grid-cols-3">
        <label className="text-sm font-medium text-slate-700">
          Periode
          <select
            value={filters.periode}
            onChange={(event) => setFilter("periode", event.target.value)}
            className="mt-2 min-h-10 w-full rounded-lg border border-slate-900/15 bg-white px-3 text-slate-900 outline-none focus:border-cyan-300"
          >
            {dashboardPeriods.map((period) => (
              <option key={period} value={period}>
                {periodLabel[period]}
              </option>
            ))}
          </select>
        </label>
        <label className="text-sm font-medium text-slate-700">
          Tanggal acuan
          <input
            type="date"
            value={filters.tanggalAcuan}
            onChange={(event) => setFilter("tanggal_acuan", event.target.value)}
            className="mt-2 min-h-10 w-full rounded-lg border border-slate-900/15 bg-white px-3 text-slate-900 outline-none focus:border-cyan-300"
          />
          <span className="mt-2 block text-xs text-slate-500">
            Kosongkan untuk memakai tanggal hari ini.
          </span>
        </label>
      </div>

      <div className="mt-6" aria-live="polite">
        {metrics.isPending && (
          <p role="status" className="text-slate-600">
            Memuat metrik dashboard…
          </p>
        )}
        {metrics.isError && (
          <div role="alert" className="rounded-xl border border-red-400/30 bg-red-400/10 p-4 text-red-700">
            <p>Metrik belum dapat dimuat. {metrics.error.message}</p>
            <Button className="mt-3" variant="secondary" onClick={() => metrics.refetch()}>
              Coba lagi
            </Button>
          </div>
        )}

        {data && (
          <>
            <p className="text-sm text-slate-500">
              Periode {periodLabel[data.periode.tipe]}: {formatDate(data.periode.tanggal_mulai)} —{" "}
              {formatDate(data.periode.tanggal_selesai)} ({data.periode.timezone})
            </p>

            <div className="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <MetricCard label="Total karyawan" value={formatNumber(data.total_karyawan)} />
              <MetricCard label="Karyawan aktif" value={formatNumber(data.karyawan_aktif)} />
              <MetricCard label="Karyawan nonaktif" value={formatNumber(data.karyawan_nonaktif)} />
              <MetricCard label="Karyawan baru" value={formatNumber(data.karyawan_baru)} hint="Bergabung dalam periode" />
              <MetricCard label="Resign" value={formatNumber(data.resign)} hint="Dinonaktifkan dalam periode" />
              <MetricCard
                label="Turnover rate"
                value={formatPercent(data.turnover_rate)}
                hint="Resign dibagi rata-rata headcount"
              />
              <MetricCard
                label="Estimasi biaya payroll"
                value={formatCurrency(data.estimasi_biaya_payroll)}
                hint="Proporsional hari kerja Senin–Jumat"
              />
              <MetricCard label="Pengajuan menunggu" value={formatNumber(data.pengajuan_menunggu)} />
            </div>

            <div className="mt-4 grid gap-4 sm:grid-cols-3">
              <MetricCard
                label="Kehadiran valid"
                value={formatNumber(data.hadir_valid)}
                hint="Memiliki clock-in dan clock-out"
              />
              <MetricCard label="Terlambat" value={formatNumber(data.terlambat)} hint="Check-in setelah 09.00 WIB" />
              <MetricCard label="Hari izin disetujui" value={formatNumber(data.hari_izin_disetujui)} />
            </div>

            <div className="mt-8 grid gap-6 lg:grid-cols-2">
              <Panel id="gender-panel-title" title="Rasio gender">
                <GenderIcons
                  items={data.rasio_gender.map((item) => ({
                    nama: genderLabel[item.kategori],
                    jumlah: item.jumlah,
                  }))}
                />
              </Panel>

              <Panel id="department-active-title" title="Komposisi departemen — aktif">
                <PieChart
                  title="Karyawan aktif per departemen"
                  items={data.komposisi_departemen_aktif}
                  emptyMessage="Belum ada karyawan aktif pada periode ini."
                />
              </Panel>

              <Panel id="department-inactive-title" title="Komposisi departemen — nonaktif">
                <PieChart
                  title="Karyawan nonaktif per departemen"
                  items={data.komposisi_departemen_nonaktif}
                  emptyMessage="Belum ada karyawan nonaktif pada periode ini."
                />
              </Panel>

              <Panel id="org-chart-title" title="Struktur organisasi">
                <OrganizationChart nodes={data.organization_chart} />
              </Panel>
            </div>
          </>
        )}
      </div>

      <ComingSoonCards />
    </section>
  );
};
