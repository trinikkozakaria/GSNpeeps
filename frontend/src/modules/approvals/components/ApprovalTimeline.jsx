import { formatDateTime } from "../../../lib/format";
import { approvalDecisionLabel, approvalStageLabel } from "../../../lib/request-status";

/**
 * Riwayat approval sebagai daftar terurut. Keputusan sistem tidak memiliki approver sehingga
 * ditampilkan sebagai eskalasi otomatis, bukan nama kosong.
 */
export const ApprovalTimeline = ({ history }) => {
  if (!history || history.length === 0) {
    return (
      <p className="text-sm text-slate-400">
        Belum ada keputusan. Pengajuan masih menunggu approver tahap aktif.
      </p>
    );
  }

  return (
    <ol className="relative border-l border-white/15 pl-5">
      {history.map((entry, index) => (
        <li key={`${entry.decided_at}-${index}`} className="mb-5 last:mb-0">
          <span
            aria-hidden="true"
            className="absolute -left-1.5 mt-1.5 h-3 w-3 rounded-full bg-cyan-300"
          />
          <p className="text-sm font-semibold text-slate-100">
            {approvalStageLabel[entry.tahap] ?? entry.tahap} ·{" "}
            {approvalDecisionLabel[entry.keputusan] ?? entry.keputusan}
          </p>
          <p className="mt-1 text-xs text-slate-400">
            {entry.approver_nama ? `Oleh ${entry.approver_nama}` : "Dipicu sistem"} ·{" "}
            <time dateTime={entry.decided_at}>{formatDateTime(entry.decided_at)}</time>
          </p>
          {entry.catatan && <p className="mt-2 text-sm text-slate-300">{entry.catatan}</p>}
        </li>
      ))}
    </ol>
  );
};
