import { useFieldArray, useWatch } from "react-hook-form";

import { FormInput } from "../../../components/form/FormInput";
import { Button } from "../../../components/ui/Button";
import { usePositions } from "../hooks/useEmployees";
import { EmployeeSelectField } from "./EmployeeSelectField";

/**
 * Bagian form Tambah/Ubah Karyawan untuk detail opsional yang sebelumnya hanya bisa dibaca:
 * BPJS, NPWP, Kontak Darurat, Pendidikan, Riwayat Jabatan, dan Gaji bulan berjalan (lihat
 * OpenAPI 0.6.0 / keputusan D-036). Kontak Darurat, Pendidikan, dan Riwayat Jabatan memakai
 * semantik replace-all sehingga baris yang dikirim selalu daftar lengkap yang berlaku;
 * dokumen pendukung BPJS/NPWP diunggah terpisah lewat bagian "Dokumen karyawan" pada
 * halaman detail setelah karyawan tersimpan.
 */

const fieldsetClassName =
  "grid gap-5 rounded-xl border border-slate-900/10 bg-slate-900/[0.03] p-6 sm:grid-cols-2 lg:grid-cols-3";
// Baris berulang (kontak darurat/pendidikan/riwayat jabatan) baru dipecah menjadi beberapa
// kolom mulai breakpoint lg agar select/input tidak berdesakan dan melebar ke luar
// kontainer (pernah terjadi saat pemecahan kolom dimulai dari sm/640px yang terlalu sempit
// untuk 3-4 field sekaligus).
const rowClassName =
  "grid gap-4 rounded-lg border border-slate-900/10 p-4 sm:col-span-2 lg:col-span-3";
const fullWidthClassName = "sm:col-span-2 lg:col-span-3";
const removeButtonClassName =
  "inline-flex min-h-9 items-center rounded-lg border border-slate-900/15 px-3 text-sm font-semibold text-slate-700 hover:bg-slate-900/5";

export const BpjsNpwpFields = ({ register, errors, disabled, idPrefix }) => (
  <fieldset className={fieldsetClassName}>
    <legend className="px-2 text-lg font-bold">BPJS dan NPWP</legend>
    <FormInput
      id={`${idPrefix}-bpjs-kesehatan`}
      label="Nomor BPJS Kesehatan"
      inputMode="numeric"
      registration={register("bpjs.nomor_kesehatan")}
      error={errors.bpjs?.nomor_kesehatan?.message}
      disabled={disabled}
    />
    <FormInput
      id={`${idPrefix}-bpjs-ketenagakerjaan`}
      label="Nomor BPJS Ketenagakerjaan"
      inputMode="numeric"
      registration={register("bpjs.nomor_ketenagakerjaan")}
      error={errors.bpjs?.nomor_ketenagakerjaan?.message}
      disabled={disabled}
    />
    <FormInput
      id={`${idPrefix}-npwp`}
      label="Nomor NPWP"
      registration={register("npwp.nomor_npwp")}
      error={errors.npwp?.nomor_npwp?.message}
      disabled={disabled}
      description="Dokumen BPJS/NPWP (gambar atau PDF) diunggah lewat bagian Dokumen karyawan pada halaman detail setelah data ini disimpan."
    />
  </fieldset>
);

export const EmergencyContactsFields = ({ control, register, errors, disabled, idPrefix }) => {
  const { fields, append, remove } = useFieldArray({ control, name: "kontak_darurat" });
  return (
    <fieldset className={fieldsetClassName}>
      <legend className="px-2 text-lg font-bold">Kontak darurat</legend>
      {fields.length === 0 && (
        <p className={`text-sm text-slate-500 ${fullWidthClassName}`}>Belum ada kontak darurat.</p>
      )}
      {fields.map((field, index) => (
        <div key={field.id} className={`${rowClassName} lg:grid-cols-3`}>
          <div className="min-w-0">
            <FormInput
              id={`${idPrefix}-emergency-${index}-nama`}
              label="Nama"
              registration={register(`kontak_darurat.${index}.nama`)}
              error={errors.kontak_darurat?.[index]?.nama?.message}
              disabled={disabled}
            />
          </div>
          <div className="min-w-0">
            <FormInput
              id={`${idPrefix}-emergency-${index}-hubungan`}
              label="Hubungan"
              registration={register(`kontak_darurat.${index}.hubungan`)}
              error={errors.kontak_darurat?.[index]?.hubungan?.message}
              disabled={disabled}
            />
          </div>
          <div className="flex min-w-0 items-end gap-3">
            <div className="min-w-0 flex-1">
              <FormInput
                id={`${idPrefix}-emergency-${index}-telepon`}
                label="Nomor telepon"
                inputMode="tel"
                registration={register(`kontak_darurat.${index}.nomor_telepon`)}
                error={errors.kontak_darurat?.[index]?.nomor_telepon?.message}
                disabled={disabled}
              />
            </div>
            <Button type="button" variant="secondary" className={removeButtonClassName} disabled={disabled} onClick={() => remove(index)}>
              Hapus
            </Button>
          </div>
        </div>
      ))}
      <div className={fullWidthClassName}>
        <Button
          type="button"
          variant="secondary"
          disabled={disabled}
          onClick={() => append({ nama: "", hubungan: "", nomor_telepon: "" })}
        >
          Tambah kontak darurat
        </Button>
      </div>
    </fieldset>
  );
};

