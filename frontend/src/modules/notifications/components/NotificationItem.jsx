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
          ? "border-slate-900/10 bg-slate-50"
          : "border-cyan-300/40 bg-cyan-50"
      }`}
    >
      <div className="flex flex-wrap items-start gap-3">
        <span aria-hidden="true" className="text-xl leading-none">
          {icon}
        </span>
        <div className="min-w-0 flex-1">
          <p className="text-xs font-semibold uppercase tracking-wider text-cyan-800">{label}</p>
          <p className="mt-1 font-semibold text-slate-900">{notification.judul}</p>
          <p className="mt-1 break-words text-sm text-slate-600">{notification.pesan}</p>
          <p className="mt-2 text-xs text-slate-500">
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
            className="inline-flex min-h-10 items-center rounded-lg bg-cyan-700 px-4 text-sm font-semibold text-white hover:bg-cyan-800 focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-700"
          >
            Buka detail
          </Link>
        ) : (
          <p className="inline-flex min-h-10 items-center text-sm text-slate-500">
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
