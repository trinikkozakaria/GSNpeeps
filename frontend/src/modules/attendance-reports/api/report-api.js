import { apiClient } from "../../../lib/api/client";
import { downloadFile } from "../../../lib/api/download";
import { attendanceReportSchema } from "../../attendance/schemas/attendance-schema";

export const attendanceReportRequest = async (params, signal) => {
  const envelope = await apiClient.get("/laporan/kehadiran", { params, signal });
  return attendanceReportSchema.parse({ items: envelope.data, meta: envelope.meta });
};

export const exportAttendanceReportRequest = (query, signal) =>
  downloadFile("/laporan/kehadiran/export", query, {
    signal,
    fallbackFileName: `laporan-kehadiran.${query.format ?? "xlsx"}`,
  });

export const exportLiveFeedRequest = (tanggal, signal) =>
  downloadFile("/absensi/livefeed/export", { tanggal }, {
    signal,
    fallbackFileName: `live-feed-absensi-${tanggal || "hari-ini"}.xlsx`,
  });
