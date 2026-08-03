import { keepPreviousData, useMutation, useQuery } from "@tanstack/react-query";

import { queryClient } from "../../../lib/query/query-client";
import {
  createLeaveRequest,
  createLeaveTypeRequest,
  decideLeaveRequest,
  delegateLeaveRequest,
  leaveApprovalInboxRequest,
  leaveRequestDetailRequest,
  leaveTypesRequest,
  myLeaveRequestsRequest,
  updateLeaveTypeRequest,
} from "../api/leave-api";

export const leaveKeys = {
  all: ["leave"],
  types: (params) => ["leave", "types", params ?? "all"],
  mine: (scope, params) => ["leave", "mine", scope, params],
  inbox: (scope, params) => ["leave", "inbox", scope, params],
  detail: (scope, id) => ["leave", "detail", scope, id],
};

export const useLeaveTypes = (params, enabled = true) =>
  useQuery({
    queryKey: leaveKeys.types(params),
    queryFn: ({ signal }) => leaveTypesRequest(params, signal),
    enabled,
    staleTime: 5 * 60 * 1000,
  });

export const useCreateLeaveType = () =>
  useMutation({
    mutationFn: (payload) => createLeaveTypeRequest(payload),
    retry: false,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: leaveKeys.all });
    },
  });

export const useUpdateLeaveType = () =>
  useMutation({
    mutationFn: ({ id, payload }) => updateLeaveTypeRequest(id, payload),
    retry: false,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: leaveKeys.all });
    },
  });

export const useCreateLeaveRequest = () =>
  useMutation({
    mutationFn: (input) => createLeaveRequest(input),
    retry: false,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: leaveKeys.all });
    },
  });

export const useMyLeaveRequests = (scope, params) =>
  useQuery({
    queryKey: leaveKeys.mine(scope, params),
    queryFn: ({ signal }) => myLeaveRequestsRequest(params, signal),
    placeholderData: keepPreviousData,
  });

export const useLeaveApprovalInbox = (scope, params, enabled = true) =>
  useQuery({
    queryKey: leaveKeys.inbox(scope, params),
    queryFn: ({ signal }) => leaveApprovalInboxRequest(params, signal),
    enabled,
    placeholderData: keepPreviousData,
  });

export const useLeaveRequestDetail = (scope, id) =>
  useQuery({
    queryKey: leaveKeys.detail(scope, id),
    queryFn: ({ signal }) => leaveRequestDetailRequest(id, signal),
    enabled: Boolean(id),
  });

// Keputusan tidak pernah optimistic; cache di-invalidate setelah server mengonfirmasi.
export const useDecideLeaveRequest = (scope, id) =>
  useMutation({
    mutationFn: (payload) => decideLeaveRequest(id, payload),
    retry: false,
    onSettled: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: leaveKeys.all }),
        queryClient.invalidateQueries({ queryKey: ["notifications"] }),
        queryClient.invalidateQueries({ queryKey: ["dashboard"] }),
      ]);
    },
  });

export const useDelegateLeaveRequest = (scope, id) =>
  useMutation({
    mutationFn: (payload) => delegateLeaveRequest(id, payload),
    retry: false,
    onSettled: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: leaveKeys.all }),
        queryClient.invalidateQueries({ queryKey: ["notifications"] }),
      ]);
    },
  });
