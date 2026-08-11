import { useState } from "react";

import { Button } from "../ui/Button";
import { FileField } from "./FileField";

const maxPhotoBytes = 5 * 1024 * 1024;
const allowedExtensions = [".jpg", ".jpeg", ".png"];

const validatePhotoFile = (file) => {
  if (!file) return "Berkas foto wajib dipilih.";
  const extension = file.name.slice(file.name.lastIndexOf(".")).toLowerCase();
  if (!file.name.includes(".") || !allowedExtensions.includes(extension)) {
    return "Format foto tidak didukung. Gunakan JPG atau PNG.";
  }
  if (file.size > maxPhotoBytes) return "Ukuran foto melebihi batas 5 MB.";
  if (file.size === 0) return "Berkas foto kosong.";
  return "";
};

const uploadErrorMessage = (error) => {
  if (error?.status === 413) return "Ukuran foto melebihi batas 5 MB.";
  if (error?.status === 415) return "Format foto ditolak server. Gunakan JPG atau PNG.";
  if (error?.status === 403) return "Anda tidak memiliki akses untuk mengubah foto ini.";
  return error?.message ?? "Foto belum dapat diunggah.";
};

/**
 * Uploader foto profil lintas modul (dipakai Keamanan Akun untuk foto sendiri dan Detail
 * Karyawan untuk foto yang diperbarui HR). `onUpload` menerima `File` dan mengembalikan
 * promise yang resolve dengan URL foto baru; komponen ini tidak tahu endpoint mana yang
 * dipanggil.
 */
export const PhotoUploadField = ({ idPrefix, currentPhotoUrl, onUpload, disabled }) => {
  const [file, setFile] = useState(null);
  const [fieldError, setFieldError] = useState("");
  const [formError, setFormError] = useState("");
  const [successMessage, setSuccessMessage] = useState("");
  const [isUploading, setIsUploading] = useState(false);
  const [previewUrl, setPreviewUrl] = useState(currentPhotoUrl ?? null);

  const handleSubmit = async (event) => {
    event.preventDefault();
    setFormError("");
    setSuccessMessage("");
    const nextError = validatePhotoFile(file);
    setFieldError(nextError);
    if (nextError) return;

    setIsUploading(true);
    try {
      const nextUrl = await onUpload(file);
      setPreviewUrl(nextUrl ?? previewUrl);
      setFile(null);
      setSuccessMessage("Foto profil berhasil diperbarui.");
    } catch (error) {
      setFormError(uploadErrorMessage(error));
    } finally {
      setIsUploading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} noValidate className="flex flex-wrap items-start gap-5">
      <span className="block h-16 w-16 shrink-0 overflow-hidden rounded-full border border-slate-900/10 bg-slate-100">
        {previewUrl ? (
          <img src={previewUrl} alt="" className="h-full w-full object-cover" />
        ) : (
          <span className="flex h-full w-full items-center justify-center text-xs font-semibold uppercase text-slate-500">
            Foto
          </span>
        )}
      </span>
      <div className="min-w-[16rem] flex-1 space-y-3">
        {formError && (
          <p role="alert" className="rounded-lg border border-rose-300/30 bg-rose-300/10 p-3 text-sm text-rose-700">
            {formError}
          </p>
        )}
        {successMessage && (
          <p role="status" className="rounded-lg border border-emerald-300/30 bg-emerald-300/10 p-3 text-sm text-emerald-700">
            {successMessage}
          </p>
        )}
        <FileField
          id={`${idPrefix}-photo-file`}
          label="Ganti foto profil"
          accept={allowedExtensions.join(",")}
          description="JPG atau PNG, maksimal 5 MB."
          file={file}
          error={fieldError}
          disabled={disabled || isUploading}
          onFileChange={(selected) => {
            setFile(selected);
            setFieldError(selected ? validatePhotoFile(selected) : "");
          }}
        />
        <Button type="submit" variant="secondary" disabled={disabled || isUploading || !file}>
          {isUploading ? "Mengunggah…" : "Unggah foto"}
        </Button>
      </div>
    </form>
  );
};
