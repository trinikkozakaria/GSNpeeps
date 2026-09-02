import { useMutation, useQuery } from "@tanstack/react-query";
import { useState } from "react";

import { Button } from "../../../components/ui/Button";
import { queryClient } from "../../../lib/query/query-client";
import { createDocumentTypeRequest, deleteDocumentTypeRequest, documentTypesRequest, updateDocumentTypeRequest } from "../api/uat-api";

export const DocumentTypesPage = () => {
  const [code, setCode] = useState("");
  const [name, setName] = useState("");
  const [required, setRequired] = useState(true);
  const [successMessage, setSuccessMessage] = useState("");
	const [editing, setEditing] = useState(null);

  const types = useQuery({
    queryKey: ["document-types"],
    queryFn: ({ signal }) => documentTypesRequest(signal),
  });
  const create = useMutation({
    mutationKey: ["create"],
    mutationFn: createDocumentTypeRequest,
    onSuccess: async () => {
      setCode("");
      setName("");
      setRequired(true);
      setSuccessMessage("Jenis dokumen berhasil ditambahkan.");
      await queryClient.invalidateQueries({ queryKey: ["document-types"] });
    },
  });
	const update = useMutation({
		mutationKey: ["update"],
		mutationFn: ({ id, payload }) => updateDocumentTypeRequest(id, payload),
		onSuccess: async () => { setEditing(null); setSuccessMessage("Jenis dokumen berhasil diperbarui."); await queryClient.invalidateQueries({ queryKey: ["document-types"] }); },
	});
	const deactivate = useMutation({
		mutationKey: ["deactivate"],
		mutationFn: deleteDocumentTypeRequest,
		onSuccess: async () => { setSuccessMessage("Jenis dokumen berhasil dinonaktifkan."); await queryClient.invalidateQueries({ queryKey: ["document-types"] }); },
	});

  const trimmedCode = code.trim();
  const trimmedName = name.trim();
  const canSubmit = Boolean(trimmedCode && trimmedName) && !create.isPending;

  const handleSubmit = (event) => {
    event.preventDefault();
    if (!canSubmit) return;
    setSuccessMessage("");
    create.mutate({ kode: trimmedCode, nama: trimmedName, wajib: required });
  };

  return (
    <section>
      <p className="text-sm font-semibold uppercase tracking-widest text-cyan-700">Master</p>
      <h1 className="mt-2 text-3xl font-bold">Jenis Dokumen Karyawan</h1>
      <p className="mt-2 text-slate-600">Jenis ini berlaku seragam untuk semua karyawan.</p>

      <form
        className="my-6 flex flex-wrap items-end gap-3 rounded-xl border p-5"
        onSubmit={handleSubmit}
      >
        <label className="text-sm">
          Kode
          <input
            required
            value={code}
            onChange={(event) => setCode(event.target.value)}
            className="mt-2 block min-h-10 rounded-lg border px-3"
          />
        </label>
        <label className="text-sm">
          Nama
          <input
            required
            value={name}
            onChange={(event) => setName(event.target.value)}
            className="mt-2 block min-h-10 rounded-lg border px-3"
          />
        </label>
        <label className="flex min-h-10 items-center gap-2">
          <input
            type="checkbox"
            checked={required}
            onChange={(event) => setRequired(event.target.checked)}
          />
          Wajib
        </label>
        <Button type="submit" disabled={!canSubmit}>
          {create.isPending ? "Menambahkan…" : "Tambah"}
        </Button>
        {create.isError && (
          <p role="alert" className="w-full text-sm text-red-700">
            Jenis dokumen belum dapat ditambahkan. Periksa apakah kode atau nama sudah digunakan.
          </p>
        )}
        {successMessage && (
          <p role="status" className="w-full text-sm text-emerald-700">
            {successMessage}
          </p>
        )}
      </form>

      {types.isPending && <p role="status">Memuat jenis dokumen…</p>}
      {types.isError && (
        <p role="alert" className="text-red-700">
          Jenis dokumen belum dapat dimuat.
        </p>
      )}
      {!types.isPending && !types.isError && (types.data?.length ?? 0) === 0 && (
        <p className="rounded-xl border border-dashed p-5 text-slate-600">
          Belum ada jenis dokumen.
        </p>
      )}
      {!types.isPending && !types.isError && (types.data?.length ?? 0) > 0 && (
        <ul className="divide-y rounded-xl border">
          {types.data.map((item) => (
            <li key={item.id} className="flex justify-between gap-4 p-4">
				{editing?.id === item.id ? (
				  <form className="flex flex-1 flex-wrap items-end gap-2" onSubmit={(event) => { event.preventDefault(); update.mutate({ id: item.id, payload: { kode: editing.kode.trim(), nama: editing.nama.trim(), wajib: editing.wajib, is_active: editing.is_active } }); }}>
					<label className="text-sm">Kode<input required value={editing.kode} onChange={(event) => setEditing({ ...editing, kode: event.target.value })} className="ml-2 rounded border px-2 py-1" /></label>
					<label className="text-sm">Nama<input required value={editing.nama} onChange={(event) => setEditing({ ...editing, nama: event.target.value })} className="ml-2 rounded border px-2 py-1" /></label>
					<label className="text-sm"><input type="checkbox" checked={editing.wajib} onChange={(event) => setEditing({ ...editing, wajib: event.target.checked })} /> Wajib</label>
					<label className="text-sm"><input type="checkbox" checked={editing.is_active} onChange={(event) => setEditing({ ...editing, is_active: event.target.checked })} /> Aktif</label>
					<Button type="submit" disabled={update.isPending}>Simpan</Button><Button type="button" variant="secondary" onClick={() => setEditing(null)}>Batal</Button>
				  </form>
				) : <>
				  <span><strong>{item.nama}</strong> <small>({item.kode})</small></span>
				  <span className="flex items-center gap-2"><span>{item.wajib ? "Wajib" : "Opsional"}{!item.is_active && " · Nonaktif"}</span><Button type="button" variant="secondary" onClick={() => setEditing(item)}>Edit</Button>{item.is_active && <Button type="button" variant="secondary" disabled={deactivate.isPending} onClick={() => deactivate.mutate(item.id)}>Nonaktifkan</Button>}</span>
				</>}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
};
