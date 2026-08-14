import { useMutation, useQuery } from "@tanstack/react-query";

import { queryClient } from "../../../lib/query/query-client";
import {
  attendanceCorrectionsRequest,
  createAttendanceCorrectionRequest,
  decideAttendanceCorrectionRequest,
} from "../api/correction-api";

const correctionKey = (scope) => ["attendance-corrections", scope];

export const useAttendanceCorrections = (scope, enabled = true) =>
  useQuery({
    queryKey: correctionKey(scope),
    queryFn: ({ signal }) => attendanceCorrectionsRequest(signal),
    enabled,
  });

const refreshCorrections = () =>
  queryClient.invalidateQueries({ queryKey: ["attendance-corrections"] });

export const useCreateAttendanceCorrection = () =>
  useMutation({
    mutationFn: createAttendanceCorrectionRequest,
    onSuccess: refreshCorrections,
  });

export const useDecideAttendanceCorrection = () =>
  useMutation({
    mutationFn: decideAttendanceCorrectionRequest,
    onSuccess: refreshCorrections,
  });
