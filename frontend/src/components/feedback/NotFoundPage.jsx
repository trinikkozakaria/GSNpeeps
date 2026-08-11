import { Link } from "react-router-dom";

export const NotFoundPage = () => {
  document.title = "Halaman tidak ditemukan — GSNpeeps";

  return (
    <section aria-labelledby="not-found-title" className="mx-auto max-w-xl py-20 text-center">
      <p className="text-sm font-semibold uppercase tracking-widest text-cyan-700">404</p>
      <h1 id="not-found-title" className="mt-3 text-3xl font-bold">
        Halaman tidak ditemukan
      </h1>
      <p className="mt-4 text-slate-600">
        Alamat yang dibuka tidak tersedia atau sudah berubah.
      </p>
      <Link
        to="/"
        className="mt-8 inline-flex min-h-11 items-center rounded-lg bg-cyan-700 px-4 py-2 font-semibold text-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-cyan-700"
      >
        Kembali ke beranda
      </Link>
    </section>
  );
};

