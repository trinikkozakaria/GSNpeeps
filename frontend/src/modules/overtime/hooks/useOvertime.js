import { keepPreviousData, useMutation, useQuery } from "@tanstack/react-query";

import { queryClient } from "../../../lib/query/query-client";
import {
  createOvertimeRequest,
  decideOvertimeRequest,
  myOvertimeRequestsRequest,
  overtimeDetailRequest,
  overtimeListRequest,
  overtimeRecapRequest,
} from "../api/overtime-api";

export const overtimeKeys = {
  all: ["overtime"],
  list: (scope, params) => ["overtime", "list", scope, params],
  mine: (scope, params) => ["overtime", "mine", scope, params],
  detail: (scope, id) => ["overtime", "detail", scope, id],
  recap: (scope, params) => ["overtime", "recap", scope, params],
};

export const useCreateOvertimeRequest = () =>
  useMutation({
    mutationFn: (input) => createOvertimeRequest(input),
    retry: false,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: overtimeKeys.all });
    },
  });

export const useOvertimeList = (scope, params, enabled = true) =>
  useQuery({
    queryKey: overtimeKeys.list(scope, params),
    queryFn: ({ signal }) => overtimeListRequest(params, signal),
    enabled,
    placeholderData: keepPreviousData,
  });

export const useMyOvertimeRequests = (scope, params, enabled = true) =>
  useQuery({
    queryKey: overtimeKeys.mine(scope, params),
    queryFn: ({ signal }) => myOvertimeRequestsRequest(params, signal),
    enabled,
    placeholderData: keepPreviousData,
  });

export const useOvertimeDetail = (scope, id) =>
  useQuery({
    queryKey: overtimeKeys.detail(scope, id),
    queryFn: ({ signal }) => overtimeDetailRequest(id, signal),
    enabled: Boolean(id),
  });

export const useDecideOvertimeRequest = (scope, id) =>
  useMutation({
    mutationFn: (payload) => decideOvertimeRequest(id, payload),
    retry: false,
    onSettled: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: overtimeKeys.all }),
        queryClient.invalidateQueries({ queryKey: ["notifications"] }),
      ]);
    },
  });

export const useOvertimeRecap = (scope, params, enabled = true) =>
  useQuery({
    queryKey: overtimeKeys.recap(scope, params),
    queryFn: ({ signal }) => overtimeRecapRequest(params, signal),
    enabled,
  });
