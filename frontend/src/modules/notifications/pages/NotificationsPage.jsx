import { useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";

import { ConfirmDialog } from "../../../components/feedback/ConfirmDialog";
import { Pagination } from "../../../components/data-table/Pagination";
import { Button } from "../../../components/ui/Button";
import { useAuth } from "../../auth/hooks/useAuth";
import { NotificationItem } from "../components/NotificationItem";
import {
  useDismissNotification,
  useMarkNotificationRead,
  useNotifications,
} from "../hooks/useNotifications";

const filterOptions = [
  { value: "semua", label: "Semua" },
  { value: "belum", label: "Belum dibaca" },
  { value: "sudah", label: "Sudah dibaca" },
];

const isReadParam = { semua: undefined, belum: false, sudah: true };

export const NotificationsPage = () => {
  document.title = "Notifikasi — GSNpeeps";
  const auth = useAuth();
  const [params, setParams] = useSearchParams();
  const [pendingDismiss, setPendingDismiss] = useState(null);
  // Kegagalan dipisah menurut tempat munculnya: aksi pada baris dilaporkan di atas daftar,
  // kegagalan dismiss dilaporkan di dalam dialog yang tetap terbuka. Satu state bersama akan
  // menampilkan pesan yang sama dua kali.
  const [rowError, setRowError] = useState("");
  const [dismissError, setDismissError] = useState("");

  const filter = filterOptions.some((option) => option.value === params.get("status"))
    ? params.get("status")
    : "semua";

  const query = useMemo(
    () => ({
      page: Number.parseInt(params.get("page") ?? "1", 10) || 1,
      limit: 10,
      is_read: isReadParam[filter],
    }),
    [filter, params],
  );

  const notifications = useNotifications(auth.user?.id, query);
  const markRead = useMarkNotificationRead();
  const dismiss = useDismissNotification();
  const busy = markRead.isPending || dismiss.isPending;

  const setParam = (key, value) => {
    const next = new URLSearchParams(params);
    if (value) next.set(key, value);
    else next.delete(key);
    if (key !== "page") next.delete("page");
    setParams(next);
  };

  const handleMarkRead = async (notification) => {
    setRowError("");
    try {
      await markRead.mutateAsync(notification.id);
    } catch (error) {
      setRowError(describeNotificationError(error, "Notifikasi gagal ditandai dibaca."));
    }
  };

  // Membuka detail sekaligus menandai dibaca; kegagalan penandaan tidak boleh menghalangi
  // navigasi karena pengguna sudah melihat isinya.
  const handleOpen = (notification) => {
    if (notification.is_read) return;
    markRead.mutate(notification.id);
  };

  const confirmDismiss = async () => {
    if (!pendingDismiss) return;
    setDismissError("");
    try {
      await dismiss.mutateAsync(pendingDismiss.id);
      setPendingDismiss(null);
    } catch (error) {
      setDismissError(describeNotificationError(error, "Notifikasi gagal dihapus."));
    }
  };

  return (
    <section aria-labelledby="notifications-title">
      <p className="text-sm font-semibold uppercase tracking-widest text-cyan-700">Aktivitas</p>
      <h1 id="notifications-title" className="mt-2 text-3xl font-bold">
        Notifikasi
      </h1>
      <p className="mt-2 text-slate-600">
        Hanya notifikasi milik akun Anda yang ditampilkan. Notifikasi yang dihapus tidak dapat
        dikembalikan.
      </p>

      <div className="mt-6 flex flex-wrap gap-2" role="group" aria-label="Filter status notifikasi">
        {filterOptions.map((option) => (
          <Button
            key={option.value}
            variant={filter === option.value ? "primary" : "secondary"}
            aria-pressed={filter === option.value}
            onClick={() => setParam("status", option.value === "semua" ? "" : option.value)}
          >
            {option.label}
          </Button>
        ))}
      </div>

      {rowError && (
        <p
          role="alert"
          className="mt-4 rounded-lg border border-rose-300/30 bg-rose-300/10 p-3 text-sm text-rose-700"
        >
          {rowError}
        </p>
      )}

      <div className="mt-6" aria-live="polite">
        {notifications.isPending && (
          <p role="status" className="text-slate-600">
            Memuat notifikasi…
          </p>
        )}

        {notifications.isError && (
          <div
            role="alert"
            className="rounded-xl border border-red-400/30 bg-red-400/10 p-4 text-red-700"
          >
            <p>Notifikasi belum dapat dimuat. {notifications.error?.message}</p>
            <Button className="mt-3" variant="secondary" onClick={() => notifications.refetch()}>
              Coba lagi
            </Button>
          </div>
        )}

        {notifications.data && notifications.data.items.length === 0 && (
          <div className="rounded-xl border border-slate-900/10 p-8 text-center text-slate-600">
            {filter === "belum"
              ? "Tidak ada notifikasi yang belum dibaca."
              : "Belum ada notifikasi untuk akun Anda."}
          </div>
        )}

        {notifications.data && notifications.data.items.length > 0 && (
          <>
            <ul className="grid gap-3" aria-label="Daftar notifikasi">
              {notifications.data.items.map((notification) => (
                <NotificationItem
                  key={notification.id}
                  notification={notification}
                  role={auth.role}
                  busy={busy}
                  onOpen={handleOpen}
                  onMarkRead={handleMarkRead}
                  onDismiss={setPendingDismiss}
                />
              ))}
            </ul>
            <Pagination
              meta={notifications.data.meta}
              label="Navigasi halaman notifikasi"
              onPageChange={(page) => setParam("page", String(page))}
            />
          </>
        )}
      </div>

      <ConfirmDialog
        open={Boolean(pendingDismiss)}
        title="Hapus notifikasi ini?"
        description="Notifikasi akan hilang dari daftar Anda dan tidak dapat dikembalikan."
        confirmLabel="Hapus"
        destructive
        busy={dismiss.isPending}
        error={dismissError}
        onConfirm={confirmDismiss}
        onCancel={() => {
          setPendingDismiss(null);
          setDismissError("");
        }}
      />
    </section>
  );
};

const describeNotificationError = (error, fallback) => {
  // Server memakai 403 seragam untuk notifikasi yang tidak ada maupun milik orang lain,
  // sehingga UI tidak dapat dan tidak perlu membedakan keduanya.
  if (error?.status === 403) return "Notifikasi ini tidak dapat diakses lagi.";
  return error?.message ?? fallback;
};
