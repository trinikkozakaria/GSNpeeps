import { apiClient } from "../../../lib/api/client";
import {
  leaveRequestDetailSchema,
  leaveRequestListSchema,
  leaveTypeListSchema,
  requestStateSchema,
} from "../schemas/leave-schema";

export const leaveTypesRequest = async (params, signal) => {
  const envelope = await apiClient.get("/master/jenis-izin", { params, signal });
  return leaveTypeListSchema.parse(envelope.data);
};

export const createLeaveTypeRequest = async (payload, signal) => {
  const envelope = await apiClient.post("/master/jenis-izin", payload, { signal });
  return envelope.data;
};

export const updateLeaveTypeRequest = async (id, payload, signal) => {
  const envelope = await apiClient.put(`/master/jenis-izin/${id}`, payload, { signal });
  return envelope.data;
};

/** Field multipart mengikuti CreateLeaveRequest pada OpenAPI. */
export const createLeaveRequest = async (input, signal) => {
  const form = new FormData();
  form.append("jenis_izin_id", input.jenis_izin_id);
  form.append("tanggal_mulai", input.tanggal_mulai);
  form.append("tanggal_selesai", input.tanggal_selesai);
  form.append("alasan", input.alasan);
  if (input.lokasi_tujuan) form.append("lokasi_tujuan", input.lokasi_tujuan);
  if (input.keterangan_lokasi) form.append("keterangan_lokasi", input.keterangan_lokasi);
  if (input.dokumen_pendukung) form.append("dokumen_pendukung", input.dokumen_pendukung);
  const envelope = await apiClient.post("/ketidakhadiran", form, { signal });
  return requestStateSchema.parse(envelope.data);
};

export const myLeaveRequestsRequest = async (params, signal) => {
  const envelope = await apiClient.get("/ketidakhadiran/saya", { params, signal });
  return leaveRequestListSchema.parse({ items: envelope.data, meta: envelope.meta });
};

export const leaveApprovalInboxRequest = async (params, signal) => {
  const envelope = await apiClient.get("/ketidakhadiran", { params, signal });
  return leaveRequestListSchema.parse({ items: envelope.data, meta: envelope.meta });
};

export const leaveRequestDetailRequest = async (id, signal) => {
  const envelope = await apiClient.get(`/ketidakhadiran/${id}`, { signal });
  return leaveRequestDetailSchema.parse(envelope.data);
};

export const decideLeaveRequest = async (id, payload, signal) => {
  const envelope = await apiClient.put(`/ketidakhadiran/${id}/decision`, payload, { signal });
  return requestStateSchema.parse(envelope.data);
};

export const delegateLeaveRequest = async (id, payload, signal) => {
  const envelope = await apiClient.put(`/ketidakhadiran/${id}/delegate`, payload, { signal });
  return requestStateSchema.parse(envelope.data);
};
