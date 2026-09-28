import { apiClient } from "../../../lib/api/client";
import {
  approvalRequestDetailSchema,
  approvalRequestListSchema,
  approvalRequestSchema,
  workflowTemplateDetailSchema,
  workflowTemplateListSchema,
} from "../schemas/document-approval-schema";

// --- Templates -------------------------------------------------------------------------

export const workflowTemplatesRequest = async (params, signal) => {
  const envelope = await apiClient.get("/master/alur-persetujuan-dokumen", { params, signal });
  return workflowTemplateListSchema.parse(envelope.data);
};

export const workflowTemplateDetailRequest = async (id, signal) => {
  const envelope = await apiClient.get(`/master/alur-persetujuan-dokumen/${id}`, { signal });
  return workflowTemplateDetailSchema.parse(envelope.data);
};

export const createWorkflowTemplateRequest = async (payload, signal) => {
  const envelope = await apiClient.post("/master/alur-persetujuan-dokumen", payload, { signal });
  return envelope.data;
};

export const updateWorkflowTemplateRequest = async (id, payload, signal) => {
  const envelope = await apiClient.put(`/master/alur-persetujuan-dokumen/${id}`, payload, { signal });
  return envelope.data;
};

// --- Approval requests -------------------------------------------------------------------

export const createApprovalRequest = async (payload, signal) => {
  const envelope = await apiClient.post("/persetujuan-dokumen", payload, { signal });
  return approvalRequestSchema.parse(envelope.data);
};

export const myApprovalRequestsRequest = async (params, signal) => {
  const envelope = await apiClient.get("/persetujuan-dokumen/saya", { params, signal });
  return approvalRequestListSchema.parse({ items: envelope.data, meta: envelope.meta });
};

export const approvalInboxRequest = async (params, signal) => {
  const envelope = await apiClient.get("/persetujuan-dokumen", { params, signal });
  return approvalRequestListSchema.parse({ items: envelope.data, meta: envelope.meta });
};

export const approvalMonitoringRequest = async (params, signal) => {
  const envelope = await apiClient.get("/persetujuan-dokumen/monitoring", { params, signal });
  return approvalRequestListSchema.parse({ items: envelope.data, meta: envelope.meta });
};

export const approvalRequestDetailRequest = async (id, signal) => {
  const envelope = await apiClient.get(`/persetujuan-dokumen/${id}`, { signal });
  return approvalRequestDetailSchema.parse(envelope.data);
};

export const decideApprovalRequest = async (id, payload, signal) => {
  const envelope = await apiClient.put(`/persetujuan-dokumen/${id}/decision`, payload, { signal });
  return approvalRequestSchema.parse(envelope.data);
};

export const cancelApprovalRequest = async (id, signal) => {
  const envelope = await apiClient.put(`/persetujuan-dokumen/${id}/batalkan`, {}, { signal });
  return approvalRequestSchema.parse(envelope.data);
};
