import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { Link, useNavigate } from "react-router-dom";

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
  useCreateEmployee,
  useDepartments,
  useEmployees,
  usePositions,
} from "../hooks/useEmployees";
import {
  buildEmployeeDetailPayload,
  createEmployeeSchema,
  emptyEmployeeDetailDefaults,
} from "../schemas/employee-schema";

const defaults = {
  nip: "",
  nama: "",
  email: "",
  jenis_kelamin: "",
  tanggal_lahir: "",
  tanggal_join: "",
  department_id: "",
  position_id: "",
  atasan_id: "",
  status_pernikahan: "",
  role: "karyawan",
  alamat: {
    jalan: "",
    kelurahan: "",
    kecamatan: "",
    kota: "",
    provinsi: "",
  },
  ktp: { nomor_ktp: "" },
  kontrak: {
    nomor_kontrak: "",
    jenis_kontrak: "PKWT",
    tanggal_mulai: "",
    tanggal_berakhir: "",
  },
  ...emptyEmployeeDetailDefaults,
};

export const EmployeeCreatePage = () => {
  document.title = "Tambah karyawan — GSNpeeps";
  const auth = useAuth();
  const navigate = useNavigate();
  const mutation = useCreateEmployee();
  const departments = useDepartments();
  const supervisors = useEmployees(auth.role, {
    status: "aktif",
    page: 1,
    limit: 100,
  });
  const [formError, setFormError] = useState("");
  const {
    register,
    handleSubmit,
    watch,
    setValue,
    setError,
    control,
    formState: { errors, isSubmitting },
  } = useForm({
    resolver: zodResolver(createEmployeeSchema),
    defaultValues: defaults,
  });
  const departmentId = watch("department_id");
  const positions = usePositions(departmentId);
  const departmentRegistration = register("department_id");

  const onSubmit = async (values) => {
    setFormError("");
    const { bpjs, npwp, kontak_darurat, pendidikan, riwayat_jabatan, gaji_berjalan, ...rest } = values;
    const payload = {
      ...rest,
      atasan_id: values.atasan_id || null,
      status_pernikahan: values.status_pernikahan || undefined,
      alamat: {
        ...values.alamat,
        kelurahan: values.alamat.kelurahan || null,
        kecamatan: values.alamat.kecamatan || null,
      },
      ...buildEmployeeDetailPayload(values),
    };
    try {
      const result = await mutation.mutateAsync(payload);
      navigate(`/app/karyawan/${result.id}`, {
        replace: true,
        state: { message: "Karyawan berhasil ditambahkan." },
      });
    } catch (error) {
      Object.entries(error.fields ?? {}).forEach(([field, message]) => {
        setError(field, { type: "server", message });
      });
      setFormError(
        error.status === 409
          ? "NIP, email, nomor KTP, nomor kontrak, atau struktur organisasi sudah digunakan/tidak sesuai."
          : "Karyawan belum dapat ditambahkan. Periksa kembali seluruh bagian form.",
      );
    }
  };

  return (
    <section aria-labelledby="employee-create-title" className="max-w-6xl">
      <Link to="/app/karyawan" className="text-sm font-semibold text-cyan-700">← Kembali ke daftar</Link>
      <h1 id="employee-create-title" className="mt-5 text-3xl font-bold">Tambah karyawan</h1>
      <p className="mt-2 text-slate-500">
        HR mengisi data awal sesuai dokumen resmi. Akun login dibuat terkunci dan tidak memiliki password yang diketahui HR.
      </p>

      <form onSubmit={handleSubmit(onSubmit)} noValidate className="mt-7 space-y-6">
        {formError && (
          <div role="alert" className="rounded-lg border border-rose-300/30 bg-rose-300/10 p-3 text-rose-700">
            {formError}
          </div>
        )}

        <fieldset className="grid gap-5 rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-6 sm:grid-cols-2 lg:grid-cols-3">
          <legend className="px-2 text-lg font-bold">Identitas dan pekerjaan</legend>
          <FormInput id="create-nip" label="NIP" registration={register("nip")} error={errors.nip?.message} disabled={isSubmitting} />
          <FormInput id="create-name" label="Nama lengkap" registration={register("nama")} error={errors.nama?.message} disabled={isSubmitting} />
          <FormInput id="create-email" label="Email login" type="email" registration={register("email")} error={errors.email?.message} disabled={isSubmitting} />
          <EmployeeSelectField id="create-gender" label="Jenis kelamin" registration={register("jenis_kelamin")} error={errors.jenis_kelamin?.message} disabled={isSubmitting}>
            <option value="">Pilih jenis kelamin</option>
            <option value="L">Laki-laki</option>
            <option value="P">Perempuan</option>
          </EmployeeSelectField>
          <FormInput id="create-birth-date" label="Tanggal lahir" type="date" registration={register("tanggal_lahir")} error={errors.tanggal_lahir?.message} disabled={isSubmitting} />
          <FormInput id="create-join-date" label="Tanggal bergabung" type="date" registration={register("tanggal_join")} error={errors.tanggal_join?.message} disabled={isSubmitting} />
          <EmployeeSelectField
            id="create-department"
            label="Departemen"
            registration={{
              ...departmentRegistration,
              onChange: (event) => {
                void departmentRegistration.onChange(event);
                setValue("position_id", "", { shouldValidate: true });
              },
            }}
            error={errors.department_id?.message}
            disabled={isSubmitting}
          >
            <option value="">Pilih departemen</option>
            {(departments.data ?? []).map((item) => <option key={item.id} value={item.id}>{item.nama}</option>)}
          </EmployeeSelectField>
          <EmployeeSelectField id="create-position" label="Jabatan" registration={register("position_id")} error={errors.position_id?.message} disabled={isSubmitting || !departmentId}>
            <option value="">Pilih jabatan</option>
            {(positions.data ?? []).map((item) => <option key={item.id} value={item.id}>{item.nama}</option>)}
          </EmployeeSelectField>
          <EmployeeSelectField id="create-supervisor" label="Atasan langsung" registration={register("atasan_id")} error={errors.atasan_id?.message} disabled={isSubmitting}>
            <option value="">Tanpa atasan langsung</option>
            {(supervisors.data?.items ?? []).map((item) => <option key={item.id} value={item.id}>{item.nama} — {item.jabatan}</option>)}
          </EmployeeSelectField>
          <EmployeeSelectField id="create-role" label="Role sistem" registration={register("role")} error={errors.role?.message} disabled={isSubmitting}>
            <option value="karyawan">Karyawan</option>
            <option value="atasan">Atasan</option>
            <option value="hr">HR</option>
            <option value="top_management">Top Management</option>
          </EmployeeSelectField>
          <EmployeeSelectField id="create-marital" label="Status pernikahan" registration={register("status_pernikahan")} error={errors.status_pernikahan?.message} disabled={isSubmitting}>
            <option value="">Belum diisi</option>
            <option value="lajang">Lajang</option>
            <option value="menikah">Menikah</option>
            <option value="cerai">Cerai</option>
          </EmployeeSelectField>
        </fieldset>

        <fieldset className="grid gap-5 rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-6 sm:grid-cols-2 lg:grid-cols-3">
          <legend className="px-2 text-lg font-bold">Alamat</legend>
          <div className="sm:col-span-2 lg:col-span-3">
            <FormInput id="create-street" label="Jalan" registration={register("alamat.jalan")} error={errors.alamat?.jalan?.message} disabled={isSubmitting} />
          </div>
          <FormInput id="create-village" label="Kelurahan" registration={register("alamat.kelurahan")} error={errors.alamat?.kelurahan?.message} disabled={isSubmitting} />
          <FormInput id="create-district" label="Kecamatan" registration={register("alamat.kecamatan")} error={errors.alamat?.kecamatan?.message} disabled={isSubmitting} />
          <FormInput id="create-city" label="Kota/Kabupaten" registration={register("alamat.kota")} error={errors.alamat?.kota?.message} disabled={isSubmitting} />
          <FormInput id="create-province" label="Provinsi" registration={register("alamat.provinsi")} error={errors.alamat?.provinsi?.message} disabled={isSubmitting} />
        </fieldset>

        <fieldset className="grid gap-5 rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-6 sm:grid-cols-2 lg:grid-cols-3">
          <legend className="px-2 text-lg font-bold">KTP dan kontrak</legend>
          <FormInput id="create-ktp" label="Nomor KTP" inputMode="numeric" registration={register("ktp.nomor_ktp")} error={errors.ktp?.nomor_ktp?.message} disabled={isSubmitting} />
          <FormInput id="create-contract-number" label="Nomor kontrak" registration={register("kontrak.nomor_kontrak")} error={errors.kontrak?.nomor_kontrak?.message} disabled={isSubmitting} />
          <EmployeeSelectField id="create-contract-type" label="Jenis kontrak" registration={register("kontrak.jenis_kontrak")} error={errors.kontrak?.jenis_kontrak?.message} disabled={isSubmitting}>
            <option value="PKWT">PKWT</option>
            <option value="PKWTT">PKWTT</option>
          </EmployeeSelectField>
          <FormInput id="create-contract-start" label="Tanggal mulai kontrak" type="date" registration={register("kontrak.tanggal_mulai")} error={errors.kontrak?.tanggal_mulai?.message} disabled={isSubmitting} />
          <FormInput id="create-contract-end" label="Tanggal berakhir kontrak" type="date" registration={register("kontrak.tanggal_berakhir")} error={errors.kontrak?.tanggal_berakhir?.message} disabled={isSubmitting} />
        </fieldset>

        <BpjsNpwpFields register={register} errors={errors} disabled={isSubmitting} idPrefix="create" />
        <EmergencyContactsFields control={control} register={register} errors={errors} disabled={isSubmitting} idPrefix="create" />
        <EducationFields control={control} register={register} errors={errors} disabled={isSubmitting} idPrefix="create" />
        <PositionHistoryFields
          control={control}
          register={register}
          errors={errors}
          disabled={isSubmitting}
          idPrefix="create"
          departments={departments.data}
        />
        <CurrentSalaryFields register={register} errors={errors} disabled={isSubmitting} idPrefix="create" />

        <div className="flex flex-wrap gap-3">
          <Button type="submit" disabled={isSubmitting}>{isSubmitting ? "Menyimpan…" : "Simpan karyawan"}</Button>
          <Link to="/app/karyawan" className="inline-flex min-h-10 items-center px-3 font-semibold text-slate-600">Batal</Link>
        </div>
      </form>
    </section>
  );
};
