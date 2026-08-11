/**
 * Chart batang horizontal yang dapat diakses. Bar hanya dekorasi (`aria-hidden`); nilai
 * sebenarnya dibaca screen reader melalui tabel yang selalu ada di DOM, bukan alternatif
 * tersembunyi yang terpisah. Status tidak pernah dibedakan oleh warna saja karena setiap
 * baris menampilkan label dan angka.
 */
export const CompositionChart = ({ title, description, items, emptyMessage, valueLabel = "Jumlah" }) => {
  const total = items.reduce((sum, item) => sum + item.jumlah, 0);

  if (items.length === 0 || total === 0) {
    return (
      <div>
        <h3 className="text-base font-semibold text-slate-900">{title}</h3>
        {description && <p className="mt-1 text-sm text-slate-500">{description}</p>}
        <p className="mt-4 text-sm text-slate-500">{emptyMessage}</p>
      </div>
    );
  }

  return (
    <div>
      <h3 className="text-base font-semibold text-slate-900">{title}</h3>
      {description && <p className="mt-1 text-sm text-slate-500">{description}</p>}
      <table className="mt-4 w-full text-left text-sm">
        <caption className="sr-only">
          {title}. Total {total} {valueLabel.toLowerCase()}.
        </caption>
        <thead className="sr-only">
          <tr>
            <th scope="col">Kategori</th>
            <th scope="col">{valueLabel}</th>
            <th scope="col">Persentase</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => {
            const percentage = (item.jumlah / total) * 100;
            return (
              <tr key={item.nama}>
                <th scope="row" className="w-2/5 py-2 pr-3 font-medium text-slate-700">
                  {item.nama}
                </th>
                <td className="py-2 pr-3">
                  <span
                    aria-hidden="true"
                    className="block h-2 rounded-full bg-cyan-700"
                    style={{ width: `${Math.max(percentage, 2)}%` }}
                  />
                </td>
                <td className="w-24 py-2 text-right tabular-nums text-slate-600">
                  {item.jumlah.toLocaleString("id-ID")} (
                  {percentage.toLocaleString("id-ID", { maximumFractionDigits: 1 })}%)
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
};
