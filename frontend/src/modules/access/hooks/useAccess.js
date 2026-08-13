import { useMutation, useQuery } from "@tanstack/react-query";

import { queryClient } from "../../../lib/query/query-client";
import {
  permissionMatrixRequest,
  rolesRequest,
  updatePermissionRequest,
} from "../api/access-api";

export const accessKeys = {
  all: ["access"],
  roles: ["access", "roles"],
  permissions: ["access", "permissions"],
};

export const useRoles = (enabled = true) =>
  useQuery({
    queryKey: accessKeys.roles,
    queryFn: ({ signal }) => rolesRequest(signal),
    enabled,
    staleTime: 5 * 60 * 1000,
  });

export const usePermissionMatrix = (enabled = true) =>
  useQuery({
    queryKey: accessKeys.permissions,
    queryFn: ({ signal }) => permissionMatrixRequest(signal),
    enabled,
  });

/**
 * Perubahan permission tidak pernah optimistic. Backend menolak konfigurasi yang melanggar
 * invariant produk, sehingga nilai yang ditulis lebih dulu di client dapat berbeda dari nilai
 * yang benar-benar tersimpan. Matriks selalu dibaca ulang setelah server menjawab.
 */
export const useUpdatePermission = () =>
  useMutation({
    mutationFn: (payload) => updatePermissionRequest(payload),
    retry: false,
    onSettled: async () => {
      await queryClient.invalidateQueries({ queryKey: accessKeys.all });
    },
  });
