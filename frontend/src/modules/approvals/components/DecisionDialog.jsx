import { useEffect, useRef, useState } from "react";

import { Button } from "../../../components/ui/Button";

/**
 * Dialog keputusan approval. Penolakan mewajibkan catatan minimal lima karakter sesuai
 * schema DecisionRequest; delegasi memakai dialog yang sama dengan catatan wajib.
 */
export const DecisionDialog = ({
  open,
  mode,
  busy,
  error,
  onSubmit,
  onCancel,
}) => {
  const [note, setNote] = useState("");
  const [noteError, setNoteError] = useState("");
  const dialogRef = useRef(null);
  const firstFieldRef = useRef(null);

  useEffect(() => {
    if (open) {
      setNote("");
      setNoteError("");
      firstFieldRef.current?.focus();
    }
  }, [open, mode]);

  if (!open) return null;

  const requiresNote = mode === "tolak" || mode === "delegate";
  const title =
    mode === "setujui"
      ? "Setujui pengajuan?"
      : mode === "tolak"
        ? "Tolak pengajuan?"
        : "Delegasikan ke HR?";
  const description =
    mode === "setujui"
      ? "Pengajuan diteruskan ke tahap berikutnya atau disetujui final oleh server."
      : mode === "tolak"
        ? "Penolakan mengakhiri alur dan wajib disertai catatan."
        : "Keputusan dipindahkan ke HR. Catatan wajib diisi agar alasan delegasi tercatat.";

  const handleSubmit = (event) => {
    event.preventDefault();
    const trimmed = note.trim();
    if (requiresNote && trimmed.length < 5) {
      setNoteError("Catatan wajib diisi minimal 5 karakter.");
      return;
    }
    setNoteError("");
    onSubmit(trimmed);
  };

  const handleKeyDown = (event) => {
    if (event.key === "Escape" && !busy) {
      event.stopPropagation();
      onCancel();
      return;
    }
    if (event.key !== "Tab") return;
    const focusable = dialogRef.current?.querySelectorAll(
      "button:not([disabled]), textarea, input, [href]",
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
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 p-4">
      <form
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="decision-dialog-title"
        onKeyDown={handleKeyDown}
        onSubmit={handleSubmit}
        noValidate
        className="w-full max-w-md rounded-xl border border-white/10 bg-slate-900 p-6 shadow-xl"
      >
        <h2 id="decision-dialog-title" className="text-xl font-bold text-white">
          {title}
        </h2>
        <p className="mt-2 text-sm text-slate-300">{description}</p>

        <label htmlFor="decision-note" className="mt-5 block text-sm font-medium text-slate-200">
          Catatan {requiresNote ? "(wajib)" : "(opsional)"}
        </label>
        <textarea
          id="decision-note"
          ref={firstFieldRef}
          value={note}
          rows={4}
          maxLength={1000}
          disabled={busy}
          aria-invalid={Boolean(noteError)}
          aria-describedby={noteError ? "decision-note-error" : undefined}
          onChange={(event) => setNote(event.target.value)}
          className="mt-2 w-full rounded-lg border border-white/15 bg-slate-950 p-3 text-white outline-none focus:border-cyan-300 disabled:opacity-60"
        />
        {noteError && (
          <p id="decision-note-error" role="alert" className="mt-2 text-sm text-rose-300">
            {noteError}
          </p>
        )}
        {error && (
          <p role="alert" className="mt-3 rounded-lg border border-rose-300/30 bg-rose-300/10 p-3 text-sm text-rose-200">
            {error}
          </p>
        )}

        <div className="mt-6 flex flex-wrap justify-end gap-3">
          <Button type="button" variant="secondary" onClick={onCancel} disabled={busy}>
            Batal
          </Button>
          <Button type="submit" disabled={busy}>
            {busy ? "Memproses…" : "Konfirmasi"}
          </Button>
        </div>
      </form>
    </div>
  );
};
