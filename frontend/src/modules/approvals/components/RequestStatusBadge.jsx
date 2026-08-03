import { requestStatusLabel } from "../../../lib/request-status";

// Status tidak pernah dibedakan oleh warna saja; label teks selalu menyertainya.
const tone = {
  menunggu_atasan: "bg-amber-400/15 text-amber-200",
  menunggu_hr: "bg-amber-400/15 text-amber-200",
  menunggu_top_management: "bg-amber-400/15 text-amber-200",
  disetujui: "bg-emerald-400/15 text-emerald-300",
  ditolak: "bg-rose-400/15 text-rose-200",
  dibatalkan: "bg-slate-700 text-slate-300",
};

export const RequestStatusBadge = ({ status }) => (
  <span
    className={`inline-flex rounded-full px-2.5 py-1 text-xs font-semibold ${
      tone[status] ?? "bg-slate-700 text-slate-300"
    }`}
  >
    {requestStatusLabel[status] ?? status}
  </span>
);
