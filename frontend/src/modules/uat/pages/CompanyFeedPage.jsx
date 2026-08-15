import { useMutation, useQuery } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { createButton, Editor, EditorProvider, Toolbar } from "react-simple-wysiwyg";

import { Button } from "../../../components/ui/Button";
import { formatDateTime } from "../../../lib/format";
import { queryClient } from "../../../lib/query/query-client";
import { createFeedRequest, feedsRequest } from "../api/uat-api";

export const CompanyFeedList = () => {
  const feeds = useQuery({
    queryKey: ["company-feed"],
    queryFn: ({ signal }) => feedsRequest(signal),
  });

  if (feeds.isPending) return <p role="status">Memuat company feed…</p>;
  if (feeds.isError) {
    return (
      <div role="alert" className="rounded-xl border border-red-300 bg-red-50 p-4 text-red-700">
        <p>Company feed belum dapat dimuat. {feeds.error?.message}</p>
        <Button className="mt-3" variant="secondary" onClick={() => feeds.refetch()}>Coba lagi</Button>
      </div>
    );
  }
  if ((feeds.data?.length ?? 0) === 0) {
    return <p className="rounded-xl border border-dashed p-5 text-slate-600">Belum ada informasi perusahaan.</p>;
  }

  return (
    <div className="grid gap-4">
      {feeds.data.map((feed) => (
        <article key={feed.id} className="rounded-xl border border-slate-900/10 p-5">
          <h3 className="text-xl font-bold">{feed.judul}</h3>
          <p className="mt-1 text-xs text-slate-600">{feed.penulis} · {formatDateTime(feed.published_at)}</p>
          <div className="wysiwyg-content mt-4 max-w-none" dangerouslySetInnerHTML={{ __html: feed.konten_html }} />
        </article>
      ))}
    </div>
  );
};

// Tombol toolbar dibuat lewat createButton bawaan react-simple-wysiwyg — persis fungsi yang
// dipakai BtnBold/BtnItalic/dst di dalam library (fokus editor lewat EditorContext lalu
// document.execCommand pada mousedown), hanya label dan style yang diganti ke Bahasa
// Indonesia dan Tailwind. Interaksi format sepenuhnya ditangani library, bukan kode kustom.
// Varian data-[active=true] diberi `!` (important) karena mouse pengguna biasanya masih
// berada di atas tombol tepat setelah diklik; tanpa ini, urutan sumber Tailwind membuat
// hover:bg-slate-900/10 menutupi warna aktif cyan meski data-active sudah true.
const toolbarButtonClassName =
  "inline-flex min-h-10 items-center justify-center rounded-lg px-4 py-2 text-sm font-semibold transition focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 border border-slate-900/15 bg-slate-900/5 text-slate-900 hover:bg-slate-900/10 focus-visible:outline-slate-900 data-[active=true]:border-cyan-700! data-[active=true]:bg-cyan-700! data-[active=true]:text-white! disabled:cursor-not-allowed disabled:opacity-50";

const BtnTebal = createButton("Tebal", "Tebal", "bold");
const BtnMiring = createButton("Miring", "Miring", "italic");
const BtnGarisBawah = createButton("Garis bawah", "Garis bawah", "underline");
const BtnDaftar = createButton("Daftar", "Daftar", "insertUnorderedList");
const BtnNomor = createButton("Nomor", "Nomor", "insertOrderedList");

export const CompanyFeedPage = () => {
  const editorRef = useRef(null);
  const [title, setTitle] = useState("");
  const [html, setHTML] = useState("");
  const [contentText, setContentText] = useState("");
  const [successMessage, setSuccessMessage] = useState("");
  const create = useMutation({ mutationFn: createFeedRequest });
  const canPublish = Boolean(title.trim() && contentText.trim()) && !create.isPending;

  const handleSubmit = async (event) => {
    event.preventDefault();
    if (!canPublish) return;
    setSuccessMessage("");
    try {
      await create.mutateAsync({ judul: title.trim(), konten_html: html });
      setTitle("");
      setHTML("");
      setContentText("");
      setSuccessMessage("Company feed berhasil diterbitkan.");
      await queryClient.invalidateQueries({ queryKey: ["company-feed"] });
    } catch {
      // State error mutation dirender di bawah form.
    }
  };

  return (
    <section aria-labelledby="company-feed-title">
      <p className="text-sm font-semibold uppercase tracking-widest text-cyan-700">Komunikasi</p>
      <h1 id="company-feed-title" className="mt-2 text-3xl font-bold">Company Feed</h1>
      <p className="mt-2 text-slate-600">Informasi yang diterbitkan tampil di beranda seluruh karyawan.</p>
      <form className="my-6 grid gap-4 rounded-xl border border-slate-900/10 p-5" onSubmit={handleSubmit}>
        <label className="text-sm font-medium">
          Judul
          <input className="mt-2 min-h-10 w-full rounded-lg border px-3" value={title} onChange={(event) => setTitle(event.target.value)} required disabled={create.isPending} />
        </label>
        <span id="feed-content-label" className="text-sm font-medium">Konten</span>
        {/* EditorProvider menyediakan EditorContext yang dipakai tombol toolbar (lewat
            useEditorState) untuk menemukan elemen contentEditable aktif; Toolbar dan Editor
            boleh menjadi sibling selama sama-sama berada di dalam provider ini. */}
        <EditorProvider>
          <Toolbar role="toolbar" aria-label="Format konten" className="flex flex-wrap gap-2 bg-transparent p-0">
            <BtnTebal className={toolbarButtonClassName} disabled={create.isPending} />
            <BtnMiring className={toolbarButtonClassName} disabled={create.isPending} />
            <BtnGarisBawah className={toolbarButtonClassName} disabled={create.isPending} />
            <BtnDaftar className={toolbarButtonClassName} disabled={create.isPending} />
            <BtnNomor className={toolbarButtonClassName} disabled={create.isPending} />
          </Toolbar>
          <Editor
            ref={editorRef}
            value={html}
            onChange={(event) => {
              setHTML(event.target.value);
              setContentText(editorRef.current?.textContent ?? "");
            }}
            disabled={create.isPending}
            role="textbox"
            aria-labelledby="feed-content-label"
            aria-multiline="true"
            containerProps={{
              className:
                "min-h-40 rounded-lg border focus-within:outline focus-within:outline-2 focus-within:outline-cyan-700",
            }}
            className="min-h-40 p-3 focus:outline-none"
          />
        </EditorProvider>
        <Button type="submit" disabled={!canPublish}>{create.isPending ? "Menerbitkan…" : "Terbitkan"}</Button>
        {create.isError && <p role="alert" className="text-red-700">Feed belum dapat diterbitkan. {create.error?.message}</p>}
        {successMessage && <p role="status" className="text-emerald-700">{successMessage}</p>}
      </form>
      <CompanyFeedList />
    </section>
  );
};
