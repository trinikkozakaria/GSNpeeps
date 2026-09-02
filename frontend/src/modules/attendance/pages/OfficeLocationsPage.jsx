import { useState } from "react";

import { DataTable } from "../../../components/data-table/DataTable";
import { Button } from "../../../components/ui/Button";
import { useCreateOfficeLocation, useDeactivateOfficeLocation, useOfficeLocations, useUpdateOfficeLocation } from "../hooks/useAttendance";

const empty = { kode: "", nama: "", alamat: "", latitude: "", longitude: "", is_active: true };
const payload = (value) => ({ ...value, kode: value.kode.trim(), nama: value.nama.trim(), alamat: value.alamat.trim() || null, latitude: Number(value.latitude), longitude: Number(value.longitude) });

export const OfficeLocationsPage = () => {
  const offices = useOfficeLocations(true);
  const create = useCreateOfficeLocation();
  const update = useUpdateOfficeLocation();
  const deactivate = useDeactivateOfficeLocation();
  const [form, setForm] = useState(empty);
  const [editing, setEditing] = useState(null);
  const [error, setError] = useState("");
  const submit = async (event) => { event.preventDefault(); setError(""); try { if (editing) { await update.mutateAsync({ id: editing.id, payload: payload(form) }); setEditing(null); } else { await create.mutateAsync(payload(form)); } setForm(empty); } catch (err) { setError(err.message ?? "Lokasi kantor belum dapat disimpan."); } };
  const columns = [
    { key: "kode", header: "Kode", render: (row) => row.kode }, { key: "nama", header: "Nama", render: (row) => row.nama }, { key: "alamat", header: "Alamat", render: (row) => row.alamat || "—" }, { key: "koordinat", header: "Koordinat", render: (row) => `${row.latitude}, ${row.longitude}` },
    { key: "aksi", srHeader: "Aksi", render: (row) => <div className="flex gap-2"><Button variant="secondary" className="min-h-9 px-3 py-1" onClick={() => { setEditing(row); setForm({ ...row, alamat: row.alamat ?? "", latitude: String(row.latitude), longitude: String(row.longitude) }); }}>Edit</Button><Button variant="secondary" className="min-h-9 px-3 py-1" disabled={deactivate.isPending} onClick={() => deactivate.mutate(row.id)}>Nonaktifkan</Button></div> },
  ];
  return <section aria-labelledby="office-locations-title"><p className="text-sm font-semibold uppercase tracking-widest text-cyan-700">Master</p><h1 id="office-locations-title" className="mt-2 text-3xl font-bold">Master Lokasi Kantor</h1><p className="mt-2 text-slate-600">Lokasi aktif tersedia untuk pilihan WFO. Menonaktifkan lokasi tidak menghapus riwayat absensi.</p>{error && <p role="alert" className="mt-4 text-rose-700">{error}</p>}<form onSubmit={submit} className="mt-6 grid gap-3 rounded-xl border p-5 sm:grid-cols-2"><label>Kode<input required value={form.kode} onChange={(e) => setForm({ ...form, kode: e.target.value })} className="mt-1 block w-full rounded border p-2" /></label><label>Nama<input required value={form.nama} onChange={(e) => setForm({ ...form, nama: e.target.value })} className="mt-1 block w-full rounded border p-2" /></label><label>Alamat<input value={form.alamat} onChange={(e) => setForm({ ...form, alamat: e.target.value })} className="mt-1 block w-full rounded border p-2" /></label><label>Latitude<input required type="number" step="any" min="-90" max="90" value={form.latitude} onChange={(e) => setForm({ ...form, latitude: e.target.value })} className="mt-1 block w-full rounded border p-2" /></label><label>Longitude<input required type="number" step="any" min="-180" max="180" value={form.longitude} onChange={(e) => setForm({ ...form, longitude: e.target.value })} className="mt-1 block w-full rounded border p-2" /></label><div className="flex items-end gap-2"><Button type="submit" disabled={create.isPending || update.isPending}>{editing ? "Simpan lokasi" : "Tambah lokasi"}</Button>{editing && <Button type="button" variant="secondary" onClick={() => { setEditing(null); setForm(empty); }}>Batal</Button>}</div></form><div className="mt-6">{offices.isPending ? <p role="status">Memuat lokasi kantor…</p> : <DataTable caption="Lokasi kantor aktif" columns={columns} rows={offices.data ?? []} rowKey={(row) => row.id} emptyMessage="Belum ada lokasi kantor aktif." />}</div></section>;
};
