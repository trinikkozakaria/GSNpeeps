import { Button } from "../ui/Button";

/**
 * Pagination server-side. Nilai halaman dan total selalu berasal dari `meta` response API,
 * tidak dihitung ulang di client.
 */
export const Pagination = ({ meta, onPageChange, label = "Navigasi halaman" }) => {
  const totalPage = Math.max(meta.total_page, 1);
  const isFirst = meta.page <= 1;
  const isLast = meta.page >= meta.total_page;

  return (
    <nav className="mt-4 flex flex-wrap items-center justify-between gap-3" aria-label={label}>
      <p className="text-sm text-slate-500">
        Halaman {meta.page} dari {totalPage} · {meta.total_data} data
      </p>
      <div className="flex gap-2">
        <Button variant="secondary" disabled={isFirst} onClick={() => onPageChange(meta.page - 1)}>
          Sebelumnya
        </Button>
        <Button variant="secondary" disabled={isLast} onClick={() => onPageChange(meta.page + 1)}>
          Berikutnya
        </Button>
      </div>
    </nav>
  );
};
