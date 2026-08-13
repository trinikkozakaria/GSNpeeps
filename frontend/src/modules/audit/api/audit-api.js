import { apiClient } from "../../../lib/api/client";
import { auditLogListSchema } from "../schemas/audit-schema";

export const auditLogsRequest = async (params, signal) => {
  const envelope = await apiClient.get("/akses/audit-log", { params, signal });
  return auditLogListSchema.parse({ items: envelope.data, meta: envelope.meta });
};
