import { useMutation, useQuery } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import Editor from "react-simple-wysiwyg";

import { Pagination } from "../../../components/data-table/Pagination";
import { ConfirmDialog } from "../../../components/feedback/ConfirmDialog";
import { Button } from "../../../components/ui/Button";
import { queryClient } from "../../../lib/query/query-client";
import {
  createFeedRequest,
  deleteFeedAttachmentRequest,
  deleteFeedRequest,
  feedsRequest,
  updateFeedRequest,
  uploadFeedAttachmentRequest,
} from "../api/uat-api";
import { FeedCard } from "../components/FeedCard";

// invalidateFeedQueries mencocokkan awalan key ["company-feed", ...], sehingga daftar
// infinite scroll Beranda dan daftar paginated di halaman ini sama-sama menjadi stale.
const invalidateFeedQueries = () => queryClient.invalidateQueries({ queryKey: ["company-feed"] });

const emptyForm = { title: "", html: "" };
const maxAttachmentBytes = 5 * 1024 * 1024;
const allowedAttachmentTypes = new Set(["application/pdf", "image/png", "image/jpeg"]);

export const CompanyFeedPage = () => {
  document.title = "Company Feed — GSNpeeps";
  const editorRef = useRef(null);
  const formRef = useRef(null);
  const [params, setParams] = useSearchParams();
  const page = Number.parseInt(params.get("page") ?? "1", 10) || 1;

  const [editingId, setEditingId] = useState(null);
  const [form, setForm] = useState(emptyForm);
  const [contentText, setContentText] = useState("");
  const [successMessage, setSuccessMessage] = useState("");
  const [deletingFeed, setDeletingFeed] = useState(null);
  const [deleteError, setDeleteError] = useState("");
  const [attachmentFiles, setAttachmentFiles] = useState([]);
  const [attachmentError, setAttachmentError] = useState("");
  const [uploadingAttachments, setUploadingAttachments] = useState(false);
  const [deletingAttachmentId, setDeletingAttachmentId] = useState("");

  const feeds = useQuery({
    queryKey: ["company-feed", "admin", page],
    queryFn: ({ signal }) => feedsRequest({ page, limit: 20 }, signal),
  });

  const create = useMutation({ mutationFn: createFeedRequest });
  const update = useMutation({ mutationFn: ({ id, payload }) => updateFeedRequest(id, payload) });
  const remove = useMutation({ mutationFn: deleteFeedRequest });

  const saving = create.isPending || update.isPending || uploadingAttachments;
  const canPublish = Boolean(form.title.trim() && contentText.trim()) && !saving;
  const isEditing = editingId !== null;
  const editingFeed = feeds.data?.items.find((feed) => feed.id === editingId);

  const resetForm = () => {
    setEditingId(null);
    setForm(emptyForm);
    setContentText("");
    setAttachmentFiles([]);
    setAttachmentError("");
  };

  const startEdit = (feed) => {
    setEditingId(feed.id);
    setForm({ title: feed.judul, html: feed.konten_html });
    // editorRef.current.textContent masih berisi konten LAMA sampai react-simple-wysiwyg
    // menyinkronkan DOM-nya ke value baru pada render berikutnya, jadi plain text diekstrak
    // langsung dari HTML yang tersimpan alih-alih dibaca dari ref.
    setContentText(new DOMParser().parseFromString(feed.konten_html, "text/html").body.textContent ?? "");
    setSuccessMessage("");
    setAttachmentFiles([]);
    setAttachmentError("");
    formRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
  };

  const handleSubmit = async (event) => {
    event.preventDefault();
    if (!canPublish) return;
    setSuccessMessage("");
    setAttachmentError("");
    const payload = { judul: form.title.trim(), konten_html: form.html };
    try {
      let targetID = editingId;
      if (isEditing) {
        await update.mutateAsync({ id: editingId, payload });
      } else {
        const created = await create.mutateAsync(payload);
        targetID = created.id;
      }

      let attachmentFailure = null;
      if (attachmentFiles.length > 0) {
        setUploadingAttachments(true);
        try {
          for (const file of attachmentFiles) {
            // Upload berurutan menjaga batas memory dan membuat pesan error deterministik.
            // eslint-disable-next-line no-await-in-loop
            await uploadFeedAttachmentRequest(targetID, file);
          }
        } catch (error) {
          attachmentFailure = error;
        } finally {
          setUploadingAttachments(false);
        }
      }
      await invalidateFeedQueries();
      if (attachmentFailure) {
        setEditingId(targetID);
        setAttachmentFiles([]);
        setAttachmentError(`Feed sudah tersimpan, tetapi ada attachment yang gagal diunggah. ${attachmentFailure.message}`);
        return;
      }
      setSuccessMessage(isEditing ? "Company feed berhasil diperbarui." : "Company feed berhasil diterbitkan.");
      resetForm();
    } catch {
      // State error mutation dirender di bawah form.
    }
  };

  const selectAttachments = (event) => {
    const files = Array.from(event.target.files ?? []);
    event.target.value = "";
    setAttachmentError("");
    const existingCount = editingFeed?.attachments?.length ?? 0;
    if (files.length + existingCount > 5) {
      setAttachmentError("Maksimal 5 attachment per feed.");
      return;
    }
    const invalid = files.find((file) => !allowedAttachmentTypes.has(file.type) || file.size > maxAttachmentBytes || file.size === 0);
    if (invalid) {
      setAttachmentError(`${invalid.name}: gunakan PDF/PNG/JPG/JPEG dengan ukuran maksimal 5 MB.`);
      return;
    }
    setAttachmentFiles(files);
  };

  const deleteAttachment = async (attachment) => {
    setAttachmentError("");
    setDeletingAttachmentId(attachment.id);
    try {
      await deleteFeedAttachmentRequest(editingId, attachment.id);
      await invalidateFeedQueries();
      setSuccessMessage(`Attachment ${attachment.file_name} berhasil dihapus.`);
    } catch (error) {
      setAttachmentError(`Attachment belum dapat dihapus. ${error.message}`);
    } finally {
      setDeletingAttachmentId("");
    }
  };

  const confirmDelete = async () => {
    if (!deletingFeed) return;
    setDeleteError("");
    try {
      await remove.mutateAsync(deletingFeed.id);
      if (editingId === deletingFeed.id) resetForm();
      setDeletingFeed(null);
      await invalidateFeedQueries();
    } catch (error) {
      setDeleteError(error.message);
    }
  };

  const activeMutationError = isEditing ? update.error : create.error;
  const activeMutationIsError = isEditing ? update.isError : create.isError;

  return (
    <section aria-labelledby="company-feed-title">
      <p className="text-sm font-semibold uppercase tracking-widest text-cyan-700">Komunikasi</p>
      <h1 id="company-feed-title" className="mt-2 text-3xl font-bold">Company Feed</h1>
      <p className="mt-2 text-slate-600">Informasi yang diterbitkan tampil di beranda seluruh karyawan.</p>
      <form ref={formRef} className="my-6 grid gap-4 rounded-xl border border-slate-900/10 p-5" onSubmit={handleSubmit}>
        <div className="flex items-center justify-between gap-3">
          <h2 className="text-lg font-bold">{isEditing ? "Edit Feed" : "Feed Baru"}</h2>
          {isEditing && (
            <Button type="button" variant="secondary" onClick={resetForm} disabled={saving}>
              Batal edit
            </Button>
          )}
        </div>
        <label className="text-sm font-medium">
          Judul
          <input
            className="mt-2 min-h-10 w-full rounded-lg border px-3"
            value={form.title}
            onChange={(event) => setForm((current) => ({ ...current, title: event.target.value }))}
            required
            disabled={saving}
          />
        </label>
        <span id="feed-content-label" className="text-sm font-medium">Konten</span>
        {/* Toolbar bawaan react-simple-wysiwyg dipakai apa adanya (tidak di-wrap children,
            sehingga DefaultEditor merender Toolbar internal library-nya sendiri). */}
        <Editor
          ref={editorRef}
          value={form.html}
          onChange={(event) => {
            setForm((current) => ({ ...current, html: event.target.value }));
            setContentText(editorRef.current?.textContent ?? "");
          }}
          disabled={saving}
          role="textbox"
          aria-labelledby="feed-content-label"
          aria-multiline="true"
          containerProps={{
            className:
              "min-h-40 rounded-lg border focus-within:outline focus-within:outline-2 focus-within:outline-cyan-700",
          }}
          className="min-h-40 p-3 focus:outline-none"
        />
        <div>
          <label htmlFor="feed-attachments" className="text-sm font-medium">Attachment (opsional)</label>
          <input
            id="feed-attachments"
            type="file"
            multiple
            accept=".pdf,.png,.jpg,.jpeg,application/pdf,image/png,image/jpeg"
            disabled={saving}
            onChange={selectAttachments}
            className="mt-2 block w-full rounded-lg border border-slate-900/15 bg-white p-3 text-sm"
          />
          <p className="mt-1 text-xs text-slate-500">Maksimal 5 file per feed, masing-masing maksimal 5 MB.</p>
          {attachmentFiles.length > 0 && (
            <ul className="mt-2 space-y-1 text-sm text-slate-600">
              {attachmentFiles.map((file) => <li key={`${file.name}-${file.lastModified}`}>{file.name}</li>)}
            </ul>
          )}
        </div>
        {isEditing && editingFeed?.attachments?.length > 0 && (
          <div className="rounded-lg border border-slate-900/10 p-3">
            <p className="text-sm font-semibold">Attachment tersimpan</p>
            <ul className="mt-2 space-y-2">
              {editingFeed.attachments.map((attachment) => (
                <li key={attachment.id} className="flex flex-wrap items-center justify-between gap-2 text-sm">
                  <span>{attachment.file_name}</span>
                  <Button
                    type="button"
                    variant="secondary"
                    disabled={Boolean(deletingAttachmentId)}
                    onClick={() => deleteAttachment(attachment)}
                    className="min-h-8 px-3 py-1 text-rose-700!"
                  >
                    {deletingAttachmentId === attachment.id ? "Menghapus…" : "Hapus attachment"}
                  </Button>
                </li>
              ))}
            </ul>
          </div>
        )}
        {attachmentError && <p role="alert" className="text-red-700">{attachmentError}</p>}
        <Button type="submit" disabled={!canPublish}>
          {uploadingAttachments ? "Mengunggah attachment…" : saving ? "Menyimpan…" : isEditing ? "Simpan Perubahan" : "Terbitkan"}
        </Button>
        {activeMutationIsError && (
          <p role="alert" className="text-red-700">
            Feed belum dapat disimpan. {activeMutationError?.message}
          </p>
        )}
        {successMessage && <p role="status" className="text-emerald-700">{successMessage}</p>}
      </form>

      <div aria-live="polite">
        {feeds.isPending && <p role="status" className="text-slate-600">Memuat company feed…</p>}
        {feeds.isError && (
          <div role="alert" className="rounded-xl border border-red-300 bg-red-50 p-4 text-red-700">
            <p>Company feed belum dapat dimuat. {feeds.error?.message}</p>
            <Button className="mt-3" variant="secondary" onClick={() => feeds.refetch()}>Coba lagi</Button>
          </div>
        )}
        {feeds.data && feeds.data.items.length === 0 && (
          <p className="rounded-xl border border-slate-900/10 p-5 text-slate-600">Belum ada informasi perusahaan.</p>
        )}
        {feeds.data && feeds.data.items.length > 0 && (
          <>
            <div className="grid gap-4">
              {feeds.data.items.map((feed) => (
                <FeedCard
                  key={feed.id}
                  feed={feed}
                  actions={
                    <>
                      <Button variant="secondary" onClick={() => startEdit(feed)}>
                        Edit
                      </Button>
                      <Button
                        variant="secondary"
                        className="border-rose-300! text-rose-700! hover:bg-rose-50!"
                        onClick={() => {
                          setDeleteError("");
                          setDeletingFeed(feed);
                        }}
                      >
                        Hapus
                      </Button>
                    </>
                  }
                />
              ))}
            </div>
            <Pagination
              meta={feeds.data.meta}
              label="Navigasi halaman company feed"
              onPageChange={(nextPage) => {
                const next = new URLSearchParams(params);
                next.set("page", String(nextPage));
                setParams(next);
              }}
            />
          </>
        )}
      </div>

      <ConfirmDialog
        open={Boolean(deletingFeed)}
        title={`Hapus feed "${deletingFeed?.judul ?? ""}"?`}
        description="Feed yang dihapus tidak dapat dikembalikan dan akan langsung hilang dari Beranda seluruh karyawan."
        confirmLabel="Hapus feed"
        destructive
        busy={remove.isPending}
        error={deleteError}
        onCancel={() => {
          setDeleteError("");
          setDeletingFeed(null);
        }}
        onConfirm={confirmDelete}
      />
    </section>
  );
};
