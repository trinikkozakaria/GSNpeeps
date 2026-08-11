import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { Link, useNavigate, useParams } from "react-router-dom";

import { FormInput } from "../../../components/form/FormInput";
import { Button } from "../../../components/ui/Button";
import { useAuth } from "../../auth/hooks/useAuth";
import {
  BpjsNpwpFields,
  CurrentSalaryFields,
  EducationFields,
  EmergencyContactsFields,
  PositionHistoryFields,
} from "../components/EmployeeDetailFormFields";
import { EmployeeSelectField } from "../components/EmployeeSelectField";
import {
  useDepartments,
  useEmployeeDetail,
  usePositions,
  useUpdateEmployee,
} from "../hooks/useEmployees";
import {
  buildEmployeeDetailPayload,
  emptyEmployeeDetailDefaults,
  mapEmployeeDetailToFormDefaults,
  updateEmployeeSchema,
} from "../schemas/employee-schema";

export const EmployeeEditPage = () => {
  document.title = "Edit karyawan — GSNpeeps";
  const { id } = useParams();
  const auth = useAuth();
  const navigate = useNavigate();
  const detail = useEmployeeDetail(auth.role, id);
  const departments = useDepartments();
  const mutation = useUpdateEmployee(auth.role, id);
  const [formError, setFormError] = useState("");
  const {
    register,
    handleSubmit,
    reset,
    watch,
    setError,
    control,
    formState: { errors, isSubmitting },
  } = useForm({
    resolver: zodResolver(updateEmployeeSchema),
    defaultValues: {
      nama: "",
      email: "",
      jenis_kelamin: "",
      tanggal_lahir: "",
      tanggal_join: "",
      department_id: "",
      position_id: "",
      status_pernikahan: "",
      status: "aktif",
      ...emptyEmployeeDetailDefaults,
    },
  });
  const departmentId = watch("department_id");
  const positions = usePositions(departmentId);

  useEffect(() => {
    if (detail.data) {
      reset({
        nama: detail.data.nama,
        email: detail.data.email,
        jenis_kelamin: detail.data.jenis_kelamin,
        tanggal_lahir: detail.data.tanggal_lahir,
        tanggal_join: detail.data.tanggal_join,
        department_id: detail.data.department_id ?? "",
        position_id: detail.data.position_id ?? "",
        status_pernikahan: detail.data.status_pernikahan ?? "",
        status: detail.data.status,
        ...mapEmployeeDetailToFormDefaults(detail.data),
      });
    }
  }, [detail.data, reset]);

  const onSubmit = async (values) => {
    setFormError("");
    const { bpjs, npwp, kontak_darurat, pendidikan, riwayat_jabatan, gaji_berjalan, ...rest } = values;
    const payload = {
      ...rest,
      status_pernikahan: values.status_pernikahan || undefined,
      ...buildEmployeeDetailPayload(values),
    };
    try {
      await mutation.mutateAsync(payload);
      navigate(`/app/karyawan/${id}`, {
        replace: true,
        state: { message: "Data karyawan berhasil diperbarui." },
      });
    } catch (error) {
      Object.entries(error.fields ?? {}).forEach(([field, message]) => {
        if (field in values) setError(field, { type: "server", message });
      });
      setFormError(
        error.status === 409
          ? "Perubahan bertentangan dengan data organisasi atau data unik yang sudah ada."
          : "Data belum dapat disimpan. Periksa kembali isian Anda.",
      );
    }
  };

  if (detail.isPending) return <p role="status">Memuat data karyawan…</p>;
  if (detail.isError) return <p role="alert">Data karyawan tidak dapat dimuat.</p>;

  return (
    <section aria-labelledby="employee-edit-title" className="max-w-3xl">
      <Link to={`/app/karyawan/${id}`} className="text-sm font-semibold text-cyan-700">← Batal dan kembali</Link>
      <h1 id="employee-edit-title" className="mt-5 text-3xl font-bold">Edit karyawan</h1>
      <p className="mt-2 text-slate-500">Perubahan akun dan status akan mencabut sesi aktif karyawan.</p>
      <form onSubmit={handleSubmit(onSubmit)} noValidate className="mt-7 grid gap-5 rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-6 sm:grid-cols-2">
        {formError && <div role="alert" className="sm:col-span-2 rounded-lg border border-rose-300/30 bg-rose-300/10 p-3 text-rose-700">{formError}</div>}
        <FormInput id="employee-name" label="Nama lengkap" registration={register("nama")} error={errors.nama?.message} disabled={isSubmitting} />
        <FormInput id="employee-email" label="Email login" type="email" registration={register("email")} error={errors.email?.message} disabled={isSubmitting} />
        <EmployeeSelectField id="employee-gender" label="Jenis kelamin" registration={register("jenis_kelamin")} error={errors.jenis_kelamin?.message} disabled={isSubmitting}>
          <option value="">Pilih jenis kelamin</option>
          <option value="L">Laki-laki</option>
          <option value="P">Perempuan</option>
        </EmployeeSelectField>
        <EmployeeSelectField id="employee-marital" label="Status pernikahan" registration={register("status_pernikahan")} error={errors.status_pernikahan?.message} disabled={isSubmitting}>
          <option value="">Belum diisi</option>
          <option value="lajang">Lajang</option>
          <option value="menikah">Menikah</option>
          <option value="cerai">Cerai</option>
        </EmployeeSelectField>
        <FormInput id="employee-birth-date" label="Tanggal lahir" type="date" registration={register("tanggal_lahir")} error={errors.tanggal_lahir?.message} disabled={isSubmitting} />
        <FormInput id="employee-join-date" label="Tanggal bergabung" type="date" registration={register("tanggal_join")} error={errors.tanggal_join?.message} disabled={isSubmitting} />
        <EmployeeSelectField id="employee-department" label="Departemen" registration={register("department_id")} error={errors.department_id?.message} disabled={isSubmitting}>
          <option value="">Pilih departemen</option>
          {(departments.data ?? []).map((item) => <option key={item.id} value={item.id}>{item.nama}</option>)}
        </EmployeeSelectField>
        <EmployeeSelectField id="employee-position" label="Jabatan" registration={register("position_id")} error={errors.position_id?.message} disabled={isSubmitting || !departmentId}>
          <option value="">Pilih jabatan</option>
          {(positions.data ?? []).map((item) => <option key={item.id} value={item.id}>{item.nama}</option>)}
        </EmployeeSelectField>
        <EmployeeSelectField id="employee-status" label="Status karyawan" registration={register("status")} error={errors.status?.message} disabled={isSubmitting}>
          <option value="aktif">Aktif</option>
          <option value="nonaktif">Nonaktif</option>
        </EmployeeSelectField>
        <div className="sm:col-span-2">
          <BpjsNpwpFields register={register} errors={errors} disabled={isSubmitting} idPrefix="edit" />
        </div>
        <div className="sm:col-span-2">
          <EmergencyContactsFields control={control} register={register} errors={errors} disabled={isSubmitting} idPrefix="edit" />
        </div>
        <div className="sm:col-span-2">
          <EducationFields control={control} register={register} errors={errors} disabled={isSubmitting} idPrefix="edit" />
        </div>
        <div className="sm:col-span-2">
          <PositionHistoryFields
            control={control}
            register={register}
            errors={errors}
            disabled={isSubmitting}
            idPrefix="edit"
            departments={departments.data}
          />
        </div>
        <div className="sm:col-span-2">
          <CurrentSalaryFields register={register} errors={errors} disabled={isSubmitting} idPrefix="edit" />
        </div>
        <div className="flex items-end gap-3 sm:col-span-2">
          <Button type="submit" disabled={isSubmitting}>{isSubmitting ? "Menyimpan…" : "Simpan perubahan"}</Button>
          <Link to={`/app/karyawan/${id}`} className="inline-flex min-h-11 items-center px-3 font-semibold text-slate-600">Batal</Link>
        </div>
      </form>
    </section>
  );
};
