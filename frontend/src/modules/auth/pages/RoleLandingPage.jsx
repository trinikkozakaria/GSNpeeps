import { useQuery } from "@tanstack/react-query";

import { Button } from "../../../components/ui/Button";
import { CompanyFeedInfiniteList } from "../../uat/components/CompanyFeedInfiniteList";
import { homeSummaryRequest } from "../../uat/api/uat-api";
import { roleLabel } from "../../../routes/navigation/navigation";
import { useAuth } from "../hooks/useAuth";

export const RoleLandingPage = () => {
  const auth = useAuth();
  const summary = useQuery({
    queryKey: ["home-summary", auth.user.id],
    queryFn: ({ signal }) => homeSummaryRequest(signal),
  });
  document.title = "Beranda — GSNpeeps";

  return (
    <section aria-labelledby="landing-title">
      <p className="text-sm font-semibold uppercase tracking-[0.2em] text-cyan-700">Sesi terverifikasi</p>
      <h1 id="landing-title" className="mt-3 text-3xl font-bold sm:text-4xl">Selamat datang, {auth.user.nama}</h1>
      <p className="mt-4 max-w-2xl leading-7 text-slate-600">
        Anda masuk sebagai {roleLabel[auth.role]}. Navigasi hanya menampilkan area yang
        relevan untuk role ini; backend tetap memverifikasi permission dan scope setiap permintaan.
      </p>
      <dl className="mt-8 grid gap-4 sm:grid-cols-2">
        <div className="rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-5">
          <dt className="text-sm text-slate-600">Email</dt>
          <dd className="mt-1 font-semibold">{auth.user.email}</dd>
        </div>
        <div className="rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-5">
          <dt className="text-sm text-slate-600">Role</dt>
          <dd className="mt-1 font-semibold">{roleLabel[auth.role]}</dd>
        </div>
      </dl>

      <section className="mt-6" aria-labelledby="home-summary-title" aria-live="polite">
        <h2 id="home-summary-title" className="mb-4 text-2xl font-bold">Ringkasan</h2>
        {summary.isPending && <p role="status" className="text-slate-600">Memuat ringkasan beranda…</p>}
        {summary.isError && (
          <div role="alert" className="rounded-xl border border-red-300 bg-red-50 p-4 text-red-700">
            <p>Ringkasan beranda belum dapat dimuat. {summary.error?.message}</p>
            <Button className="mt-3" variant="secondary" onClick={() => summary.refetch()}>Coba lagi</Button>
          </div>
        )}
        {summary.data && (
          <div className="grid gap-4 sm:grid-cols-3" aria-label="Ringkasan beranda">
            <div className="rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-5">
              <p className="text-sm text-slate-600">Perlu disetujui</p>
              <p className="mt-1 text-3xl font-bold">{summary.data.pengajuan_perlu_disetujui}</p>
            </div>
            <div className="rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-5">
              <p className="text-sm text-slate-600">Ketidakhadiran pribadi</p>
              <p className="mt-1 text-3xl font-bold">{summary.data.pengajuan_ketidakhadiran_pribadi}</p>
            </div>
            <div className="rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-5">
              <p className="text-sm text-slate-600">Saldo cuti</p>
              {summary.data.saldo_cuti.length ? summary.data.saldo_cuti.map((item) => (
                <p key={item.jenis} className="mt-1 flex flex-wrap items-baseline gap-x-1.5">
                  <strong className="text-3xl font-bold leading-none">{item.sisa}</strong>
                  <span className="text-sm">hari · {item.jenis}</span>
                </p>
              )) : <p className="text-sm">Belum tersedia</p>}
            </div>
          </div>
        )}
      </section>

      <section className="mt-8" aria-labelledby="feed-title">
        <h2 id="feed-title" className="mb-4 text-2xl font-bold">Company feed</h2>
        <CompanyFeedInfiniteList />
      </section>
    </section>
  );
};
