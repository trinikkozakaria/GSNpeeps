import { useMutation, useQuery } from "@tanstack/react-query";
import { useRef, useState } from "react";

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
          <div className="prose mt-4 max-w-none" dangerouslySetInnerHTML={{ __html: feed.konten_html }} />
        </article>
      ))}
    </div>
  );
};

const tools = [
  ["bold", "Tebal"],
  ["italic", "Miring"],
  ["underline", "Garis bawah"],
  ["insertUnorderedList", "Daftar"],
  ["insertOrderedList", "Nomor"],
];

export const CompanyFeedPage = () => {
  const editorRef = useRef(null);
  const [title, setTitle] = useState("");
  const [html, setHTML] = useState("");
  const [contentText, setContentText] = useState("");
  const [successMessage, setSuccessMessage] = useState("");
  const create = useMutation({ mutationFn: createFeedRequest });
  const canPublish = Boolean(title.trim() && contentText.trim()) && !create.isPending;

  const command = (name) => {
    editorRef.current?.focus();
    document.execCommand?.(name, false);
    setHTML(editorRef.current?.innerHTML ?? "");
    setContentText(editorRef.current?.textContent ?? "");
  };

  const handleSubmit = async (event) => {
    event.preventDefault();
    if (!canPublish) return;
    setSuccessMessage("");
    try {
      await create.mutateAsync({ judul: title.trim(), konten_html: html });
      setTitle("");
      setHTML("");
      setContentText("");
      if (editorRef.current) editorRef.current.innerHTML = "";
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
        <div className="flex flex-wrap gap-2" role="toolbar" aria-label="Format konten">
          {tools.map(([name, label]) => <Button key={name} variant="secondary" disabled={create.isPending} onMouseDown={(event) => event.preventDefault()} onClick={() => command(name)}>{label}</Button>)}
        </div>
        <div
          ref={editorRef}
          role="textbox"
          aria-labelledby="feed-content-label"
          aria-multiline="true"
          contentEditable={!create.isPending}
          suppressContentEditableWarning
          className="min-h-40 rounded-lg border p-3 focus:outline-cyan-700"
          onInput={(event) => {
            setHTML(event.currentTarget.innerHTML);
            setContentText(event.currentTarget.textContent ?? "");
          }}
        />
        <Button type="submit" disabled={!canPublish}>{create.isPending ? "Menerbitkan…" : "Terbitkan"}</Button>
        {create.isError && <p role="alert" className="text-red-700">Feed belum dapat diterbitkan. {create.error?.message}</p>}
        {successMessage && <p role="status" className="text-emerald-700">{successMessage}</p>}
      </form>
      <CompanyFeedList />
    </section>
  );
};
