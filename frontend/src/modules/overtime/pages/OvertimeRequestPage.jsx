import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";

import { FileField } from "../../../components/form/FileField";
import { FormInput } from "../../../components/form/FormInput";
import { Button } from "../../../components/ui/Button";
import { validateSupportingDocument } from "../../leave/schemas/leave-schema";
import { useCreateOvertimeRequest } from "../hooks/useOvertime";
import { createOvertimeFormSchema } from "../schemas/overtime-schema";

const submitErrorMessage = (error) => {
  if (error?.status === 413) return "Ukuran dokumen melebihi batas 5 MB.";
  if (error?.status === 415) return "Format dokumen ditolak server. Gunakan PDF, JPG, atau PNG.";
  if (error?.status === 403) return "Anda tidak memiliki akses untuk mengajukan lembur.";
  return error?.message ?? "Pengajuan lembur belum dapat dikirim.";
};

export const OvertimeRequestPage = () => {
  document.title = "Ajukan Lembur — GSNpeeps";
  const create = useCreateOvertimeRequest();
  const [supportingDocument, setSupportingDocument] = useState(null);
  const [documentError, setDocumentError] = useState("");
  const [formError, setFormError] = useState("");
  const [submitted, setSubmitted] = useState(false);

  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors, isSubmitting },
  } = useForm({
    resolver: zodResolver(createOvertimeFormSchema),
    defaultValues: { tanggal: "", waktu_mulai: "", waktu_selesai: "", alasan: "" },
  });

  const onSubmit = async (values) => {
    setFormError("");
    setSubmitted(false);

    // Dokumen lembur opsional; validasi hanya berlaku bila berkas dipilih.
    const documentProblem = validateSupportingDocument(supportingDocument, false);
    setDocumentError(documentProblem);
    if (documentProblem) return;

    try {
      await create.mutateAsync({ ...values, dokumen_pendukung: document });
      setSubmitted(true);
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
    <section aria-labelledby="overtime-request-title" className="max-w-3xl">
      <p className="text-sm font-semibold uppercase tracking-widest text-cyan-700">Absensi</p>
      <h1 id="overtime-request-title" className="mt-2 text-3xl font-bold">
        Ajukan Lembur
      </h1>
      <p className="mt-2 text-slate-600">
        Durasi lembur dihitung server dari jam mulai dan selesai yang Anda kirim.
      </p>

      {submitted && (
        <p
          role="status"
          className="mt-5 rounded-lg border border-emerald-300/30 bg-emerald-300/10 p-4 text-sm text-emerald-700"
        >
          Pengajuan lembur terkirim dan sedang menunggu keputusan approver.
        </p>
      )}

      <form
        onSubmit={handleSubmit(onSubmit)}
        noValidate
        className="mt-7 grid gap-5 rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-6"
      >
        {formError && (
          <p role="alert" className="rounded-lg border border-rose-300/30 bg-rose-300/10 p-3 text-rose-700">
            {formError}
          </p>
        )}

        <FormInput
          id="overtime-date"
          label="Tanggal"
          type="date"
          registration={register("tanggal")}
          error={errors.tanggal?.message}
          disabled={isSubmitting}
        />
        <div className="grid gap-5 sm:grid-cols-2">
          <FormInput
            id="overtime-start"
            label="Jam mulai"
            type="time"
            registration={register("waktu_mulai")}
            error={errors.waktu_mulai?.message}
            disabled={isSubmitting}
          />
          <FormInput
            id="overtime-end"
            label="Jam selesai"
            type="time"
            registration={register("waktu_selesai")}
            error={errors.waktu_selesai?.message}
            disabled={isSubmitting}
          />
        </div>

        <div>
          <label htmlFor="overtime-reason" className="mb-2 block text-sm font-medium text-slate-700">
            Alasan
          </label>
          <textarea
            id="overtime-reason"
            rows={4}
            {...register("alasan")}
            disabled={isSubmitting}
            aria-invalid={Boolean(errors.alasan)}
            className="w-full rounded-lg border border-slate-900/15 bg-white p-3 text-slate-900 outline-none focus:border-cyan-300"
          />
          {errors.alasan && (
            <p role="alert" className="mt-2 text-sm text-rose-700">
              {errors.alasan.message}
            </p>
          )}
        </div>

        <FileField
          id="overtime-document"
          label="Dokumen pendukung (opsional)"
          accept=".pdf,.jpg,.jpeg,.png"
          description="PDF, JPG, atau PNG maksimal 5 MB."
          file={supportingDocument}
          error={documentError}
          disabled={isSubmitting}
          onFileChange={(selected) => {
            setSupportingDocument(selected);
            setDocumentError(selected ? validateSupportingDocument(selected, false) : "");
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
