import { useMutation, useQuery } from "@tanstack/react-query";
import { useState } from "react";

import { Button } from "../../../components/ui/Button";
import { queryClient } from "../../../lib/query/query-client";
import { createDocumentTypeRequest, documentTypesRequest } from "../api/uat-api";

export const DocumentTypesPage = () => {
  const [code, setCode] = useState("");
  const [name, setName] = useState("");
  const [required, setRequired] = useState(true);
  const [successMessage, setSuccessMessage] = useState("");

  const types = useQuery({
    queryKey: ["document-types"],
    queryFn: ({ signal }) => documentTypesRequest(signal),
  });
  const create = useMutation({
    mutationFn: createDocumentTypeRequest,
    onSuccess: async () => {
      setCode("");
      setName("");
      setRequired(true);
      setSuccessMessage("Jenis dokumen berhasil ditambahkan.");
      await queryClient.invalidateQueries({ queryKey: ["document-types"] });
    },
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
              <span>
                <strong>{item.nama}</strong> <small>({item.kode})</small>
              </span>
              <span>{item.wajib ? "Wajib" : "Opsional"}</span>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
};
