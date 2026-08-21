/**
 * Pagination server-side. Nilai halaman dan total selalu berasal dari `meta` response API,
 * tidak dihitung ulang di client.
 */
export const Pagination = ({
  meta,
  onPageChange,
  onPageSizeChange,
  pageSizeOptions = [10, 20, 50, 100],
  itemLabel = "data",
  itemAriaLabel = itemLabel,
  label = "Navigasi halaman",
}) => {
  const totalPage = Math.max(meta.total_page, 1);
  const isFirst = meta.page <= 1;
  const isLast = meta.page >= meta.total_page;

  return (
    <nav className="mt-4 flex flex-wrap items-center justify-between gap-3 text-sm" aria-label={label}>
      <div className="flex flex-wrap items-center gap-2 text-slate-500">
        {onPageSizeChange && (
          <label className="flex items-center gap-2">
            Showing
            <select
              aria-label={`Jumlah ${itemAriaLabel} per halaman`}
              value={meta.limit}
              onChange={(event) => onPageSizeChange(Number(event.target.value))}
              className="h-9 rounded-md border border-slate-900/15 bg-white px-2 text-slate-700"
            >
              {pageSizeOptions.map((size) => <option key={size} value={size}>{size}</option>)}
            </select>
            from {meta.total_data} {itemLabel}
          </label>
        )}
        {!onPageSizeChange && <span>{meta.total_data} {itemLabel}</span>}
      </div>
      <div className="flex items-center gap-2 text-slate-500">
        <span className="flex h-9 min-w-10 items-center justify-center rounded-md border border-slate-900/15 bg-white px-2 text-slate-700">
          {meta.page}
        </span>
        <span>from {totalPage} pages</span>
        <button
          type="button"
          aria-label="Sebelumnya"
          disabled={isFirst}
          onClick={() => onPageChange(meta.page - 1)}
          className="flex h-8 w-8 items-center justify-center rounded-md text-slate-500 hover:bg-slate-900/5 hover:text-slate-900 disabled:cursor-not-allowed disabled:opacity-30"
        >
          <svg aria-hidden="true" viewBox="0 0 20 20" className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="m12 5-5 5 5 5" /></svg>
        </button>
        <button
          type="button"
          aria-label="Berikutnya"
          disabled={isLast}
          onClick={() => onPageChange(meta.page + 1)}
          className="flex h-8 w-8 items-center justify-center rounded-md text-slate-500 hover:bg-slate-900/5 hover:text-slate-900 disabled:cursor-not-allowed disabled:opacity-30"
        >
          <svg aria-hidden="true" viewBox="0 0 20 20" className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="m8 5 5 5-5 5" /></svg>
        </button>
      </div>
    </nav>
  );
};
