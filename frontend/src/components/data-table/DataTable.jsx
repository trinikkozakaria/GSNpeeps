/**
 * Tabel data generik dengan perilaku responsif: tabel penuh pada layar lebar dan daftar
 * kartu pada layar sempit, memakai data yang sama tanpa duplikasi sumber.
 *
 * `columns` berisi { key, header, srHeader, render, headerClassName, cellClassName }.
 * Kolom tanpa `header` memakai `srHeader` sebagai label khusus screen reader.
 */
export const DataTable = ({ caption, columns, rows, rowKey, emptyMessage }) => {
  if (rows.length === 0) {
    return (
      <div className="rounded-xl border border-slate-900/10 p-8 text-center text-slate-600">
        {emptyMessage}
      </div>
    );
  }

  return (
    <>
      <div
        className="hidden overflow-x-auto rounded-xl border border-slate-900/10 md:block"
        role="region"
        aria-label={caption}
        tabIndex={0}
      >
        <table className="min-w-full divide-y divide-slate-900/10 text-left text-sm">
          <caption className="sr-only">{caption}</caption>
          <thead className="bg-slate-900/5 text-xs uppercase tracking-wider text-slate-500">
            <tr>
              {columns.map((column) => (
                <th key={column.key} scope="col" className={`px-4 py-3 ${column.headerClassName ?? ""}`}>
                  {column.header ?? <span className="sr-only">{column.srHeader ?? "Aksi"}</span>}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-900/10">
            {rows.map((row) => (
              <tr key={rowKey(row)} className="bg-slate-50">
                {columns.map((column) => (
                  <td key={column.key} className={`px-4 py-4 ${column.cellClassName ?? ""}`}>
                    {column.render(row)}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <ul className="grid gap-3 md:hidden" aria-label={caption}>
        {rows.map((row) => (
          <li key={rowKey(row)} className="rounded-xl border border-slate-900/10 bg-slate-50 p-4">
            <dl className="grid gap-2">
              {columns.map((column) => (
                <div key={column.key} className="flex flex-wrap items-baseline justify-between gap-2">
                  <dt className="text-xs font-semibold uppercase tracking-wider text-slate-500">
                    {column.header ?? column.srHeader ?? "Aksi"}
                  </dt>
                  <dd className="text-sm text-slate-900">{column.render(row)}</dd>
                </div>
              ))}
            </dl>
          </li>
        ))}
      </ul>
    </>
  );
};
