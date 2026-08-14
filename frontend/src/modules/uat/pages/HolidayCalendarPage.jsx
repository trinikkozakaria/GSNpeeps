import { useMutation, useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";

import { Button } from "../../../components/ui/Button";
import { queryClient } from "../../../lib/query/query-client";
import { useAuth } from "../../auth/hooks/useAuth";
import { holidaysRequest, upsertHolidaysRequest } from "../api/uat-api";

const monthNames = ["Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"];
const dayNames = ["Sen", "Sel", "Rab", "Kam", "Jum", "Sab", "Min"];
const emptyHoliday = () => ({ tanggal: "", nama: "", keterangan: "" });

export const MonthGrid = ({ year, month, holidays }) => {
  const cells = useMemo(() => {
    const first = new Date(Date.UTC(year, month, 1));
    const offset = (first.getUTCDay() + 6) % 7;
    const count = new Date(Date.UTC(year, month + 1, 0)).getUTCDate();
    return [...Array(offset).fill(null), ...Array.from({ length: count }, (_, index) => index + 1)];
  }, [month, year]);
  const indexed = new Map(holidays.map((item) => [item.tanggal, item]));

  return (
    <div className="overflow-x-auto rounded-xl border p-4">
      <h2 className="text-xl font-bold">{monthNames[month]} {year}</h2>
      <div className="mt-4 grid min-w-[42rem] grid-cols-7 gap-1">
        {dayNames.map((day) => <div key={day} className="p-2 text-center text-xs font-bold text-slate-600">{day}</div>)}
        {cells.map((day, index) => {
          if (!day) return <span key={`empty-${index}`} aria-hidden="true" />;
          const date = `${year}-${String(month + 1).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
          const holiday = indexed.get(date);
          return (
            <div key={date} title={holiday?.nama} className={`min-h-20 rounded-lg border p-2 ${holiday ? "border-red-300 bg-red-100 text-red-900" : "border-slate-100"}`}>
              <span className="font-bold">{day}</span>
              {holiday && <span className="mt-1 block text-xs font-semibold">{holiday.nama}</span>}
            </div>
          );
        })}
      </div>
    </div>
  );
};

export const HolidayCalendarPage = () => {
  const auth = useAuth();
  const today = new Date();
  const [year, setYear] = useState(today.getFullYear());
  const [month, setMonth] = useState(today.getMonth());
  const [rows, setRows] = useState([emptyHoliday()]);
  const [successMessage, setSuccessMessage] = useState("");
  const holidays = useQuery({
    queryKey: ["holidays", year],
    queryFn: ({ signal }) => holidaysRequest(year, signal),
  });
  const save = useMutation({ mutationFn: upsertHolidaysRequest });
  const rowsValid = rows.every((row) => row.tanggal && row.nama.trim());

  const update = (index, field, value) => {
    setSuccessMessage("");
    setRows((current) => current.map((row, rowIndex) =>
      rowIndex === index ? { ...row, [field]: value } : row));
  };

  const handleSave = async (event) => {
    event.preventDefault();
    if (!rowsValid || save.isPending) return;
    setSuccessMessage("");
    try {
      await save.mutateAsync(rows.map((row) => ({
        tanggal: row.tanggal,
        nama: row.nama.trim(),
        keterangan: row.keterangan.trim() || null,
      })));
      setRows([emptyHoliday()]);
      setSuccessMessage("Hari libur berhasil disimpan.");
      await queryClient.invalidateQueries({ queryKey: ["holidays", year] });
    } catch {
      // State error mutation dirender di bawah form.
    }
  };

  return (
    <section aria-labelledby="holiday-title">
      <p className="text-sm font-semibold uppercase tracking-widest text-cyan-700">Kalender</p>
      <h1 id="holiday-title" className="mt-2 text-3xl font-bold">Kalender Hari Libur</h1>
      <div className="my-5 flex flex-wrap gap-3">
        <label className="text-sm">Tahun<input type="number" value={year} min="1900" max="2200" onChange={(event) => setYear(Number(event.target.value))} className="mt-2 block min-h-10 rounded-lg border px-3" /></label>
        <label className="text-sm">Bulan<select value={month} onChange={(event) => setMonth(Number(event.target.value))} className="mt-2 block min-h-10 rounded-lg border bg-white px-3">{monthNames.map((name, index) => <option key={name} value={index}>{name}</option>)}</select></label>
      </div>

      {holidays.isPending && <p role="status">Memuat kalender…</p>}
      {holidays.isError && (
        <div role="alert" className="rounded-xl border border-red-300 bg-red-50 p-4 text-red-700">
          <p>Kalender belum dapat dimuat. {holidays.error?.message}</p>
          <Button className="mt-3" variant="secondary" onClick={() => holidays.refetch()}>Coba lagi</Button>
        </div>
      )}
      {!holidays.isPending && !holidays.isError && <MonthGrid year={year} month={month} holidays={holidays.data ?? []} />}

      {auth.role === "hr" && (
        <form className="my-6 grid gap-3 rounded-xl border p-5" onSubmit={handleSave}>
          <h2 className="font-bold">Bulk insert / update hari libur</h2>
          {rows.map((row, index) => (
            <div key={index} className="grid gap-2 sm:grid-cols-3">
              <input aria-label={`Tanggal ${index + 1}`} type="date" required value={row.tanggal} onChange={(event) => update(index, "tanggal", event.target.value)} className="min-h-10 rounded-lg border px-3" />
              <input aria-label={`Nama ${index + 1}`} required placeholder="Nama hari libur" value={row.nama} onChange={(event) => update(index, "nama", event.target.value)} className="min-h-10 rounded-lg border px-3" />
              <input aria-label={`Keterangan ${index + 1}`} placeholder="Keterangan" value={row.keterangan} onChange={(event) => update(index, "keterangan", event.target.value)} className="min-h-10 rounded-lg border px-3" />
            </div>
          ))}
          <div className="flex flex-wrap gap-2">
            <Button variant="secondary" disabled={save.isPending} onClick={() => setRows((current) => [...current, emptyHoliday()])}>Tambah baris</Button>
            {rows.length > 1 && <Button variant="secondary" disabled={save.isPending} onClick={() => setRows((current) => current.slice(0, -1))}>Hapus baris terakhir</Button>}
            <Button type="submit" disabled={!rowsValid || save.isPending}>{save.isPending ? "Menyimpan…" : "Simpan sekaligus"}</Button>
          </div>
          {save.isError && <p role="alert" className="text-red-700">Hari libur belum dapat disimpan. {save.error?.message}</p>}
          {successMessage && <p role="status" className="text-emerald-700">{successMessage}</p>}
        </form>
      )}
    </section>
  );
};