export const EducationFields = ({ control, register, errors, disabled, idPrefix }) => {
  const { fields, append, remove } = useFieldArray({ control, name: "pendidikan" });
  return (
    <fieldset className={fieldsetClassName}>
      <legend className="px-2 text-lg font-bold">Pendidikan</legend>
      {fields.length === 0 && (
        <p className={`text-sm text-slate-500 ${fullWidthClassName}`}>Belum ada riwayat pendidikan.</p>
      )}
      {fields.map((field, index) => (
        <div key={field.id} className={`${rowClassName} lg:grid-cols-3`}>
          <div className="min-w-0">
            <EmployeeSelectField
              id={`${idPrefix}-education-${index}-jenjang`}
              label="Jenjang"
              registration={register(`pendidikan.${index}.jenjang`)}
              error={errors.pendidikan?.[index]?.jenjang?.message}
              disabled={disabled}
            >
              <option value="">Pilih jenjang</option>
              <option value="SD">SD</option>
              <option value="SMP">SMP</option>
              <option value="SMA/SMK">SMA/SMK</option>
              <option value="D3">D3</option>
              <option value="S1">S1</option>
              <option value="S2">S2</option>
              <option value="S3">S3</option>
            </EmployeeSelectField>
          </div>
          <div className="min-w-0">
            <FormInput
              id={`${idPrefix}-education-${index}-institusi`}
              label="Institusi/sekolah"
              registration={register(`pendidikan.${index}.institusi`)}
              error={errors.pendidikan?.[index]?.institusi?.message}
              disabled={disabled}
            />
          </div>
          <div className="flex min-w-0 items-end gap-3">
            <div className="min-w-0 flex-1">
              <FormInput
                id={`${idPrefix}-education-${index}-tahun`}
                label="Tahun lulus"
                inputMode="numeric"
                registration={register(`pendidikan.${index}.tahun_lulus`)}
                error={errors.pendidikan?.[index]?.tahun_lulus?.message}
                disabled={disabled}
              />
            </div>
            <Button type="button" variant="secondary" className={removeButtonClassName} disabled={disabled} onClick={() => remove(index)}>
              Hapus
            </Button>
          </div>
        </div>
      ))}
      <div className={fullWidthClassName}>
        <Button
          type="button"
          variant="secondary"
          disabled={disabled}
          onClick={() => append({ jenjang: "", institusi: "", tahun_lulus: "" })}
        >
          Tambah pendidikan
        </Button>
      </div>
    </fieldset>
  );
};

