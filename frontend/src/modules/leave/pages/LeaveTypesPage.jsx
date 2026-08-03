import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";

import { DataTable } from "../../../components/data-table/DataTable";
import { FormInput } from "../../../components/form/FormInput";
import { Button } from "../../../components/ui/Button";
import { formatNumber } from "../../../lib/format";
import { useAuth } from "../../auth/hooks/useAuth";
import { useCreateLeaveType, useLeaveTypes, useUpdateLeaveType } from "../hooks/useLeave";
import { createLeaveTypeFormSchema } from "../schemas/leave-schema";

/**
 * Master jenis izin. Kontrak hanya menyediakan GET, POST, dan PUT; tidak ada operasi hapus
 * sehingga master dinonaktifkan melalui flag aktif, bukan dihapus.
 */
export const LeaveTypesPage = () => {
  document.title = "Master Jenis Izin — GSNpeeps";
  const auth = useAuth();
  const isHR = auth.role === "hr";
  const leaveTypes = useLeaveTypes(undefined, isHR);
  const createType = useCreateLeaveType();
  const updateType = useUpdateLeaveType();
  const [formError, setFormError] = useState("");

  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors, isSubmitting },
  } = useForm({
    resolver: zodResolver(createLeaveTypeFormSchema),
    defaultValues: { kode: "", nama: "", kuota_tahunan: 0, memerlukan_dokumen: true },
  });

  const onSubmit = async (values) => {
    setFormError("");
    try {
      await createType.mutateAsync(values);
      reset();
    } catch (error) {
      Object.entries(error?.fields ?? {}).forEach(([field, message]) => {
        setError(field, { type: "server", message });
      });
      setFormError(
        error?.status === 409
          ? "Kode atau nama jenis izin sudah digunakan."
          : (error?.message ?? "Jenis izin belum dapat disimpan."),
      );
    }
  };

  const toggleActive = async (type) => {
    setFormError("");
    try {
      await updateType.mutateAsync({ id: type.id, payload: { is_active: !type.is_active } });
    } catch (error) {
      setFormError(error?.message ?? "Perubahan status belum dapat disimpan.");
    }
  };

  if (!isHR) {
    return (
      <section aria-labelledby="leave-types-title">
        <h1 id="leave-types-title" className="text-3xl font-bold">Master Jenis Izin</h1>
        <p role="status" className="mt-3 text-slate-300">
          Master jenis izin hanya dapat diakses HR.
        </p>
      </section>
    );
  }

  const columns = [
    { key: "kode", header: "Kode", render: (row) => row.kode },
    { key: "nama", header: "Nama", render: (row) => row.nama },
    { key: "kuota", header: "Kuota tahunan", render: (row) => `${formatNumber(row.kuota_tahunan)} hari` },
    {
      key: "dokumen",
      header: "Dokumen",
      render: (row) => (row.memerlukan_dokumen ? "Wajib" : "Opsional"),
    },
    { key: "status", header: "Status", render: (row) => (row.is_active ? "Aktif" : "Nonaktif") },
    {
      key: "aksi",
      srHeader: "Aksi",
      cellClassName: "text-right",
      render: (row) => (
        <Button
          variant="secondary"
          className="min-h-9 px-3 py-1"
          disabled={updateType.isPending}
          onClick={() => toggleActive(row)}
        >
          {row.is_active ? "Nonaktifkan" : "Aktifkan"}
          <span className="sr-only"> {row.nama}</span>
        </Button>
      ),
    },
  ];

  return (
    <section aria-labelledby="leave-types-title">
      <p className="text-sm font-semibold uppercase tracking-widest text-cyan-300">Master</p>
      <h1 id="leave-types-title" className="mt-2 text-3xl font-bold">Master Jenis Izin</h1>
      <p className="mt-2 max-w-2xl text-slate-300">
        Daftar resmi jenis izin belum tersedia pada dokumen sumber, sehingga master diisi HR.
        Jenis izin dinonaktifkan, bukan dihapus.
      </p>

      {formError && (
        <p role="alert" className="mt-5 rounded-lg border border-rose-300/30 bg-rose-300/10 p-3 text-rose-200">
          {formError}
        </p>
      )}

      <form
        onSubmit={handleSubmit(onSubmit)}
        noValidate
        className="mt-6 grid gap-5 rounded-xl border border-white/10 bg-white/[0.03] p-5 sm:grid-cols-2"
      >
        <FormInput id="leave-type-code" label="Kode" registration={register("kode")} error={errors.kode?.message} disabled={isSubmitting} />
        <FormInput id="leave-type-name" label="Nama" registration={register("nama")} error={errors.nama?.message} disabled={isSubmitting} />
        <FormInput
          id="leave-type-quota"
          label="Kuota tahunan (hari)"
          type="number"
          min="0"
          registration={register("kuota_tahunan")}
          error={errors.kuota_tahunan?.message}
          disabled={isSubmitting}
        />
        <label className="flex items-end gap-2 text-sm text-slate-200">
          <input type="checkbox" {...register("memerlukan_dokumen")} disabled={isSubmitting} className="h-4 w-4" />
          Dokumen pendukung wajib
        </label>
        <div className="sm:col-span-2">
          <Button type="submit" disabled={isSubmitting}>
            {isSubmitting ? "Menyimpan…" : "Tambah jenis izin"}
          </Button>
        </div>
      </form>

      <div className="mt-6" aria-live="polite">
        {leaveTypes.isPending && <p role="status" className="text-slate-300">Memuat master…</p>}
        {leaveTypes.isError && (
          <div role="alert" className="rounded-xl border border-red-400/30 bg-red-400/10 p-4 text-red-100">
            <p>Master belum dapat dimuat. {leaveTypes.error.message}</p>
          </div>
        )}
        {leaveTypes.data && (
          <DataTable
            caption="Master jenis izin"
            columns={columns}
            rows={leaveTypes.data}
            rowKey={(row) => row.id}
            emptyMessage="Belum ada jenis izin. Tambahkan melalui formulir di atas."
          />
        )}
      </div>
    </section>
  );
};
