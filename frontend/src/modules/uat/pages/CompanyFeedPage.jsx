import { useMutation, useQuery } from "@tanstack/react-query";
import { useRef, useState } from "react";

import { Button } from "../../../components/ui/Button";
import { formatDateTime } from "../../../lib/format";
import { queryClient } from "../../../lib/query/query-client";
import { createFeedRequest, feedsRequest } from "../api/uat-api";

export const CompanyFeedList = () => {
  const feeds = useQuery({ queryKey: ["company-feed"], queryFn: ({ signal }) => feedsRequest(signal) });
  if (feeds.isPending) return <p role="status">Memuat company feed…</p>;
  if (feeds.isError) return <p role="alert">Company feed belum dapat dimuat.</p>;
  if (feeds.data.length === 0) return <p className="text-slate-500">Belum ada informasi perusahaan.</p>;
  return <div className="grid gap-4">{feeds.data.map((feed) => <article key={feed.id} className="rounded-xl border border-slate-900/10 p-5"><h3 className="text-xl font-bold">{feed.judul}</h3><p className="mt-1 text-xs text-slate-500">{feed.penulis} · {formatDateTime(feed.published_at)}</p><div className="prose mt-4 max-w-none" dangerouslySetInnerHTML={{ __html: feed.konten_html }} /></article>)}</div>;
};

const tools = [["bold", "Tebal"], ["italic", "Miring"], ["underline", "Garis bawah"], ["insertUnorderedList", "Daftar"], ["insertOrderedList", "Nomor"]];

export const CompanyFeedPage = () => {
  const editorRef = useRef(null);
  const [title, setTitle] = useState("");
  const [html, setHTML] = useState("");
  const create = useMutation({ mutationFn: createFeedRequest, onSuccess: async () => { setTitle(""); setHTML(""); if (editorRef.current) editorRef.current.innerHTML = ""; await queryClient.invalidateQueries({ queryKey: ["company-feed"] }); } });
  const command = (name) => { editorRef.current?.focus(); document.execCommand(name, false); setHTML(editorRef.current?.innerHTML ?? ""); };
  return <section><p className="text-sm font-semibold uppercase tracking-widest text-cyan-700">Komunikasi</p><h1 className="mt-2 text-3xl font-bold">Company Feed</h1><p className="mt-2 text-slate-600">Informasi yang diterbitkan tampil di beranda seluruh karyawan.</p><form className="my-6 grid gap-4 rounded-xl border border-slate-900/10 p-5" onSubmit={(event) => { event.preventDefault(); create.mutate({ judul: title, konten_html: html }); }}><label className="text-sm font-medium">Judul<input className="mt-2 min-h-10 w-full rounded-lg border px-3" value={title} onChange={(event) => setTitle(event.target.value)} required /></label><span className="text-sm font-medium">Konten</span><div className="flex flex-wrap gap-2" role="toolbar" aria-label="Format konten">{tools.map(([name, label]) => <Button key={name} variant="secondary" onMouseDown={(event) => event.preventDefault()} onClick={() => command(name)}>{label}</Button>)}</div><div ref={editorRef} role="textbox" aria-label="Konten company feed" aria-multiline="true" contentEditable suppressContentEditableWarning className="min-h-40 rounded-lg border p-3 focus:outline-cyan-700" onInput={(event) => setHTML(event.currentTarget.innerHTML)} /><Button type="submit" disabled={create.isPending || !title.trim() || !editorRef.current?.textContent?.trim()}>{create.isPending ? "Menerbitkan…" : "Terbitkan"}</Button>{create.isError && <p role="alert" className="text-red-700">Feed belum dapat diterbitkan. {create.error.message}</p>}</form><CompanyFeedList /></section>;
};
