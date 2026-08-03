import { Link } from "react-router-dom";

import { useAuth } from "../../auth/hooks/useAuth";
import { useUnreadNotificationCount } from "../hooks/useNotifications";

const maximumBadgeCount = 99;

/**
 * Lonceng notifikasi pada header aplikasi.
 *
 * Permintaan hanya berjalan setelah sesi terautentikasi, sehingga tidak ada request inbox
 * yang terkirim saat halaman publik atau saat identitas belum selesai dipulihkan. Selama
 * memuat, badge tidak menampilkan angka apa pun; angka nol palsu akan sama menyesatkannya
 * dengan angka acak.
 */
export const NotificationBell = () => {
  const auth = useAuth();
  const isAuthenticated = auth.status === "authenticated";
  const unread = useUnreadNotificationCount(auth.user?.id, isAuthenticated);

  const count = unread.data ?? 0;
  const hasUnread = unread.isSuccess && count > 0;
  const badgeText = count > maximumBadgeCount ? `${maximumBadgeCount}+` : String(count);

  const label = () => {
    if (unread.isPending) return "Notifikasi, jumlah belum dibaca sedang dimuat";
    if (unread.isError) return "Notifikasi, jumlah belum dibaca tidak dapat dimuat";
    if (!hasUnread) return "Notifikasi, tidak ada yang belum dibaca";
    return `Notifikasi, ${count} belum dibaca`;
  };

  return (
    <Link
      to="/app/notifikasi"
      aria-label={label()}
      className="relative inline-flex min-h-11 min-w-11 items-center justify-center rounded-lg border border-white/15 text-lg text-slate-200 hover:bg-white/5 hover:text-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-300"
    >
      <span aria-hidden="true">🔔</span>
      {hasUnread && (
        <span
          aria-hidden="true"
          className="absolute -right-1.5 -top-1.5 min-w-5 rounded-full bg-cyan-300 px-1.5 text-center text-xs font-bold leading-5 text-slate-950"
        >
          {badgeText}
        </span>
      )}
      {/* Perubahan jumlah diumumkan sekali tanpa membacakan seluruh isi lonceng. */}
      <span className="sr-only" role="status" aria-live="polite">
        {hasUnread ? `${count} notifikasi belum dibaca` : ""}
      </span>
    </Link>
  );
};
