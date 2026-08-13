import { apiClient } from "../../../lib/api/client";
import { requestStateSchema } from "../../leave/schemas/leave-schema";
import {
  overtimeDetailSchema,
  overtimeListSchema,
  overtimeRecapListSchema,
} from "../schemas/overtime-schema";

/** Field multipart mengikuti CreateOvertimeRequest; dokumen bersifat opsional. */
export const createOvertimeRequest = async (input, signal) => {
  const form = new FormData();
  form.append("tanggal", input.tanggal);
  form.append("waktu_mulai", input.waktu_mulai);
  form.append("waktu_selesai", input.waktu_selesai);
  form.append("alasan", input.alasan);
  if (input.dokumen_pendukung) form.append("dokumen_pendukung", input.dokumen_pendukung);
  const envelope = await apiClient.post("/lembur", form, { signal });
  return requestStateSchema.parse(envelope.data);
};

export const overtimeListRequest = async (params, signal) => {
  const envelope = await apiClient.get("/lembur", { params, signal });
  return overtimeListSchema.parse({ items: envelope.data, meta: envelope.meta });
};

export const myOvertimeRequestsRequest = async (params, signal) => {
  const envelope = await apiClient.get("/lembur/saya", { params, signal });
  return overtimeListSchema.parse({ items: envelope.data, meta: envelope.meta });
};

export const overtimeDetailRequest = async (id, signal) => {
  const envelope = await apiClient.get(`/lembur/${id}`, { signal });
  return overtimeDetailSchema.parse(envelope.data);
};

export const decideOvertimeRequest = async (id, payload, signal) => {
  const envelope = await apiClient.put(`/lembur/${id}/decision`, payload, { signal });
  return requestStateSchema.parse(envelope.data);
};

export const overtimeRecapRequest = async (params, signal) => {
  const envelope = await apiClient.get("/lembur/rekap", { params, signal });
  return overtimeRecapListSchema.parse(envelope.data);
};
