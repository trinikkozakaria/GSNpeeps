import { roleLabel } from "../../../routes/navigation/navigation";
import { useAuth } from "../hooks/useAuth";
import { CompanyFeedList } from "../../uat/pages/CompanyFeedPage";
import { useQuery } from "@tanstack/react-query";
import { apiClient } from "../../../lib/api/client";

export const RoleLandingPage = () => {
  const auth = useAuth();
  const summary=useQuery({queryKey:["home-summary",auth.user.id],queryFn:async({signal})=>(await apiClient.get("/beranda",{signal})).data});
  document.title = "Beranda — GSNpeeps";

  return (
    <section aria-labelledby="landing-title">
      <p className="text-sm font-semibold uppercase tracking-[0.2em] text-cyan-700">
        Sesi terverifikasi
      </p>
      <h1 id="landing-title" className="mt-3 text-3xl font-bold sm:text-4xl">
        Selamat datang, {auth.user.nama}
      </h1>
      <p className="mt-4 max-w-2xl leading-7 text-slate-600">
        Anda masuk sebagai {roleLabel[auth.role]}. Navigasi hanya menampilkan area yang
        relevan untuk role ini; backend tetap memverifikasi permission dan scope setiap
        permintaan.
      </p>
      <dl className="mt-8 grid gap-4 sm:grid-cols-2">
        <div className="rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-5">
          <dt className="text-sm text-slate-500">Email</dt>
          <dd className="mt-1 font-semibold">{auth.user.email}</dd>
        </div>
        <div className="rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-5">
          <dt className="text-sm text-slate-500">Role</dt>
          <dd className="mt-1 font-semibold">{roleLabel[auth.role]}</dd>
        </div>
      </dl>
      {summary.data&&<section className="mt-6 grid gap-4 sm:grid-cols-3" aria-label="Ringkasan beranda"><div className="rounded-xl border p-5"><p className="text-sm text-slate-500">Perlu disetujui</p><p className="text-3xl font-bold">{summary.data.pengajuan_perlu_disetujui}</p></div><div className="rounded-xl border p-5"><p className="text-sm text-slate-500">Ketidakhadiran pribadi</p><p className="text-3xl font-bold">{summary.data.pengajuan_ketidakhadiran_pribadi}</p></div><div className="rounded-xl border p-5"><p className="text-sm text-slate-500">Saldo cuti</p>{summary.data.saldo_cuti.length?summary.data.saldo_cuti.map(item=><p key={item.jenis}><strong>{item.sisa}</strong> hari · {item.jenis}</p>):<p className="text-sm">Belum tersedia</p>}</div></section>}
      <section className="mt-8" aria-labelledby="feed-title">
        <h2 id="feed-title" className="mb-4 text-2xl font-bold">Company feed</h2>
        <CompanyFeedList />
      </section>
    </section>
  );
};

