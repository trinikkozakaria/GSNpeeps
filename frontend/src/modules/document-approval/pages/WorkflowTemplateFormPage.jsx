import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect, useState } from "react";
import { useFieldArray, useForm } from "react-hook-form";
import { Link, useNavigate, useParams } from "react-router-dom";

import { apiClient } from "../../../lib/api/client";
import { FormInput } from "../../../components/form/FormInput";
import { Button } from "../../../components/ui/Button";
import {
  useCreateWorkflowTemplate,
  useUpdateWorkflowTemplate,
  useWorkflowTemplateDetail,
} from "../hooks/useDocumentApproval";
import { createWorkflowTemplateFormSchema } from "../schemas/document-approval-schema";

// null = still loading. The edit form must not fill its dropdowns before the options exist,
// otherwise the browser silently falls back to the empty option.
const usePositionOptions = () => {
  const [positions, setPositions] = useState(null);
  useEffect(() => {
    apiClient.get("/master/jabatan").then((envelope) => setPositions(envelope.data ?? []));
  }, []);
  return positions;
};

export const WorkflowTemplateFormPage = () => {
  const { id } = useParams();
  const isEdit = Boolean(id);
  document.title = `${isEdit ? "Ubah" : "Tambah"} Alur Persetujuan — GSNpeeps`;
  const navigate = useNavigate();
  const positionOptions = usePositionOptions();
  const positions = positionOptions ?? [];
  const templateDetail = useWorkflowTemplateDetail(id);
  const createMutation = useCreateWorkflowTemplate();
  const updateMutation = useUpdateWorkflowTemplate();
  const [formError, setFormError] = useState("");

  const {
    register,
    control,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm({
    resolver: zodResolver(createWorkflowTemplateFormSchema),
    defaultValues: { nama: "", aktif: true, tahap: [{ nama_tahap: "", position_id: "" }] },
  });
  const { fields, append, remove } = useFieldArray({ control, name: "tahap" });

  useEffect(() => {
    if (isEdit && templateDetail.data && positionOptions !== null) {
      reset({
        nama: templateDetail.data.nama,
        aktif: templateDetail.data.aktif,
        tahap: templateDetail.data.tahap.map((stage) => ({
          nama_tahap: stage.nama_tahap,
          position_id: stage.position_id,
        })),
      });
    }
  }, [isEdit, templateDetail.data, positionOptions, reset]);

  const onSubmit = async (values) => {
    setFormError("");
    const payload = {
      nama: values.nama,
      aktif: values.aktif,
      tahap: values.tahap.map((stage, index) => ({
        urutan: index,
        nama_tahap: stage.nama_tahap,
        position_id: stage.position_id,
      })),
    };
    try {
      if (isEdit) {
        await updateMutation.mutateAsync({ id, payload });
      } else {
        await createMutation.mutateAsync(payload);
      }
      navigate("/app/master/alur-persetujuan-dokumen", { replace: true });
    } catch (error) {
      setFormError(error.message ?? "Alur belum dapat disimpan.");
    }
  };

  return (
    <section aria-labelledby="workflow-form-title" className="max-w-3xl">
      <Link to="/app/master/alur-persetujuan-dokumen" className="text-sm font-semibold text-cyan-700">
        ← Kembali ke daftar
      </Link>
      <h1 id="workflow-form-title" className="mt-5 text-3xl font-bold">
        {isEdit ? "Ubah alur persetujuan" : "Tambah alur persetujuan"}
      </h1>

      {isEdit && templateDetail.isPending && (
        <p role="status" className="mt-6 text-slate-600">Memuat data alur…</p>
      )}
      {isEdit && templateDetail.isError && (
        <div role="alert" className="mt-6 rounded-xl border border-red-400/30 bg-red-400/10 p-4 text-red-700">
          Alur belum dapat dimuat. {templateDetail.error.message}
        </div>
      )}

      <form onSubmit={handleSubmit(onSubmit)} noValidate className="mt-7 space-y-6">
        {formError && (
          <div role="alert" className="rounded-lg border border-rose-300/30 bg-rose-300/10 p-3 text-rose-700">
            {formError}
          </div>
        )}

        <fieldset className="grid gap-5 rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-6">
          <legend className="px-2 text-lg font-bold">Detail alur</legend>
          <FormInput
            id="template-nama"
            label="Nama alur"
            registration={register("nama")}
            error={errors.nama?.message}
            disabled={isSubmitting}
          />
          <label className="flex items-center gap-2 text-sm font-medium">
            <input type="checkbox" {...register("aktif")} disabled={isSubmitting} />
            Aktif
          </label>
        </fieldset>

        <fieldset className="rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-6">
          <legend className="px-2 text-lg font-bold">Tahap persetujuan</legend>
          {errors.tahap?.message && (
            <p role="alert" className="mb-3 text-rose-700">{errors.tahap.message}</p>
          )}
          <div className="space-y-4">
            {fields.map((field, index) => (
              <div key={field.id} className="flex flex-wrap items-start gap-3 rounded-lg border border-slate-900/10 p-4">
                <span className="mt-2 text-sm font-semibold text-slate-500">{index + 1}.</span>
                <div className="min-w-[200px] flex-1">
                  <FormInput
                    id={`tahap-nama-${index}`}
                    label="Nama tahap"
                    registration={register(`tahap.${index}.nama_tahap`)}
                    error={errors.tahap?.[index]?.nama_tahap?.message}
                    disabled={isSubmitting}
                  />
                </div>
                <div className="min-w-[200px] flex-1">
                  <label htmlFor={`tahap-position-${index}`} className="block text-sm font-medium">
                    Posisi
                  </label>
                  <select
                    id={`tahap-position-${index}`}
                    {...register(`tahap.${index}.position_id`)}
                    disabled={isSubmitting}
                    className="mt-1 w-full rounded-lg border border-slate-300 p-2"
                  >
                    <option value="">Pilih posisi</option>
                    {positions.map((position) => (
                      <option key={position.id} value={position.id}>
                        {position.nama}
                      </option>
                    ))}
                  </select>
                  {errors.tahap?.[index]?.position_id?.message && (
                    <p role="alert" className="mt-1 text-sm text-rose-700">
                      {errors.tahap[index].position_id.message}
                    </p>
                  )}
                </div>
                <Button
                  type="button"
                  variant="secondary"
                  className="mt-6"
                  disabled={isSubmitting || fields.length <= 1}
                  onClick={() => remove(index)}
                >
                  Hapus
                </Button>
              </div>
            ))}
          </div>
          <Button
            type="button"
            variant="secondary"
            className="mt-4"
            disabled={isSubmitting}
            onClick={() => append({ nama_tahap: "", position_id: "" })}
          >
            Tambah tahap
          </Button>
        </fieldset>

        <div className="flex flex-wrap gap-3">
          <Button type="submit" disabled={isSubmitting}>
            {isSubmitting ? "Menyimpan…" : "Simpan alur"}
          </Button>
          <Link to="/app/master/alur-persetujuan-dokumen" className="inline-flex min-h-10 items-center px-3 font-semibold text-slate-600">
            Batal
          </Link>
        </div>
      </form>
    </section>
  );
};
