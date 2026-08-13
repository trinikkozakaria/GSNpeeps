const colors = ["#0e7490", "#f59e0b", "#8b5cf6", "#10b981", "#ef4444", "#64748b"];

export const PieChart = ({ title, items, emptyMessage }) => {
  const total = items.reduce((sum, item) => sum + item.jumlah, 0);
  if (!total) return <p className="text-sm text-slate-500">{emptyMessage}</p>;
  let cursor = 0;
  const stops = items.map((item, index) => { const start=cursor; cursor+=(item.jumlah/total)*100; return `${colors[index%colors.length]} ${start}% ${cursor}%`; });
  return <figure><div role="img" aria-label={`${title}, total ${total} karyawan`} className="mx-auto aspect-square w-48 rounded-full" style={{background:`conic-gradient(${stops.join(",")})`}}/><figcaption className="mt-4"><ul className="grid gap-2 sm:grid-cols-2">{items.map((item,index)=><li key={item.nama} className="flex items-center gap-2 text-sm"><span className="h-3 w-3 rounded-full" style={{backgroundColor:colors[index%colors.length]}}/><span>{item.nama}: <strong>{item.jumlah.toLocaleString("id-ID")}</strong></span></li>)}</ul></figcaption></figure>;
};

export const GenderIcons = ({ items }) => <div className="grid grid-cols-2 gap-4">{items.map((item)=><div key={item.nama} className="rounded-xl bg-white p-4 text-center"><span aria-hidden="true" className="text-5xl text-cyan-700">{item.nama==="Laki-laki"?"♂":item.nama==="Perempuan"?"♀":"⚧"}</span><p className="mt-2 font-semibold">{item.nama}</p><p className="text-2xl font-bold">{item.jumlah.toLocaleString("id-ID")}</p></div>)}</div>;
