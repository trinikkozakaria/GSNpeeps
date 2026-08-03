import { useMutation, useQuery } from "@tanstack/react-query";

import { queryClient } from "../../../lib/query/query-client";
import {
  liveFeedRequest,
  officeLocationsRequest,
  recordAttendanceRequest,
} from "../api/attendance-api";

export const attendanceKeys = {
  all: ["attendance"],
  offices: ["organization", "office-locations"],
  liveFeed: (scope, tanggal) => ["attendance", "live-feed", scope, tanggal],
};

export const useOfficeLocations = (enabled = true) =>
  useQuery({
    queryKey: attendanceKeys.offices,
    queryFn: ({ signal }) => officeLocationsRequest(signal),
    enabled,
    staleTime: 10 * 60 * 1000,
  });

export const useRecordAttendance = () =>
  useMutation({
    mutationFn: (input) => recordAttendanceRequest(input),
    retry: false,
    onSuccess: async () => {
      // Absensi baru mengubah live feed, laporan, dan metrik personal.
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: attendanceKeys.all }),
        queryClient.invalidateQueries({ queryKey: ["profile"] }),
        queryClient.invalidateQueries({ queryKey: ["dashboard"] }),
      ]);
    },
  });

export const useLiveFeed = (scope, tanggal, enabled = true) =>
  useQuery({
    queryKey: attendanceKeys.liveFeed(scope, tanggal),
    queryFn: ({ signal }) => liveFeedRequest(tanggal, signal),
    enabled,
  });
