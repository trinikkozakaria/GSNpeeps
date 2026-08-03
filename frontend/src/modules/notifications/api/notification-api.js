import { apiClient } from "../../../lib/api/client";
import {
  notificationListSchema,
  unreadCountSchema,
} from "../schemas/notification-schema";

export const notificationsRequest = async (params, signal) => {
  const envelope = await apiClient.get("/notifikasi", { params, signal });
  return notificationListSchema.parse({ items: envelope.data, meta: envelope.meta });
};

export const unreadCountRequest = async (signal) => {
  const envelope = await apiClient.get("/notifikasi/unread-count", { signal });
  return unreadCountSchema.parse(envelope.data).unread_count;
};

export const markNotificationReadRequest = async (id, signal) => {
  const envelope = await apiClient.put(`/notifikasi/${id}/read`, undefined, { signal });
  return envelope.data;
};

// Dismiss adalah soft-delete di server; tidak ada operasi restore pada kontrak.
export const dismissNotificationRequest = async (id, signal) => {
  await apiClient.delete(`/notifikasi/${id}`, { signal });
  return id;
};
