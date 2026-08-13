import { z } from "zod";

export const roleSummarySchema = z.object({
  id: z.string(),
  nama: z.string(),
  deskripsi: z.string(),
});

export const roleListSchema = z.array(roleSummarySchema);

// `aksi` mengikuti enum Permission pada OpenAPI.
export const permissionActions = ["create", "read", "update", "delete", "approve", "export"];

export const permissionSchema = z.object({
  id: z.string(),
  role_id: z.string(),
  modul: z.string(),
  aksi: z.string(),
  is_allowed: z.boolean(),
});

export const permissionMatrixSchema = z.array(permissionSchema);

export const updatePermissionSchema = z.object({
  role_id: z.string().uuid(),
  modul: z.string().min(1).max(100),
  aksi: z.enum(permissionActions),
  is_allowed: z.boolean(),
});
