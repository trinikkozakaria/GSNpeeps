export const ModuleAccessPage = ({ title, description, readOnly = false }) => {
  document.title = `${title} — GSNpeeps`;
  return (
    <section aria-labelledby="module-title">
      <div className="flex flex-wrap items-center gap-3">
        <h1 id="module-title" className="text-3xl font-bold">
          {title}
        </h1>
        {readOnly ? (
          <span className="rounded-full border border-amber-300/30 bg-amber-300/10 px-3 py-1 text-xs font-semibold text-amber-200">
            Read-only
          </span>
        ) : null}
      </div>
      <p className="mt-4 max-w-2xl leading-7 text-slate-300">{description}</p>
      <div className="mt-8 rounded-xl border border-dashed border-white/15 p-6 text-sm text-slate-400">
        Batas akses route sudah aktif. Data modul akan disambungkan pada vertical slice
        endpoint terkait tanpa menggunakan data tiruan.
      </div>
    </section>
  );
};

