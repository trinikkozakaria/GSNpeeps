import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";

import { FileField } from "../../../components/form/FileField";
import { FormInput } from "../../../components/form/FormInput";
import { Button } from "../../../components/ui/Button";
import { useCreateLeaveRequest, useLeaveTypes } from "../hooks/useLeave";
import {
  createLeaveFormSchema,
  isTravelLeaveType,
  validateSupportingDocument,
} from "../schemas/leave-schema";

const submitErrorMessage = (error) => {
  if (error?.code === "INSUFFICIENT_LEAVE_BALANCE") {
    return "Saldo atau kuota cuti tidak mencukupi untuk rentang tanggal ini.";
  }
  if (error?.status === 413) return "Ukuran dokumen melebihi batas 5 MB.";
  if (error?.status === 415) return "Format dokumen ditolak server. Gunakan PDF, JPG, atau PNG.";
  if (error?.status === 403) return "Anda tidak memiliki akses untuk mengajukan ketidakhadiran.";
  return error?.message ?? "Pengajuan belum dapat dikirim.";
};

export const LeaveRequestPage = () => {
  document.title = "Ajukan Ketidakhadiran — GSNpeeps";
  const create = useCreateLeaveRequest();
  // Master jenis izin dapat dibaca seluruh role terautentikasi (resolusi G-012); backend
  // membatasi non-HR ke jenis izin aktif. Kegagalan memuat tidak boleh memblokir halaman,
  // sehingga daftar kosong dan error ditangani sebagai state tersendiri.
  const leaveTypes = useLeaveTypes({ aktif: true });
  const [supportingDocument, setSupportingDocument] = useState(null);
  const [documentError, setDocumentError] = useState("");
  const [formError, setFormError] = useState("");
  const [successState, setSuccessState] = useState(null);

  const {
    register,
    handleSubmit,
    watch,
    reset,
    setError,
    formState: { errors, isSubmitting },
  } = useForm({
    resolver: zodResolver(createLeaveFormSchema),
    defaultValues: {
      jenis_izin_id: "",
      tanggal_mulai: "",
      tanggal_selesai: "",
      alasan: "",
      lokasi_tujuan: "",
      keterangan_lokasi: "",
    },
  });

  const selectedTypeID = watch("jenis_izin_id");
  const selectedType = (leaveTypes.data ?? []).find((item) => item.id === selectedTypeID);
  const isTravel = isTravelLeaveType(selectedType?.nama);
  // Kewajiban dokumen berasal dari master jenis izin (D-024).
  const documentRequired = selectedType?.memerlukan_dokumen ?? false;

  const onSubmit = async (values) => {
    setFormError("");
    setSuccessState(null);

    const documentProblem = validateSupportingDocument(supportingDocument, documentRequired);
    setDocumentError(documentProblem);
    if (documentProblem) return;

    if (isTravel && !values.lokasi_tujuan?.trim()) {
      setError("lokasi_tujuan", {
        type: "manual",
        message: "Lokasi tujuan wajib diisi untuk Perjalanan Dinas",
      });
      return;
    }
    if (isTravel && !values.keterangan_lokasi?.trim()) {
      setError("keterangan_lokasi", {
        type: "manual",
        message: "Keperluan tugas wajib diisi untuk Perjalanan Dinas",
      });
      return;
    }

    try {
      const created = await create.mutateAsync({
        ...values,
        lokasi_tujuan: isTravel ? values.lokasi_tujuan : undefined,
        keterangan_lokasi: isTravel ? values.keterangan_lokasi : undefined,
        dokumen_pendukung: supportingDocument,
      });
      // Status berasal dari server; tidak ada tebakan status disetujui di client.
      setSuccessState(created);
      setSupportingDocument(null);
      reset();
    } catch (error) {
      Object.entries(error?.fields ?? {}).forEach(([field, message]) => {
        setError(field, { type: "server", message });
      });
      setFormError(submitErrorMessage(error));
    }
  };

  return (
    <section aria-labelledby="leave-request-title" className="max-w-3xl">
      <p className="text-sm font-semibold uppercase tracking-widest text-cyan-300">Absensi</p>
      <h1 id="leave-request-title" className="mt-2 text-3xl font-bold">
        Ajukan Ketidakhadiran
      </h1>
      <p className="mt-2 text-slate-300">
        Pengajuan diteruskan ke approver sesuai struktur organisasi Anda. Status akhir
        ditentukan server.
      </p>

      {successState && (
        <p
          role="status"
          className="mt-5 rounded-lg border border-emerald-300/30 bg-emerald-300/10 p-4 text-sm text-emerald-100"
        >
          Pengajuan terkirim dan sedang menunggu keputusan approver.
        </p>
      )}

      {leaveTypes.isError && (
        <p role="alert" className="mt-5 rounded-lg border border-rose-300/30 bg-rose-300/10 p-4 text-sm text-rose-100">
          Daftar jenis izin gagal dimuat. Muat ulang halaman, dan hubungi HR bila masalah
          berlanjut.
        </p>
      )}

      {/* Master jenis izin diisi HR dan dapat kosong (G-011); jelaskan agar pemohon tidak
          mengira formulir rusak. */}
      {!leaveTypes.isPending && !leaveTypes.isError && (leaveTypes.data ?? []).length === 0 && (
        <p role="status" className="mt-5 rounded-lg border border-amber-300/30 bg-amber-300/10 p-4 text-sm text-amber-100">
          Belum ada jenis izin aktif yang dapat diajukan. Hubungi HR untuk mengaktifkan
          master jenis izin.
        </p>
      )}

      <form
        onSubmit={handleSubmit(onSubmit)}
        noValidate
        className="mt-7 grid gap-5 rounded-xl border border-white/10 bg-white/[0.03] p-6"
      >
        {formError && (
          <p role="alert" className="rounded-lg border border-rose-300/30 bg-rose-300/10 p-3 text-rose-200">
            {formError}
          </p>
        )}

        <div>
          <label htmlFor="leave-type" className="mb-2 block text-sm font-medium text-slate-200">
            Jenis izin
          </label>
          <select
            id="leave-type"
            {...register("jenis_izin_id")}
            disabled={isSubmitting}
            aria-invalid={Boolean(errors.jenis_izin_id)}
            className="min-h-11 w-full rounded-lg border border-white/15 bg-slate-950 px-3 text-white outline-none focus:border-cyan-300"
          >
            <option value="">Pilih jenis izin</option>
            {(leaveTypes.data ?? []).map((type) => (
              <option key={type.id} value={type.id}>
                {type.nama}
              </option>
            ))}
          </select>
          {errors.jenis_izin_id && (
            <p role="alert" className="mt-2 text-sm text-rose-300">
              {errors.jenis_izin_id.message}
            </p>
          )}
        </div>

        <div className="grid gap-5 sm:grid-cols-2">
          <FormInput
            id="leave-start"
            label="Tanggal mulai"
            type="date"
            registration={register("tanggal_mulai")}
            error={errors.tanggal_mulai?.message}
            disabled={isSubmitting}
          />
          <FormInput
            id="leave-end"
            label="Tanggal selesai"
            type="date"
            registration={register("tanggal_selesai")}
            error={errors.tanggal_selesai?.message}
            disabled={isSubmitting}
          />
        </div>

        <div>
          <label htmlFor="leave-reason" className="mb-2 block text-sm font-medium text-slate-200">
            Alasan
          </label>
          <textarea
            id="leave-reason"
            rows={4}
            {...register("alasan")}
            disabled={isSubmitting}
            aria-invalid={Boolean(errors.alasan)}
            className="w-full rounded-lg border border-white/15 bg-slate-950 p-3 text-white outline-none focus:border-cyan-300"
          />
          {errors.alasan && (
            <p role="alert" className="mt-2 text-sm text-rose-300">
              {errors.alasan.message}
            </p>
          )}
        </div>

        {isTravel && (
          <div className="grid gap-5 sm:grid-cols-2">
            <FormInput
              id="leave-destination"
              label="Lokasi tujuan"
              registration={register("lokasi_tujuan")}
              error={errors.lokasi_tujuan?.message}
              disabled={isSubmitting}
            />
            <FormInput
              id="leave-destination-note"
              label="Keperluan tugas"
              registration={register("keterangan_lokasi")}
              error={errors.keterangan_lokasi?.message}
              disabled={isSubmitting}
            />
          </div>
        )}

        <FileField
          id="leave-document"
          label={`Dokumen pendukung ${documentRequired ? "(wajib)" : "(opsional)"}`}
          accept=".pdf,.jpg,.jpeg,.png"
          description="PDF, JPG, atau PNG maksimal 5 MB."
          file={supportingDocument}
          error={documentError}
          disabled={isSubmitting}
          onFileChange={(selected) => {
            setSupportingDocument(selected);
            setDocumentError(selected ? validateSupportingDocument(selected, documentRequired) : "");
          }}
        />

        <div>
          <Button type="submit" disabled={isSubmitting}>
            {isSubmitting ? "Mengirim…" : "Kirim pengajuan"}
          </Button>
        </div>
      </form>
    </section>
  );
};
