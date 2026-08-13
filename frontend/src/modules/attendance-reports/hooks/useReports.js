import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { attendanceReportRequest } from "../api/report-api";

export const reportKeys = {
  all: ["attendance-reports"],
  report: (scope, params) => ["attendance-reports", "report", scope, params],
};

export const useAttendanceReport = (scope, params, enabled = true) =>
  useQuery({
    queryKey: reportKeys.report(scope, params),
    queryFn: ({ signal }) => attendanceReportRequest(params, signal),
    enabled,
    placeholderData: keepPreviousData,
  });
