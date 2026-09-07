import { useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";

import { Pagination } from "../../../components/data-table/Pagination";
import { Button } from "../../../components/ui/Button";
import { formatDate, formatDateTime } from "../../../lib/format";
import { requestStatusLabel } from "../../../lib/request-status";
import { useAuth } from "../../auth/hooks/useAuth";
import {
  useCreateAttendanceCorrection,
  useMyAttendanceCorrections,
} from "../hooks/useAttendanceCorrections";

const emptyForm = { tanggal: "", waktu_check_in: "", waktu_check_out: "", alasan: "" };

const parsePositiveInteger = (value, fallback) => {
  const parsed = Number.parseInt(value ?? "", 10);
  return Number.isInteger(parsed) && parsed > 0 ? parsed : fallback;
};

// Halaman pribadi: hanya pengajuan + riwayat koreksi milik user sendiri, setara
// MyRequestsPage untuk ketidakhadiran/lembur. Antrean keputusan untuk pengajuan orang
// lain ada di halaman Persetujuan (ApprovalInboxPage), bukan di sini.
export const AttendanceCorrectionPage = () => {
  const auth = useAuth();
  const [form, setForm] = useState(emptyForm);
  const [successMessage, setSuccessMessage] = useState("");
  const [params, setParams] = useSearchParams();
  const filters = useMemo(
    () => ({
      page: parsePositiveInteger(params.get("page"), 1),
      limit: parsePositiveInteger(params.get("limit"), 10),
    }),
    [params],
  );
  const corrections = useMyAttendanceCorrections(auth.user?.id, filters);
  const create = useCreateAttendanceCorrection();
  const formValid = Boolean(
    form.tanggal &&
      (form.waktu_check_in || form.waktu_check_out) &&
      form.alasan.trim().length >= 10,
  );

  const updateForm = (field, value) => setForm((current) => ({ ...current, [field]: value }));

  const handleSubmit = async (event) => {
    event.preventDefault();
    if (!formValid || create.isPending) return;
    setSuccessMessage("");
    try {
      await create.mutateAsync({
        ...form,
        alasan: form.alasan.trim(),
        waktu_check_in: form.waktu_check_in || null,
        waktu_check_out: form.waktu_check_out || null,
      });
      setForm(emptyForm);
      setSuccessMessage("Koreksi absensi berhasil diajukan.");
    } catch {
      // State error mutation dirender di bawah form.
    }
  };

  return (
    <section aria-labelledby="correction-title">
      <p className="text-sm font-semibold uppercase tracking-widest text-cyan-700">Absensi</p>
      <h1 id="correction-title" className="mt-2 text-3xl font-bold">Koreksi Absensi</h1>
      <p className="mt-2 text-slate-600">Perubahan jam mengikuti persetujuan Atasan lalu HR.</p>

      <form onSubmit={handleSubmit} className="my-6 grid gap-4 rounded-xl border p-5 sm:grid-cols-3">
        <label className="text-sm">
          Tanggal
          <input required type="date" value={form.tanggal} onChange={(event) => updateForm("tanggal", event.target.value)} className="mt-2 block min-h-10 w-full rounded-lg border px-3" />
        </label>
        <label className="text-sm">
          Jam masuk
          <input type="time" value={form.waktu_check_in} onChange={(event) => updateForm("waktu_check_in", event.target.value)} className="mt-2 block min-h-10 w-full rounded-lg border px-3" />
        </label>
        <label className="text-sm">
          Jam pulang
          <input type="time" value={form.waktu_check_out} onChange={(event) => updateForm("waktu_check_out", event.target.value)} className="mt-2 block min-h-10 w-full rounded-lg border px-3" />
        </label>
        <label className="text-sm sm:col-span-3">
          Alasan
          <textarea required minLength="10" value={form.alasan} onChange={(event) => updateForm("alasan", event.target.value)} className="mt-2 block min-h-24 w-full rounded-lg border p-3" />
          <span className="mt-1 block text-xs text-slate-500">Minimal 10 karakter dan pilih setidaknya satu jam koreksi.</span>
        </label>
        <Button type="submit" disabled={!formValid || create.isPending}>
          {create.isPending ? "Mengajukan…" : "Ajukan koreksi"}
        </Button>
        {create.isError && <p role="alert" className="sm:col-span-3 text-red-700">Koreksi belum dapat diajukan. {create.error?.message}</p>}
        {successMessage && <p role="status" className="sm:col-span-3 text-emerald-700">{successMessage}</p>}
      </form>

      <section aria-labelledby="correction-history-title" className="mt-7">
        <h2 id="correction-history-title" className="text-xl font-bold">
          Riwayat koreksi saya
        </h2>
        {corrections.isPending && <p role="status" className="mt-4">Memuat koreksi…</p>}
        {corrections.isError && (
          <div role="alert" className="mt-4 rounded-xl border border-red-300 bg-red-50 p-4 text-red-700">
            <p>Koreksi belum dapat dimuat. {corrections.error?.message}</p>
            <Button className="mt-3" variant="secondary" onClick={() => corrections.refetch()}>Coba lagi</Button>
          </div>
        )}
        {!corrections.isPending && !corrections.isError && (corrections.data?.items?.length ?? 0) === 0 && (
          <p className="mt-4 rounded-xl border border-dashed p-5 text-slate-600">Belum ada koreksi absensi.</p>
        )}
        <div className="mt-4 grid gap-3">
          {(corrections.data?.items ?? []).map((item) => (
            <article key={item.id} className="rounded-xl border p-4">
              <h3 className="font-bold">{formatDate(item.tanggal)}</h3>
              <p className="text-sm">Masuk {item.waktu_check_in || "—"} · Pulang {item.waktu_check_out || "—"}</p>
              <p className="mt-2 text-sm text-slate-600">{item.alasan}</p>
              <p className="mt-2 text-xs text-slate-500">{formatDateTime(item.created_at)} · {requestStatusLabel[item.status] ?? item.status}</p>
            </article>
          ))}
        </div>
        {corrections.data && corrections.data.items.length > 0 && (
          <Pagination
            meta={corrections.data.meta}
            label="Navigasi halaman koreksi absensi"
            itemLabel="koreksi"
            onPageChange={(nextPage) => {
              const next = new URLSearchParams(params);
              next.set("page", String(nextPage));
              setParams(next);
            }}
          />
        )}
      </section>
    </section>
  );
};
