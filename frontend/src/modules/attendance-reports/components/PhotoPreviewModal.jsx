import { useEffect, useRef } from "react";

/**
 * Modal pratinjau foto absensi. Dipakai live feed agar tabel tidak memuat gambar penuh di
 * setiap baris; foto hanya dimuat saat pengguna membuka modal.
 */
export const PhotoPreviewModal = ({ open, photoUrl, title, onClose }) => {
  const dialogRef = useRef(null);
  const closeButtonRef = useRef(null);

  useEffect(() => {
    if (open) {
      closeButtonRef.current?.focus();
    }
  }, [open]);

  if (!open) return null;

  const handleKeyDown = (event) => {
    if (event.key === "Escape") {
      event.stopPropagation();
      onClose();
      return;
    }
    if (event.key !== "Tab") return;
    const focusable = dialogRef.current?.querySelectorAll("button:not([disabled]), [href]");
    if (!focusable || focusable.length === 0) return;
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/60 p-4"
      onClick={onClose}
    >
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="photo-preview-title"
        onKeyDown={handleKeyDown}
        onClick={(event) => event.stopPropagation()}
        className="w-full max-w-lg rounded-xl border border-slate-900/10 bg-white p-6 shadow-xl"
      >
        <div className="flex items-start justify-between gap-4">
          <h2 id="photo-preview-title" className="text-lg font-bold text-slate-900">
            {title}
          </h2>
          <button
            ref={closeButtonRef}
            type="button"
            onClick={onClose}
            aria-label="Tutup pratinjau foto"
            className="min-h-11 min-w-11 rounded-lg text-slate-600 hover:bg-slate-900/5 hover:text-slate-900 focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-700"
          >
            ✕
          </button>
        </div>
        <div className="mt-4">
          {photoUrl ? (
            <img
              src={photoUrl}
              alt={title}
              className="max-h-[70vh] w-full rounded-lg object-contain"
            />
          ) : (
            <p className="text-sm text-slate-500">Foto tidak tersedia.</p>
          )}
        </div>
      </div>
    </div>
  );
};