const PositionHistoryRow = ({ control, register, errors, disabled, idPrefix, index, departments, remove }) => {
  const departmentId = register(`riwayat_jabatan.${index}.department_id`);
  const watchedDepartmentId = useWatch({ control, name: `riwayat_jabatan.${index}.department_id` });
  const positions = usePositions(watchedDepartmentId);

  return (
    <div className={`${rowClassName} xl:grid-cols-4`}>
      <div className="min-w-0">
        <EmployeeSelectField
          id={`${idPrefix}-history-${index}-department`}
          label="Departemen"
          registration={departmentId}
          error={errors.riwayat_jabatan?.[index]?.department_id?.message}
          disabled={disabled}
        >
          <option value="">Tidak ditentukan</option>
          {(departments ?? []).map((item) => (
            <option key={item.id} value={item.id}>{item.nama}</option>
          ))}
        </EmployeeSelectField>
      </div>
      <div className="min-w-0">
        <EmployeeSelectField
          id={`${idPrefix}-history-${index}-position`}
          label="Jabatan"
          registration={register(`riwayat_jabatan.${index}.position_id`)}
          error={errors.riwayat_jabatan?.[index]?.position_id?.message}
          disabled={disabled || !watchedDepartmentId}
        >
          <option value="">Tidak ditentukan</option>
          {(positions.data ?? []).map((item) => (
            <option key={item.id} value={item.id}>{item.nama}</option>
          ))}
        </EmployeeSelectField>
      </div>
      <div className="min-w-0">
        <FormInput
          id={`${idPrefix}-history-${index}-mulai`}
          label="Tanggal mulai"
          type="date"
          registration={register(`riwayat_jabatan.${index}.tanggal_mulai`)}
          error={errors.riwayat_jabatan?.[index]?.tanggal_mulai?.message}
          disabled={disabled}
        />
      </div>
      <div className="flex min-w-0 items-end gap-3">
        <div className="min-w-0 flex-1">
          <FormInput
            id={`${idPrefix}-history-${index}-selesai`}
            label="Tanggal selesai"
            type="date"
            registration={register(`riwayat_jabatan.${index}.tanggal_selesai`)}
            error={errors.riwayat_jabatan?.[index]?.tanggal_selesai?.message}
            disabled={disabled}
          />
        </div>
        <Button type="button" variant="secondary" className={removeButtonClassName} disabled={disabled} onClick={() => remove(index)}>
          Hapus
        </Button>
      </div>
    </div>
  );
};

export const PositionHistoryFields = ({ control, register, errors, disabled, idPrefix, departments }) => {
  const { fields, append, remove } = useFieldArray({ control, name: "riwayat_jabatan" });
  return (
    <fieldset className={fieldsetClassName}>
      <legend className="px-2 text-lg font-bold">Riwayat jabatan</legend>
      <p className={`text-sm text-slate-500 ${fullWidthClassName}`}>
        Riwayat posisi sebelumnya di perusahaan ini, bukan posisi yang sedang berjalan saat ini.
      </p>
      {fields.length === 0 && (
        <p className={`text-sm text-slate-500 ${fullWidthClassName}`}>Belum ada riwayat jabatan.</p>
      )}
      {fields.map((field, index) => (
        <PositionHistoryRow
          key={field.id}
          control={control}
          register={register}
          errors={errors}
          disabled={disabled}
          idPrefix={idPrefix}
          index={index}
          departments={departments}
          remove={remove}
        />
      ))}
      <div className={fullWidthClassName}>
        <Button
          type="button"
          variant="secondary"
          disabled={disabled}
          onClick={() => append({ department_id: "", position_id: "", tanggal_mulai: "", tanggal_selesai: "" })}
        >
          Tambah riwayat jabatan
        </Button>
      </div>
    </fieldset>
  );
};

export const CurrentSalaryFields = ({ register, errors, disabled, idPrefix }) => (
  <fieldset className={fieldsetClassName}>
    <legend className="px-2 text-lg font-bold">Gaji bulan berjalan</legend>
    <FormInput
      id={`${idPrefix}-salary-periode`}
      label="Periode (YYYY-MM)"
      type="month"
      registration={register("gaji_berjalan.periode")}
      error={errors.gaji_berjalan?.periode?.message}
      disabled={disabled}
    />
    <FormInput
      id={`${idPrefix}-salary-pokok`}
      label="Gaji pokok"
      inputMode="numeric"
      registration={register("gaji_berjalan.gaji_pokok")}
      error={errors.gaji_berjalan?.gaji_pokok?.message}
      disabled={disabled}
    />
    <FormInput
      id={`${idPrefix}-salary-tunjangan`}
      label="Tunjangan"
      inputMode="numeric"
      registration={register("gaji_berjalan.tunjangan")}
      error={errors.gaji_berjalan?.tunjangan?.message}
      disabled={disabled}
    />
    <FormInput
      id={`${idPrefix}-salary-potongan`}
      label="Potongan"
      inputMode="numeric"
      registration={register("gaji_berjalan.potongan")}
      error={errors.gaji_berjalan?.potongan?.message}
      disabled={disabled}
      description="Take home pay dihitung otomatis oleh server (gaji pokok + tunjangan − potongan)."
    />
  </fieldset>
);
