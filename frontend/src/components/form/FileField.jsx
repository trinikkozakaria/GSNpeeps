import { useId, useRef } from "react";

import { Button } from "../ui/Button";

export const formatFileSize = (bytes) => {
  if (!Number.isFinite(bytes) || bytes < 0) return "";
  if (bytes < 1024) return `${bytes} B`;
  const kilobytes = bytes / 1024;
  if (kilobytes < 1024) return `${kilobytes.toLocaleString("id-ID", { maximumFractionDigits: 1 })} KB`;
  return `${(kilobytes / 1024).toLocaleString("id-ID", { maximumFractionDigits: 2 })} MB`;
};

/**
 * Input berkas dengan pratinjau nama dan ukuran serta tombol hapus. Isi berkas tidak pernah
 * dibaca ke memory maupun disimpan di state global; hanya objek File yang dipegang pemanggil.
 */
export const FileField = ({
  id,
  label,
  accept,
  description,
  error,
  file,
  disabled,
  onFileChange,
}) => {
  const inputRef = useRef(null);
  const generatedId = useId();
  const fieldId = id ?? generatedId;
  const descriptionId = description ? `${fieldId}-description` : undefined;
  const errorId = error ? `${fieldId}-error` : undefined;
  const describedBy = [descriptionId, errorId].filter(Boolean).join(" ") || undefined;

  const clear = () => {
    if (inputRef.current) inputRef.current.value = "";
    onFileChange(null);
  };

  return (
    <div>
      <label htmlFor={fieldId} className="mb-2 block text-sm font-medium text-slate-700">
        {label}
      </label>
      <input
        ref={inputRef}
        id={fieldId}
        type="file"
        accept={accept}
        disabled={disabled}
        aria-invalid={Boolean(error)}
        aria-describedby={describedBy}
        onChange={(event) => onFileChange(event.target.files?.[0] ?? null)}
        className="block w-full rounded-lg border border-slate-900/15 bg-white p-2 text-sm text-slate-700 file:mr-3 file:min-h-9 file:rounded-md file:border-0 file:bg-slate-900/10 file:px-3 file:text-sm file:font-semibold file:text-slate-900 disabled:opacity-60"
      />
      {file && (
        <p className="mt-2 flex flex-wrap items-center gap-3 text-sm text-slate-600">
          <span className="font-medium text-slate-900">{file.name}</span>
          <span className="text-slate-500">{formatFileSize(file.size)}</span>
          <Button
            type="button"
            variant="secondary"
            className="min-h-9 px-3 py-1"
            onClick={clear}
            disabled={disabled}
          >
            Hapus berkas
          </Button>
        </p>
      )}
      {description && (
        <p id={descriptionId} className="mt-2 text-xs text-slate-500">
          {description}
        </p>
      )}
      {error && (
        <p id={errorId} role="alert" className="mt-2 text-sm text-rose-700">
          {error}
        </p>
      )}
    </div>
  );
};
