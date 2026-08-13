import { apiClient } from "../../../lib/api/client";
import {
  attendanceSchema,
  liveFeedListSchema,
  officeLocationListSchema,
} from "../schemas/attendance-schema";

export const officeLocationsRequest = async (signal) => {
  const envelope = await apiClient.get("/master/lokasi-kantor", { signal });
  return officeLocationListSchema.parse(envelope.data);
};

/**
 * Field multipart mengikuti AttendanceCheckRequest. `office_location_id` hanya dikirim untuk
 * WFO; WFH dan WFA tetap mengirim koordinat tanpa lokasi kantor.
 */
export const recordAttendanceRequest = async (input, signal) => {
  const form = new FormData();
  form.append("tipe", input.tipe);
  form.append("mode_kerja", input.mode_kerja);
  form.append("gps_lat", String(input.gps_lat));
  form.append("gps_long", String(input.gps_long));
  if (input.office_location_id) {
    form.append("office_location_id", input.office_location_id);
  }
  form.append("foto", input.foto);
  const envelope = await apiClient.post("/absensi/checkin", form, { signal });
  return attendanceSchema.parse(envelope.data);
};

export const liveFeedRequest = async (tanggal, signal) => {
  const envelope = await apiClient.get("/absensi/livefeed", {
    params: tanggal ? { tanggal } : undefined,
    signal,
  });
  return liveFeedListSchema.parse(envelope.data);
};
