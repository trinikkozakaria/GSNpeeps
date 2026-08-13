import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { dashboardMetricsRequest } from "../api/dashboard-api";

export const dashboardKeys = {
  all: ["dashboard"],
  metrics: (scope, filters) => ["dashboard", "metrics", scope, filters],
};

export const useDashboardMetrics = (scope, filters) =>
  useQuery({
    queryKey: dashboardKeys.metrics(scope, filters),
    queryFn: ({ signal }) => dashboardMetricsRequest(filters, signal),
    placeholderData: keepPreviousData,
  });
