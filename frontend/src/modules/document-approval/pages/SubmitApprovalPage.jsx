import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";

import { FormInput } from "../../../components/form/FormInput";
import { Button } from "../../../components/ui/Button";
import { useCreateApprovalRequest, useWorkflowTemplates } from "../hooks/useDocumentApproval";
import { submitApprovalFormSchema } from "../schemas/document-approval-schema";

export const SubmitApprovalPage = () => {
  document.title = "Ajukan Persetujuan Dokumen — GSNpeeps";
  const templates = useWorkflowTemplates();
  const mutation = useCreateApprovalRequest();
  const [formError, setFormError] = useState("");
  const [result, setResult] = useState(null);

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm({
    resolver: zodResolver(submitApprovalFormSchema),
    defaultValues: { document_url: "", judul: "", template_id: "" },
  });

  const activeTemplates = (templates.data ?? []).filter((template) => template.aktif);

  const onSubmit = async (values) => {
    setFormError("");
    setResult(null);
    try {
      const response = await mutation.mutateAsync(values);
      setResult(response);
      reset();
    } catch (error) {
      setFormError(error.message ?? "Pengajuan belum dapat dikirim.");
    }
  };

  return (
    <section aria-labelledby="submit-approval-title" className="max-w-2xl">
      <p className="text-sm font-semibold uppercase tracking-widest text-cyan-700">
        Persetujuan Dokumen
      </p>
      <h1 id="submit-approval-title" className="mt-2 text-3xl font-bold">
        Ajukan Persetujuan Dokumen
      </h1>
      <p className="mt-2 text-slate-500">
        Masukkan tautan dokumen (mis. Nextcloud) untuk memulai proses persetujuan.
      </p>

      <form onSubmit={handleSubmit(onSubmit)} noValidate className="mt-7 space-y-5">
        {formError && (
          <div role="alert" className="rounded-lg border border-rose-300/30 bg-rose-300/10 p-3 text-rose-700">
            {formError}
          </div>
        )}
        {result && (
          <div role="status" className="rounded-lg border border-emerald-300/30 bg-emerald-300/10 p-3 text-emerald-800">
            {result.status === "berjalan" && result.stage_index === 0
              ? "Pengajuan berhasil dikirim dan sedang menunggu persetujuan tahap pertama."
              : "Dokumen ini sudah memiliki pengajuan yang sedang berjalan."}
            {" "}(ID: {result.id})
          </div>
        )}

        <FormInput
          id="submit-document-url"
          label="URL Dokumen"
          type="url"
          placeholder="https://..."
          registration={register("document_url")}
          error={errors.document_url?.message}
          disabled={isSubmitting}
        />
        <FormInput
          id="submit-judul"
          label="Judul dokumen"
          registration={register("judul")}
          error={errors.judul?.message}
          disabled={isSubmitting}
        />
        <div>
          <label htmlFor="submit-template" className="block text-sm font-medium">
            Alur persetujuan
          </label>
          <select
            id="submit-template"
            {...register("template_id")}
            disabled={isSubmitting || templates.isPending}
            className="mt-1 w-full rounded-lg border border-slate-300 p-2"
          >
            <option value="">Pilih alur persetujuan</option>
            {activeTemplates.map((template) => (
              <option key={template.id} value={template.id}>
                {template.nama}
              </option>
            ))}
          </select>
          {errors.template_id?.message && (
            <p role="alert" className="mt-1 text-sm text-rose-700">{errors.template_id.message}</p>
          )}
          {!templates.isPending && activeTemplates.length === 0 && (
            <p role="status" className="mt-1 text-sm text-amber-700">
              Belum ada alur persetujuan aktif. Hubungi HR untuk membuat alur terlebih dahulu.
            </p>
          )}
        </div>

        <Button type="submit" disabled={isSubmitting || activeTemplates.length === 0}>
          {isSubmitting ? "Mengirim…" : "Kirim Pengajuan"}
        </Button>
      </form>
    </section>
  );
};
