import { Link } from "react-router-dom";

import { Button } from "../../../components/ui/Button";
import { formatDateTime } from "../../../lib/format";
import {
  notificationPresentation,
  notificationTargetForRole,
} from "../utils/notification-presentation";

/**
 * Satu baris notifikasi.
 *
 * `judul` dan `pesan` selalu dirender sebagai teks React. Tidak ada jalur `dangerouslySetInnerHTML`,
 * sehingga payload berisi markup tampil apa adanya dan tidak pernah dieksekusi.
 */
export const NotificationItem = ({ notification, role, onOpen, onMarkRead, onDismiss, busy }) => {
  const { icon, label } = notificationPresentation(notification.tipe);
  const target = notificationTargetForRole(notification, role);

  return (
    <li
      className={`rounded-xl border p-4 ${
        notification.is_read
          ? "border-white/10 bg-slate-900/40"
          : "border-cyan-300/40 bg-cyan-300/[0.06]"
      }`}
    >
      <div className="flex flex-wrap items-start gap-3">
        <span aria-hidden="true" className="text-xl leading-none">
          {icon}
        </span>
        <div className="min-w-0 flex-1">
          <p className="text-xs font-semibold uppercase tracking-wider text-cyan-200">{label}</p>
          <p className="mt-1 font-semibold text-white">{notification.judul}</p>
          <p className="mt-1 break-words text-sm text-slate-300">{notification.pesan}</p>
          <p className="mt-2 text-xs text-slate-400">
            <time dateTime={notification.created_at}>{formatDateTime(notification.created_at)}</time>
            {/* Status dibaca disampaikan sebagai teks, bukan hanya melalui warna latar. */}
            {notification.is_read ? " · Sudah dibaca" : " · Belum dibaca"}
          </p>
        </div>
      </div>

      <div className="mt-4 flex flex-wrap gap-2">
        {target ? (
          <Link
            to={target}
            onClick={() => onOpen(notification)}
            className="inline-flex min-h-11 items-center rounded-lg bg-cyan-300 px-4 text-sm font-semibold text-slate-950 hover:bg-cyan-200 focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-300"
          >
            Buka detail
          </Link>
        ) : (
          <p className="inline-flex min-h-11 items-center text-sm text-slate-400">
            Tidak ada halaman detail untuk notifikasi ini.
          </p>
        )}
        {!notification.is_read && (
          <Button variant="secondary" disabled={busy} onClick={() => onMarkRead(notification)}>
            Tandai dibaca
          </Button>
        )}
        <Button variant="secondary" disabled={busy} onClick={() => onDismiss(notification)}>
          Hapus
        </Button>
      </div>
    </li>
  );
};
