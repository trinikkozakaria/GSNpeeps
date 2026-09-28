import { keepPreviousData, useMutation, useQuery } from "@tanstack/react-query";

import { queryClient } from "../../../lib/query/query-client";
import {
  approvalInboxRequest,
  approvalMonitoringRequest,
  approvalRequestDetailRequest,
  cancelApprovalRequest,
  createApprovalRequest,
  createWorkflowTemplateRequest,
  decideApprovalRequest,
  myApprovalRequestsRequest,
  updateWorkflowTemplateRequest,
  workflowTemplateDetailRequest,
  workflowTemplatesRequest,
} from "../api/document-approval-api";

export const documentApprovalKeys = {
  all: ["document-approval"],
  templates: (params) => ["document-approval", "templates", params ?? "all"],
  template: (id) => ["document-approval", "template", id],
  mine: (params) => ["document-approval", "mine", params],
  inbox: (params) => ["document-approval", "inbox", params],
  monitoring: (params) => ["document-approval", "monitoring", params],
  detail: (id) => ["document-approval", "detail", id],
};

// --- Templates -------------------------------------------------------------------------

export const useWorkflowTemplates = (params, enabled = true) =>
  useQuery({
    queryKey: documentApprovalKeys.templates(params),
    queryFn: ({ signal }) => workflowTemplatesRequest(params, signal),
    enabled,
    staleTime: 5 * 60 * 1000,
  });

export const useWorkflowTemplateDetail = (id) =>
  useQuery({
    queryKey: documentApprovalKeys.template(id),
    queryFn: ({ signal }) => workflowTemplateDetailRequest(id, signal),
    enabled: Boolean(id),
  });

export const useCreateWorkflowTemplate = () =>
  useMutation({
    mutationFn: (payload) => createWorkflowTemplateRequest(payload),
    retry: false,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: documentApprovalKeys.all });
    },
  });

export const useUpdateWorkflowTemplate = () =>
  useMutation({
    mutationFn: ({ id, payload }) => updateWorkflowTemplateRequest(id, payload),
    retry: false,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: documentApprovalKeys.all });
    },
  });

// --- Approval requests -------------------------------------------------------------------

export const useCreateApprovalRequest = () =>
  useMutation({
    mutationFn: (payload) => createApprovalRequest(payload),
    retry: false,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: documentApprovalKeys.all });
    },
  });

export const useMyApprovalRequests = (params) =>
  useQuery({
    queryKey: documentApprovalKeys.mine(params),
    queryFn: ({ signal }) => myApprovalRequestsRequest(params, signal),
    placeholderData: keepPreviousData,
  });

export const useApprovalInbox = (params, enabled = true) =>
  useQuery({
    queryKey: documentApprovalKeys.inbox(params),
    queryFn: ({ signal }) => approvalInboxRequest(params, signal),
    enabled,
    placeholderData: keepPreviousData,
  });

export const useApprovalMonitoring = (params, enabled = true) =>
  useQuery({
    queryKey: documentApprovalKeys.monitoring(params),
    queryFn: ({ signal }) => approvalMonitoringRequest(params, signal),
    enabled,
    placeholderData: keepPreviousData,
  });

export const useApprovalRequestDetail = (id) =>
  useQuery({
    queryKey: documentApprovalKeys.detail(id),
    queryFn: ({ signal }) => approvalRequestDetailRequest(id, signal),
    enabled: Boolean(id),
  });

// Keputusan tidak optimistic — sama seperti leave, cache di-invalidate setelah server
// mengonfirmasi.
export const useDecideApprovalRequest = (id) =>
  useMutation({
    mutationFn: (payload) => decideApprovalRequest(id, payload),
    retry: false,
    onSettled: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: documentApprovalKeys.all }),
        queryClient.invalidateQueries({ queryKey: ["notifications"] }),
      ]);
    },
  });

export const useCancelApprovalRequest = (id) =>
  useMutation({
    mutationFn: () => cancelApprovalRequest(id),
    retry: false,
    onSettled: async () => {
      await queryClient.invalidateQueries({ queryKey: documentApprovalKeys.all });
    },
  });
