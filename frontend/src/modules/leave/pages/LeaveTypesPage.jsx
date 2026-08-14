import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";

import { DataTable } from "../../../components/data-table/DataTable";
import { FormInput } from "../../../components/form/FormInput";
import { Button } from "../../../components/ui/Button";
import { formatNumber } from "../../../lib/format";
import { useAuth } from "../../auth/hooks/useAuth";
import { useCreateLeaveType, useLeaveTypes, useUpdateLeaveType } from "../hooks/useLeave";
import { createLeaveTypeFormSchema, updateLeaveTypeFormSchema } from "../schemas/leave-schema";
import { getLeavePolicy } from "../data/sop-leave-policy";

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
  const [editingType, setEditingType] = useState(null);

  const {
    register,
    handleSubmit,
    reset,
    setError,
    watch,
    formState: { errors, isSubmitting },
  } = useForm({
    resolver: zodResolver(createLeaveTypeFormSchema),
    defaultValues: { kode: "", nama: "", kategori: "cuti", kuota_tahunan: 0, maksimal_hari: 0, memerlukan_dokumen: true },
  });
  const createCategory = watch("kategori");

  const editForm = useForm({
    resolver: zodResolver(updateLeaveTypeFormSchema),
    defaultValues: { nama: "", kategori: "cuti", kuota_tahunan: 0, maksimal_hari: 0, memerlukan_dokumen: true },
  });

  useEffect(() => {
    if (!editingType) return undefined;
    editForm.setFocus("nama");
    const onKeyDown = (event) => {
      if (event.key === "Escape" && !updateType.isPending) setEditingType(null);
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [editForm, editingType, updateType.isPending]);

  const onSubmit = async (values) => {
    setFormError("");
    try {
      await createType.mutateAsync({
        ...values,
        kuota_tahunan: values.kategori === "izin" ? 0 : values.kuota_tahunan,
        maksimal_hari: values.kategori === "izin" ? values.maksimal_hari : null,
      });
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

  const openEditor = (type) => {
    setFormError("");
    editForm.clearErrors();
    editForm.reset({
      nama: type.nama,
      kategori: type.kategori,
      kuota_tahunan: type.kuota_tahunan,
      maksimal_hari: type.maksimal_hari ?? 0,
      memerlukan_dokumen: type.memerlukan_dokumen,
    });
    setEditingType(type);
  };

  const submitEdit = async (values) => {
    try {
      await updateType.mutateAsync({
        id: editingType.id,
        payload: {
          nama: values.nama,
          kuota_tahunan: editingType.kategori === "cuti" ? values.kuota_tahunan : 0,
          maksimal_hari: editingType.kategori === "izin" ? values.maksimal_hari : undefined,
          memerlukan_dokumen: values.memerlukan_dokumen,
        },
      });
      setEditingType(null);
    } catch (error) {
      Object.entries(error?.fields ?? {}).forEach(([field, message]) => {
        editForm.setError(field, { type: "server", message });
      });
      editForm.setError("root", {
        type: "server",
        message: error?.status === 409
          ? "Nama jenis izin sudah digunakan."
          : (error?.message ?? "Perubahan jenis izin belum dapat disimpan."),
      });
    }
  };

  if (!isHR) {
    return (
      <section aria-labelledby="leave-types-title">
        <h1 id="leave-types-title" className="text-3xl font-bold">Master Jenis Izin</h1>
        <p role="status" className="mt-3 text-slate-600">
          Master jenis izin hanya dapat diakses HR.
        </p>
      </section>
    );
  }

  const columns = [
    { key: "kode", header: "Kode", render: (row) => row.kode },
    { key: "nama", header: "Nama", render: (row) => row.nama },
    { key: "kategori", header: "Kategori", render: (row) => row.kategori === "cuti" ? "Cuti (bersaldo)" : "Izin (tanpa saldo)" },
    { key: "batas", header: "Batas izin", render: (row) => row.kategori === "izin" ? `${formatNumber(row.maksimal_hari)} hari` : "Mengikuti saldo" },
    { key: "kuota", header: "Kuota tahunan", render: (row) => `${formatNumber(row.kuota_tahunan)} hari` },
    {
      key: "dokumen",
      header: "Dokumen",
      render: (row) => (row.memerlukan_dokumen ? "Wajib" : "Opsional"),
    },
    { key: "ketentuan", header: "Ketentuan SOP", render: (row) => getLeavePolicy(row) },
    { key: "status", header: "Status", render: (row) => (row.is_active ? "Aktif" : "Nonaktif") },
    {
      key: "aksi",
      srHeader: "Aksi",
      cellClassName: "text-right",
      render: (row) => (
        <div className="flex justify-end gap-2">
          <Button
            variant="secondary"
            className="min-h-9 px-3 py-1"
            disabled={updateType.isPending}
            onClick={() => openEditor(row)}
          >
            Edit<span className="sr-only"> {row.nama}</span>
          </Button>
          <Button
            variant="secondary"
            className="min-h-9 px-3 py-1"
            disabled={updateType.isPending}
            onClick={() => toggleActive(row)}
          >
            {row.is_active ? "Nonaktifkan" : "Aktifkan"}
            <span className="sr-only"> {row.nama}</span>
          </Button>
        </div>
      ),
    },
  ];

  return (
    <section aria-labelledby="leave-types-title">
      <p className="text-sm font-semibold uppercase tracking-widest text-cyan-700">Master</p>
      <h1 id="leave-types-title" className="mt-2 text-3xl font-bold">Master Jenis Izin</h1>
      <p className="mt-2 max-w-2xl text-slate-600">
        Daftar bawaan mengikuti SOP Cuti, Kehadiran, Absensi, dan Izin bagian 1.4-1.9.
        HR dapat menambah jenis lain atau menonaktifkan jenis yang tidak lagi berlaku.
      </p>

      <div className="mt-5 rounded-xl border border-amber-300/40 bg-amber-50 p-4 text-sm text-amber-900">
        <p className="font-semibold">Ketidakhadiran tanpa keterangan (Alpha)</p>
        <p className="mt-1">
          Alpha bukan jenis izin yang dapat diajukan. Akumulasi maksimal 5 kali dalam 1 bulan
          dikenakan Surat Peringatan sesuai SOP.
        </p>
      </div>

      {formError && (
        <p role="alert" className="mt-5 rounded-lg border border-rose-300/30 bg-rose-300/10 p-3 text-rose-700">
          {formError}
        </p>
      )}

      <form
        onSubmit={handleSubmit(onSubmit)}
        noValidate
        className="mt-6 grid gap-5 rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-5 sm:grid-cols-2"
      >
        <FormInput id="leave-type-code" label="Kode" registration={register("kode")} error={errors.kode?.message} disabled={isSubmitting} />
        <FormInput id="leave-type-name" label="Nama" registration={register("nama")} error={errors.nama?.message} disabled={isSubmitting} />
        <label className="text-sm font-medium text-slate-700">Kategori<select {...register("kategori")} className="mt-2 min-h-10 w-full rounded-lg border bg-white px-3"><option value="cuti">Cuti (mengurangi saldo)</option><option value="izin">Izin (tanpa saldo)</option></select></label>
        <FormInput
          id="leave-type-quota"
          label="Kuota tahunan (hari)"
          type="number"
          min="0"
          registration={register("kuota_tahunan")}
          error={errors.kuota_tahunan?.message}
          disabled={isSubmitting || createCategory === "izin"}
        />
        <FormInput id="leave-type-maximum" label="Maksimal hari izin" type="number" min="1" max="365" registration={register("maksimal_hari")} error={errors.maksimal_hari?.message} disabled={isSubmitting || createCategory === "cuti"} />
        <label className="flex items-end gap-2 text-sm text-slate-700">
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
        {leaveTypes.isPending && <p role="status" className="text-slate-600">Memuat master…</p>}
        {leaveTypes.isError && (
          <div role="alert" className="rounded-xl border border-red-400/30 bg-red-400/10 p-4 text-red-700">
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

      {editingType && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/60 p-4">
          <form
            role="dialog"
            aria-modal="true"
            aria-labelledby="edit-leave-type-title"
            onSubmit={editForm.handleSubmit(submitEdit)}
            noValidate
            className="max-h-[90vh] w-full max-w-xl overflow-y-auto rounded-xl bg-white p-6 shadow-xl"
          >
            <h2 id="edit-leave-type-title" className="text-xl font-bold text-slate-900">
              Edit {editingType.nama}
            </h2>
            <p className="mt-2 text-sm text-slate-600">
              Kode <strong>{editingType.kode}</strong> dan kategori {editingType.kategori} dikunci
              agar pengajuan serta saldo lama tetap konsisten.
            </p>
            {editForm.formState.errors.root && (
              <p role="alert" className="mt-4 rounded-lg bg-rose-50 p-3 text-sm text-rose-700">
                {editForm.formState.errors.root.message}
              </p>
            )}
            <div className="mt-5 grid gap-4 sm:grid-cols-2">
              <input type="hidden" {...editForm.register("kategori")} />
              {editingType.kategori === "izin" && <input type="hidden" {...editForm.register("kuota_tahunan")} />}
              {editingType.kategori === "cuti" && <input type="hidden" {...editForm.register("maksimal_hari")} />}
              <FormInput
                id="edit-leave-type-name"
                label="Nama"
                registration={editForm.register("nama")}
                error={editForm.formState.errors.nama?.message}
                disabled={updateType.isPending}
              />
              {editingType.kategori === "cuti" ? (
                <FormInput
                  id="edit-leave-type-quota"
                  label="Kuota tahunan (hari)"
                  type="number"
                  min="0"
                  registration={editForm.register("kuota_tahunan")}
                  error={editForm.formState.errors.kuota_tahunan?.message}
                  disabled={updateType.isPending}
                />
              ) : (
                <FormInput
                  id="edit-leave-type-maximum"
                  label="Maksimal hari izin"
                  type="number"
                  min="1"
                  max="365"
                  registration={editForm.register("maksimal_hari")}
                  error={editForm.formState.errors.maksimal_hari?.message}
                  disabled={updateType.isPending}
                />
              )}
              <label className="flex items-center gap-2 text-sm text-slate-700 sm:col-span-2">
                <input type="checkbox" {...editForm.register("memerlukan_dokumen")} disabled={updateType.isPending} className="h-4 w-4" />
                Dokumen pendukung wajib
              </label>
            </div>
            <div className="mt-6 flex justify-end gap-3">
              <Button type="button" variant="secondary" disabled={updateType.isPending} onClick={() => setEditingType(null)}>
                Batal
              </Button>
              <Button type="submit" disabled={updateType.isPending}>
                {updateType.isPending ? "Menyimpanâ€¦" : "Simpan perubahan"}
              </Button>
            </div>
          </form>
        </div>
      )}
    </section>
  );
};
