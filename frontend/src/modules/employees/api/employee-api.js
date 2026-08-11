import { apiClient } from "../../../lib/api/client";
import { downloadFile } from "../../../lib/api/download";
import {
  departmentListSchema,
  employeeDetailSchema,
  employeeDocumentListSchema,
  employeeListSchema,
  positionListSchema,
} from "../schemas/employee-schema";

export const listDepartmentsRequest = async (signal) => {
  const envelope = await apiClient.get("/master/departemen", { signal });
  return departmentListSchema.parse(envelope.data);
};

export const listPositionsRequest = async (departmentId, signal) => {
  const envelope = await apiClient.get("/master/jabatan", {
    params: departmentId ? { department_id: departmentId } : undefined,
    signal,
  });
  return positionListSchema.parse(envelope.data);
};

export const listEmployeesRequest = async (filters, signal) => {
  const envelope = await apiClient.get("/karyawan", {
    params: filters,
    signal,
  });
  return employeeListSchema.parse({
    items: envelope.data,
    meta: envelope.meta,
  });
};

export const employeeDetailRequest = async (id, signal) => {
  const envelope = await apiClient.get(`/karyawan/${id}`, { signal });
  return employeeDetailSchema.parse(envelope.data);
};

export const updateEmployeeRequest = async (id, payload, signal) => {
  const envelope = await apiClient.put(`/karyawan/${id}`, payload, { signal });
  return envelope.data;
};

export const deactivateEmployeeRequest = async (id, signal) => {
  const envelope = await apiClient.delete(`/karyawan/${id}`, { signal });
  return envelope.data;
};

export const createEmployeeRequest = async (payload, signal) => {
  const envelope = await apiClient.post("/karyawan", payload, { signal });
  return envelope.data;
};

export const employeeDocumentsRequest = async (id, signal) => {
  const envelope = await apiClient.get(`/karyawan/${id}/dokumen`, { signal });
  return employeeDocumentListSchema.parse(envelope.data);
};

/**
 * Field multipart mengikuti EmployeeDocumentUploadRequest: `jenis_dokumen` dan `file`.
 * Isi berkas tidak pernah disimpan ke state global; hanya diteruskan ke request.
 */
export const uploadEmployeeDocumentRequest = async (id, { jenisDokumen, file }, signal) => {
  const form = new FormData();
  form.append("jenis_dokumen", jenisDokumen);
  form.append("file", file);
  const envelope = await apiClient.post(`/karyawan/${id}/dokumen`, form, { signal });
  return envelope.data;
};

/** HR memperbarui foto profil karyawan lain. Diri sendiri memakai PUT /auth/me/foto. */
export const uploadEmployeePhotoRequest = async (id, file, signal) => {
  const form = new FormData();
  form.append("foto", file);
  const envelope = await apiClient.put(`/karyawan/${id}/foto`, form, { signal });
  return envelope.data;
};

export const exportEmployeesRequest = (query, signal) =>
  downloadFile("/karyawan/export", query, {
    signal,
    fallbackFileName: `karyawan.${query.format ?? "xlsx"}`,
  });
