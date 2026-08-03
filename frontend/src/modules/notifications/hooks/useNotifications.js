import { keepPreviousData, useMutation, useQuery } from "@tanstack/react-query";

import { queryClient } from "../../../lib/query/query-client";
import {
  dismissNotificationRequest,
  markNotificationReadRequest,
  notificationsRequest,
  unreadCountRequest,
} from "../api/notification-api";

/**
 * Query key selalu memuat user ID. Logout sudah membersihkan seluruh cache, tetapi scoping
 * eksplisit memastikan inbox satu pengguna tidak pernah terlihat oleh pengguna berikutnya
 * bila urutan pembersihan berubah.
 */
export const notificationKeys = {
  all: ["notifications"],
  list: (userId, params) => ["notifications", userId, "list", params],
  unreadCount: (userId) => ["notifications", userId, "unread-count"],
};

export const useUnreadNotificationCount = (userId, enabled = true) =>
  useQuery({
    queryKey: notificationKeys.unreadCount(userId),
    queryFn: ({ signal }) => unreadCountRequest(signal),
    enabled: enabled && Boolean(userId),
    // Badge menyegarkan diri ketika pengguna kembali ke tab. Polling berinterval tidak
    // diaktifkan karena belum ada keputusan produk untuk kanal real-time.
    refetchOnWindowFocus: true,
    staleTime: 60_000,
  });

export const useNotifications = (userId, params, enabled = true) =>
  useQuery({
    queryKey: notificationKeys.list(userId, params),
    queryFn: ({ signal }) => notificationsRequest(params, signal),
    enabled: enabled && Boolean(userId),
    placeholderData: keepPreviousData,
    refetchOnWindowFocus: true,
  });

/**
 * Mark read dan dismiss tidak pernah optimistic. Server adalah sumber kebenaran untuk
 * `is_read` dan `dismissed_at`; menulis hasil lebih dulu di client berisiko menampilkan
 * status yang tidak jadi tersimpan, dan tidak ada endpoint restore untuk memulihkannya.
 *
 * Invalidasi mencakup list dan unread count sekaligus sehingga badge dan daftar tidak pernah
 * saling bertentangan setelah mutation.
 */
const invalidateNotifications = async () => {
  await queryClient.invalidateQueries({ queryKey: notificationKeys.all });
};

export const useMarkNotificationRead = () =>
  useMutation({
    mutationFn: (id) => markNotificationReadRequest(id),
    retry: false,
    onSettled: invalidateNotifications,
  });

export const useDismissNotification = () =>
  useMutation({
    mutationFn: (id) => dismissNotificationRequest(id),
    retry: false,
    onSettled: invalidateNotifications,
  });
