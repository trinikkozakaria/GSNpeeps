import { keepPreviousData, useMutation, useQuery } from "@tanstack/react-query";

import { queryClient } from "../../../lib/query/query-client";
import {
  deactivateEmployeeRequest,
  createEmployeeRequest,
  employeeDetailRequest,
  employeeDocumentsRequest,
  listDepartmentsRequest,
  listEmployeesRequest,
  listPositionsRequest,
  updateEmployeeRequest,
  uploadEmployeeDocumentRequest,
  uploadEmployeePhotoRequest,
  bulkEmployeesRequest,
} from "../api/employee-api";

export const employeeKeys = {
  all: ["employees"],
  list: (scope, filters) => ["employees", "list", scope, filters],
  detail: (scope, id) => ["employees", "detail", scope, id],
  documents: (scope, id) => ["employees", "documents", scope, id],
  departments: ["organization", "departments"],
  positions: (departmentId) => ["organization", "positions", departmentId ?? "all"],
};

export const useDepartments = () =>
  useQuery({
    queryKey: employeeKeys.departments,
    queryFn: ({ signal }) => listDepartmentsRequest(signal),
    staleTime: 10 * 60 * 1000,
  });

export const useEmployees = (scope, filters) =>
  useQuery({
    queryKey: employeeKeys.list(scope, filters),
    queryFn: ({ signal }) => listEmployeesRequest(filters, signal),
    placeholderData: keepPreviousData,
  });

export const usePositions = (departmentId) =>
  useQuery({
    queryKey: employeeKeys.positions(departmentId),
    queryFn: ({ signal }) => listPositionsRequest(departmentId, signal),
    enabled: Boolean(departmentId),
    staleTime: 10 * 60 * 1000,
  });

export const useEmployeeDetail = (scope, id) =>
  useQuery({
    queryKey: employeeKeys.detail(scope, id),
    queryFn: ({ signal }) => employeeDetailRequest(id, signal),
    enabled: Boolean(id),
    staleTime: 30 * 1000,
  });

export const useUpdateEmployee = (scope, id) =>
  useMutation({
    mutationFn: (payload) => updateEmployeeRequest(id, payload),
    retry: false,
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: employeeKeys.all }),
        queryClient.invalidateQueries({ queryKey: employeeKeys.detail(scope, id) }),
      ]);
    },
  });

export const useCreateEmployee = () =>
  useMutation({
    mutationFn: (payload) => createEmployeeRequest(payload),
    retry: false,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: employeeKeys.all });
    },
  });

export const useBulkEmployees = () =>
  useMutation({
    mutationFn: (file) => bulkEmployeesRequest(file),
    retry: false,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: employeeKeys.all });
    },
  });

export const useEmployeeDocuments = (scope, id, enabled = true) =>
  useQuery({
    queryKey: employeeKeys.documents(scope, id),
    queryFn: ({ signal }) => employeeDocumentsRequest(id, signal),
    enabled: Boolean(id) && enabled,
  });

export const useUploadEmployeeDocument = (scope, id) =>
  useMutation({
    mutationFn: (input) => uploadEmployeeDocumentRequest(id, input),
    retry: false,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: employeeKeys.documents(scope, id) });
    },
  });

export const useUploadEmployeePhoto = (scope, id) =>
  useMutation({
    mutationFn: (file) => uploadEmployeePhotoRequest(id, file),
    retry: false,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: employeeKeys.detail(scope, id) });
    },
  });

export const useDeactivateEmployee = (scope, id) =>
  useMutation({
    mutationFn: () => deactivateEmployeeRequest(id),
    retry: false,
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["employees", "list"] }),
        queryClient.invalidateQueries({
          queryKey: employeeKeys.detail(scope, id),
          refetchType: "none",
        }),
        queryClient.invalidateQueries({
          queryKey: employeeKeys.documents(scope, id),
          refetchType: "none",
        }),
      ]);
    },
  });
