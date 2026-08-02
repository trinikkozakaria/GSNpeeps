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
        <li key={item.title} className="rounded-xl border border-dashed border-white/15 bg-white/[0.02] p-5">
          <div className="flex items-center gap-2">
            <h3 className="font-semibold text-slate-200">{item.title}</h3>
            <span className="rounded-full border border-white/15 px-2 py-0.5 text-xs font-semibold text-slate-400">
              Coming Soon
            </span>
          </div>
          <p className="mt-2 text-sm text-slate-400">{item.description}</p>
        </li>
      ))}
    </ul>
  </section>
);
