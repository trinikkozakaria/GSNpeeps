import { useEffect, useRef, useState } from "react";

import { Button } from "../../../components/ui/Button";
import { useBulkEmployees } from "../hooks/useEmployees";

const headers = "nip,nama,email,jenis_kelamin,tanggal_lahir,tanggal_join,department_id,position_id,atasan_id,status_pernikahan,role,jalan,kota,provinsi,nomor_ktp,nomor_kontrak,jenis_kontrak,tanggal_mulai_kontrak,tanggal_berakhir_kontrak\n";

export const EmployeeBulkUpload = ({ buttonSize = "default" }) => {
  const [open, setOpen] = useState(false);
  const [file, setFile] = useState(null);
  const bulk = useBulkEmployees();
  const dialogRef = useRef(null);
  const closeRef = useRef(null);

  useEffect(() => {
    if (!open) return undefined;
    const previousFocus = document.activeElement;
    closeRef.current?.focus();
    return () => previousFocus?.focus();
  }, [open]);

  const close = () => {
    if (bulk.isPending) return;
    setOpen(false);
    setFile(null);
    bulk.reset?.();
  };

  const handleKeyDown = (event) => {
    if (event.key === "Escape") {
      event.stopPropagation();
      close();
      return;
    }
    if (event.key !== "Tab") return;
    const focusable = dialogRef.current?.querySelectorAll(
      "button:not([disabled]), input:not([disabled]), [href], [tabindex]:not([tabindex='-1'])",
    );
    if (!focusable?.length) return;
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  };

  const downloadTemplate = () => {
    const url = URL.createObjectURL(new Blob([headers], { type: "text/csv;charset=utf-8" }));
    const link = document.createElement("a");
    link.href = url;
    link.download = "template-bulk-karyawan.csv";
    link.click();
    URL.revokeObjectURL(url);
  };

  const submit = async (event) => {
    event.preventDefault();
    if (!file || bulk.isPending) return;
    try {
      await bulk.mutateAsync(file);
    } catch {
      // Pesan error dinormalisasi hook/API dan dirender dari state mutation.
    }
  };

  return (
    <>
      <Button size={buttonSize} variant="secondary" onClick={() => setOpen(true)}>Bulk Upload</Button>
      {open && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/60 p-4">
          <form
            ref={dialogRef}
            role="dialog"
            aria-modal="true"
            aria-labelledby="bulk-title"
            aria-describedby="bulk-description"
            onKeyDown={handleKeyDown}
            onSubmit={submit}
            className="max-h-[90vh] w-full max-w-xl overflow-y-auto rounded-xl bg-white p-6 shadow-xl"
          >
            <h2 id="bulk-title" className="text-xl font-bold">Bulk upload karyawan</h2>
            <p id="bulk-description" className="mt-2 text-sm text-slate-600">
              Unggah maksimal 1.000 baris CSV. Kosongkan email agar sistem membuat alamat
              @janjikupadamu.id dari nama pertama; nama kedua digunakan bila duplikat.
            </p>
            <Button className="mt-4" variant="secondary" onClick={downloadTemplate}>
              Unduh template CSV
            </Button>
            <label htmlFor="bulk-employee-file" className="mt-4 block text-sm font-medium text-slate-700">
              Berkas CSV
            </label>
            <input
              id="bulk-employee-file"
              className="mt-2 block w-full rounded-lg border p-3"
              type="file"
              accept=".csv,text/csv"
              required
              disabled={bulk.isPending}
              onChange={(event) => setFile(event.target.files?.[0] ?? null)}
            />
            {bulk.data && (
              <div role="status" className="mt-4 rounded-lg bg-emerald-50 p-3">
                <p>{bulk.data.dibuat.length} karyawan dibuat, {bulk.data.gagal.length} gagal.</p>
                {bulk.data.dibuat.map((item) => (
                  <p key={item.baris} className="text-sm">Baris {item.baris}: {item.email}</p>
                ))}
                {bulk.data.gagal.map((item) => (
                  <p key={item.baris} className="text-sm text-amber-800">Baris {item.baris}: {item.message}</p>
                ))}
              </div>
            )}
            {bulk.isError && (
              <p role="alert" className="mt-3 text-red-700">
                Bulk upload gagal. {bulk.error.message}
              </p>
            )}
            <div className="mt-5 flex justify-end gap-2">
              <Button ref={closeRef} variant="secondary" onClick={close} disabled={bulk.isPending}>
                Tutup
              </Button>
              <Button type="submit" disabled={!file || bulk.isPending}>
                {bulk.isPending ? "Mengunggahâ€¦" : "Unggah"}
              </Button>
            </div>
          </form>
        </div>
      )}
    </>
  );
};
