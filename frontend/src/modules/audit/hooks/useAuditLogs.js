import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { auditLogsRequest } from "../api/audit-api";

export const auditKeys = {
  all: ["audit"],
  list: (scope, params) => ["audit", "list", scope, params],
};

// Audit Log hanya memiliki operasi baca; tidak ada mutation hook karena kontrak tidak
// menyediakan endpoint edit maupun hapus.
export const useAuditLogs = (scope, params, enabled = true) =>
  useQuery({
    queryKey: auditKeys.list(scope, params),
    queryFn: ({ signal }) => auditLogsRequest(params, signal),
    enabled,
    placeholderData: keepPreviousData,
  });
