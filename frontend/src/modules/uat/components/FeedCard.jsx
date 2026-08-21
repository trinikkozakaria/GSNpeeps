import { formatDateTime } from "../../../lib/format";
import { ProtectedDocumentPreview } from "../../../components/media/ProtectedImage";

/** Kartu tampilan satu company feed, dipakai baik oleh Beranda maupun halaman Company Feed. */
export const FeedCard = ({ feed, actions }) => (
  <article className="rounded-xl border border-slate-900/10 p-5">
    <div className="flex flex-wrap items-start justify-between gap-3">
      <div className="min-w-0">
        <h3 className="text-xl font-bold">{feed.judul}</h3>
        <p className="mt-1 text-xs text-slate-600">{feed.penulis} · {formatDateTime(feed.published_at)}</p>
      </div>
      {actions && <div className="flex shrink-0 flex-wrap gap-2">{actions}</div>}
    </div>
    <div className="wysiwyg-content mt-4 max-w-none" dangerouslySetInnerHTML={{ __html: feed.konten_html }} />
    {feed.attachments?.length > 0 && (
      <section className="mt-5 border-t border-slate-900/10 pt-4" aria-label={`Attachment ${feed.judul}`}>
        <p className="mb-3 text-sm font-semibold text-slate-700">Attachment</p>
        <div className="grid gap-4 sm:grid-cols-2">
          {feed.attachments.map((attachment) => (
            <ProtectedDocumentPreview
              key={attachment.id}
              path={attachment.file_url}
              fileName={attachment.file_name}
            />
          ))}
        </div>
      </section>
    )}
  </article>
);
