// Hanya tiga modul yang disetujui berstatus Coming Soon. Kartu ini tidak pernah menampilkan
// angka contoh agar tidak terbaca sebagai data bisnis nyata.
const comingSoonModules = [
  {
    title: "Hiring Progress",
    description: "Pemantauan proses rekrutmen belum termasuk fase ini.",
  },
  {
    title: "Recruitment Cost",
    description: "Perhitungan biaya rekrutmen menunggu keputusan produk.",
  },
  {
    title: "Benefit",
    description: "Pengelolaan benefit dan budget belum termasuk fase ini.",
  },
];

export const ComingSoonCards = () => (
  <section aria-labelledby="coming-soon-title" className="mt-10">
    <h2 id="coming-soon-title" className="text-lg font-bold">
      Belum tersedia
    </h2>
    <ul className="mt-4 grid gap-4 sm:grid-cols-3">
      {comingSoonModules.map((item) => (
        <li key={item.title} className="rounded-xl border border-dashed border-slate-900/15 bg-slate-900/[0.02] p-5">
          <div className="flex items-center gap-2">
            <h3 className="font-semibold text-slate-700">{item.title}</h3>
            <span className="rounded-full border border-slate-900/15 px-2 py-0.5 text-xs font-semibold text-slate-500">
              Coming Soon
            </span>
          </div>
          <p className="mt-2 text-sm text-slate-500">{item.description}</p>
        </li>
      ))}
    </ul>
  </section>
);
