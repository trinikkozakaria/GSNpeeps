import { Link } from "react-router-dom";

export const ForbiddenPage = () => {
  document.title = "Akses ditolak — GSNpeeps";
  return (
    <main className="grid min-h-screen place-items-center bg-slate-950 px-6 text-slate-100">
      <section aria-labelledby="forbidden-title" className="max-w-lg text-center">
        <p className="text-sm font-semibold uppercase tracking-widest text-amber-300">403</p>
        <h1 id="forbidden-title" className="mt-3 text-3xl font-bold">
          Anda tidak memiliki akses
        </h1>
        <p className="mt-4 text-slate-300">
          Sesi tetap aktif. Kembali ke area yang tersedia untuk role Anda.
        </p>
        <Link
          to="/app"
          className="mt-7 inline-flex min-h-11 items-center rounded-lg bg-cyan-300 px-4 py-2 font-semibold text-slate-950"
        >
          Kembali ke beranda
        </Link>
      </section>
    </main>
  );
};

