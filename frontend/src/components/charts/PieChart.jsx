import { useState } from "react";

const colorForIndex = (index) => `hsl(${(index * 137.508 + 214) % 360} 68% 45%)`;

export const PieChart = ({ title, items, emptyMessage }) => {
  const [activeSegment, setActiveSegment] = useState(null);
  const total = items.reduce((sum, item) => sum + item.jumlah, 0);
  if (!total) return <p className="text-sm text-slate-500">{emptyMessage}</p>;
  const radius = 42;
  const circumference = 2 * Math.PI * radius;
  let cursor = 0;
  const segments = items.map((item, index) => {
    const percentage = (item.jumlah / total) * 100;
    const start = cursor;
    cursor += percentage;
    const angle = ((start + percentage / 2) / 100) * Math.PI * 2 - Math.PI / 2;
    return {
      ...item,
      color: colorForIndex(index),
      percentage,
      dash: (percentage / 100) * circumference,
      offset: -(start / 100) * circumference,
      labelX: 60 + Math.cos(angle) * radius,
      labelY: 60 + Math.sin(angle) * radius,
    };
  });

  return (
    <figure>
      <div className="relative mx-auto w-72 max-w-full">
        <svg role="img" aria-label={`${title}, total ${total} karyawan`} viewBox="0 0 120 120" className="aspect-square w-full overflow-visible">
          <circle cx="60" cy="60" r={radius} fill="none" stroke="#e2e8f0" strokeWidth="22" />
          {segments.map((segment) => (
            <circle
              key={segment.nama}
              cx="60"
              cy="60"
              r={radius}
              fill="none"
              stroke={segment.color}
              strokeWidth={activeSegment?.nama === segment.nama ? 24 : 22}
              strokeDasharray={`${segment.dash} ${circumference - segment.dash}`}
              strokeDashoffset={segment.offset}
              transform="rotate(-90 60 60)"
              tabIndex="0"
              role="graphics-symbol"
              aria-label={`${segment.nama}: ${segment.jumlah.toLocaleString("id-ID")} karyawan, ${segment.percentage.toLocaleString("id-ID", { maximumFractionDigits: 1 })} persen`}
              className="cursor-pointer transition-[stroke-width] focus:outline-none"
              onMouseEnter={() => setActiveSegment(segment)}
              onMouseLeave={() => setActiveSegment(null)}
              onFocus={() => setActiveSegment(segment)}
              onBlur={() => setActiveSegment(null)}
            />
          ))}
          <text x="60" y="57" textAnchor="middle" className="pointer-events-none fill-slate-500 text-[7px]">Total</text>
          <text x="60" y="66" textAnchor="middle" className="pointer-events-none fill-slate-900 text-[10px] font-bold">{total.toLocaleString("id-ID")}</text>
          {segments.map((segment) => (
            <text
              key={`${segment.nama}-label`}
              x={segment.labelX}
              y={segment.labelY}
              textAnchor="middle"
              dominantBaseline="central"
              className="pointer-events-none fill-slate-950 text-[5px] font-bold"
              style={{ paintOrder: "stroke", stroke: "white", strokeWidth: 1.5 }}
            >
              {segment.percentage.toLocaleString("id-ID", { maximumFractionDigits: 1 })}%
            </text>
          ))}
        </svg>
        {activeSegment && (
          <div
            role="tooltip"
            className="pointer-events-none absolute z-10 -translate-x-1/2 -translate-y-full whitespace-nowrap rounded-lg bg-slate-950 px-3 py-2 text-xs font-medium text-white shadow-lg"
            style={{
              left: `${Math.min(88, Math.max(12, (activeSegment.labelX / 120) * 100))}%`,
              top: `${Math.min(92, Math.max(12, (activeSegment.labelY / 120) * 100))}%`,
            }}
          >
            {activeSegment.nama}: {activeSegment.jumlah.toLocaleString("id-ID")} karyawan ({activeSegment.percentage.toLocaleString("id-ID", { maximumFractionDigits: 1 })}%)
          </div>
        )}
      </div>
      <figcaption className="mt-4">
        <ul className="grid gap-2 sm:grid-cols-2">
          {segments.map((segment) => (
            <li key={segment.nama} className="flex items-center gap-2 text-sm">
              <span className="h-3 w-3 shrink-0 rounded-full" style={{ backgroundColor: segment.color }} />
              <span>{segment.nama}: <strong>{segment.jumlah.toLocaleString("id-ID")}</strong> ({segment.percentage.toLocaleString("id-ID", { maximumFractionDigits: 1 })}%)</span>
            </li>
          ))}
        </ul>
      </figcaption>
    </figure>
  );
};

export const GenderIcons = ({ items }) => (
  <ul aria-label="Populasi aktif menurut gender" className="grid grid-cols-2 gap-4">
    {items.map((item) => (
      <li
        key={item.nama}
        aria-label={`${item.nama}: ${item.jumlah.toLocaleString("id-ID")}`}
        className="rounded-xl bg-white p-4 text-center"
      >
        <span aria-hidden="true" className="text-5xl text-cyan-700">
          {item.nama === "Laki-laki" ? "♂" : item.nama === "Perempuan" ? "♀" : "⚧"}
        </span>
        <p className="mt-2 font-semibold">{item.nama}</p>
        <p className="text-2xl font-bold">{item.jumlah.toLocaleString("id-ID")}</p>
      </li>
    ))}
  </ul>
);
