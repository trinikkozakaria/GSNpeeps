import { useEffect, useRef } from "react";

import { Button } from "../ui/Button";

/**
 * Dialog konfirmasi modal yang dapat diakses keyboard. Fokus dipindahkan ke tombol batal
 * saat dibuka, Escape menutup dialog, fokus dikurung di dalam dialog, lalu dikembalikan
 * ke elemen pemicu saat dialog ditutup.
 */
export const ConfirmDialog = ({
  open,
  title,
  description,
  confirmLabel,
  cancelLabel = "Batal",
  destructive = false,
  busy = false,
  error,
  onConfirm,
  onCancel,
}) => {
  const dialogRef = useRef(null);
  const cancelRef = useRef(null);

  useEffect(() => {
    if (!open) return undefined;

    const previousFocus = document.activeElement;
    cancelRef.current?.focus();

    return () => previousFocus?.focus();
  }, [open]);

  if (!open) return null;

  const handleKeyDown = (event) => {
    if (event.key === "Escape" && !busy) {
      event.stopPropagation();
      onCancel();
      return;
    }
    if (event.key !== "Tab") return;
    const focusable = dialogRef.current?.querySelectorAll(
      "button:not([disabled]), [href], input, select, textarea, [tabindex]:not([tabindex='-1'])",
    );
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
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/60 p-4">
      <div
        ref={dialogRef}
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="confirm-dialog-title"
        aria-describedby="confirm-dialog-description"
        onKeyDown={handleKeyDown}
        className="w-full max-w-md rounded-xl border border-slate-900/10 bg-white p-6 shadow-xl"
      >
        <h2 id="confirm-dialog-title" className="text-xl font-bold text-slate-900">
          {title}
        </h2>
        <p id="confirm-dialog-description" className="mt-3 text-sm text-slate-600">
          {description}
        </p>
        {error && (
          <p role="alert" className="mt-4 rounded-lg border border-rose-300/30 bg-rose-300/10 p-3 text-sm text-rose-700">
            {error}
          </p>
        )}
        <div className="mt-6 flex flex-wrap justify-end gap-3">
          <Button ref={cancelRef} variant="secondary" onClick={onCancel} disabled={busy}>
            {cancelLabel}
          </Button>
          <Button
            onClick={onConfirm}
            disabled={busy}
            className={destructive ? "bg-rose-600 text-white hover:bg-rose-800" : undefined}
          >
            {busy ? "Memproses…" : confirmLabel}
          </Button>
        </div>
      </div>
    </div>
  );
};
