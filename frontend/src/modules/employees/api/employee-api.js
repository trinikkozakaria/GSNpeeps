import { apiClient } from "../../../lib/api/client";
import {
  departmentListSchema,
  employeeDetailSchema,
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
