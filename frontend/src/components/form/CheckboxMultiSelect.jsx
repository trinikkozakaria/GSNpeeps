import { useEffect, useRef, useState } from "react";

/**
 * Filter multi-pilih Departemen (dan sejenisnya) berbentuk dropdown: satu tombol pemicu yang
 * menampilkan ringkasan pilihan, membuka panel berisi daftar checkbox.
 *
 * `options` berbentuk `{ value, label }`. `selected` dan `onChange` bekerja pada array
 * `value`. Panel ditutup saat klik di luar atau menekan Escape. Kontrol memakai
 * `<input type="checkbox">` asli sehingga status tidak bergantung warna dan navigasi keyboard
 * tetap bekerja.
 */
export const CheckboxMultiSelect = ({
  legend,
  options = [],
  selected = [],
  onChange,
  disabled = false,
  emptyLabel = "Tidak ada opsi.",
  allLabel = "Semua",
}) => {
  const [open, setOpen] = useState(false);
  const containerRef = useRef(null);

  useEffect(() => {
    if (!open) return undefined;
    const onPointerDown = (event) => {
      if (containerRef.current && !containerRef.current.contains(event.target)) {
        setOpen(false);
      }
    };
    const onKeyDown = (event) => {
      if (event.key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("mousedown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [open]);

  const toggle = (value) => {
    const next = new Set(selected);
    if (next.has(value)) next.delete(value);
    else next.add(value);
    onChange([...next]);
  };

  const summary =
    selected.length === 0
      ? allLabel
      : selected.length === 1
        ? (options.find((option) => option.value === selected[0])?.label ?? "1 dipilih")
        : `${selected.length} dipilih`;

  return (
    <div className="text-sm font-medium text-slate-700" ref={containerRef}>
      <span className="mb-2 block">{legend}</span>
      <div className="relative">
        <button
          type="button"
          disabled={disabled}
          onClick={() => setOpen((value) => !value)}
          aria-haspopup="listbox"
          aria-expanded={open}
          aria-label={`${legend}: ${summary}`}
          className="flex min-h-10 w-full items-center justify-between gap-2 rounded-lg border border-slate-900/15 bg-white px-3 text-left font-normal text-slate-900 outline-none focus:border-cyan-300 disabled:cursor-not-allowed disabled:opacity-60"
        >
          <span className="truncate">{summary}</span>
          <svg
            aria-hidden="true"
            viewBox="0 0 20 20"
            className={`h-4 w-4 shrink-0 text-slate-500 transition ${open ? "rotate-180" : ""}`}
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="m5 8 5 5 5-5" />
          </svg>
        </button>
        {open && (
          <div
            role="listbox"
            aria-multiselectable="true"
            aria-label={legend}
            className="absolute z-20 mt-1 max-h-56 w-full overflow-y-auto rounded-lg border border-slate-900/15 bg-white p-2 shadow-lg"
          >
            {options.length === 0 ? (
              <p className="px-1 py-1 text-xs text-slate-500">{emptyLabel}</p>
            ) : (
              options.map((option) => (
                <label
                  key={option.value}
                  className="flex items-center gap-2 rounded px-1 py-1 font-normal text-slate-700 hover:bg-slate-50"
                >
                  <input
                    type="checkbox"
                    className="h-4 w-4 rounded border-slate-400 text-cyan-700 focus:ring-cyan-300"
                    checked={selected.includes(option.value)}
                    onChange={() => toggle(option.value)}
                  />
                  <span>{option.label}</span>
                </label>
              ))
            )}
          </div>
        )}
      </div>
      {selected.length > 0 && (
        <button
          type="button"
          onClick={() => onChange([])}
          className="mt-1.5 text-xs font-semibold text-cyan-700 hover:text-cyan-800"
        >
          Bersihkan pilihan ({selected.length})
        </button>
      )}
    </div>
  );
};
