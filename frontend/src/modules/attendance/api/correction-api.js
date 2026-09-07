import { apiClient } from "../../../lib/api/client";

const data = (envelope) => envelope.data;

export const attendanceCorrectionsRequest = async (params, signal) => {
  const envelope = await apiClient.get("/absensi/koreksi", { params, signal });
  return { items: envelope.data, meta: envelope.meta };
};

// Riwayat koreksi milik user yang login saja, setara GET /ketidakhadiran/saya.
export const myAttendanceCorrectionsRequest = async (params, signal) => {
  const envelope = await apiClient.get("/absensi/koreksi/saya", { params, signal });
  return { items: envelope.data, meta: envelope.meta };
};

export const createAttendanceCorrectionRequest = async (payload) =>
  data(await apiClient.post("/absensi/koreksi", payload));

export const decideAttendanceCorrectionRequest = async ({ id, keputusan, catatan = null }) =>
  data(await apiClient.put(`/absensi/koreksi/${id}`, { keputusan, catatan }));
