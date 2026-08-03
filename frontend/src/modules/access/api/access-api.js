import { apiClient } from "../../../lib/api/client";
import {
  permissionMatrixSchema,
  roleListSchema,
} from "../schemas/access-schema";

export const rolesRequest = async (signal) => {
  const envelope = await apiClient.get("/akses/role", { signal });
  return roleListSchema.parse(envelope.data);
};

export const permissionMatrixRequest = async (signal) => {
  const envelope = await apiClient.get("/akses/permission", { signal });
  return permissionMatrixSchema.parse(envelope.data);
};

export const updatePermissionRequest = async (payload, signal) => {
  const envelope = await apiClient.put("/akses/permission", payload, { signal });
  return envelope.data;
};
