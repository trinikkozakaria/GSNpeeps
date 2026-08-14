import { apiClient } from "../../../lib/api/client";

const data = (envelope) => envelope.data;

export const attendanceCorrectionsRequest = async (signal) =>
  data(await apiClient.get("/absensi/koreksi", { signal }));

export const createAttendanceCorrectionRequest = async (payload) =>
  data(await apiClient.post("/absensi/koreksi", payload));

export const decideAttendanceCorrectionRequest = async ({ id, keputusan, catatan = null }) =>
  data(await apiClient.put(`/absensi/koreksi/${id}`, { keputusan, catatan }));
