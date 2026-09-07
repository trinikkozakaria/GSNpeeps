import { keepPreviousData, useMutation, useQuery } from "@tanstack/react-query";

import { queryClient } from "../../../lib/query/query-client";
import {
  attendanceCorrectionsRequest,
  createAttendanceCorrectionRequest,
  decideAttendanceCorrectionRequest,
  myAttendanceCorrectionsRequest,
} from "../api/correction-api";

// Antrean persetujuan (halaman Persetujuan): role menentukan cakupan (bawahan, seluruh
// koreksi, atau milik HR — lihat backend ListCorrections).
const correctionKey = (scope, params) => ["attendance-corrections", scope, params];

export const useAttendanceCorrections = (scope, params, enabled = true) =>
  useQuery({
    queryKey: correctionKey(scope, params),
    queryFn: ({ signal }) => attendanceCorrectionsRequest(params, signal),
    enabled,
    placeholderData: keepPreviousData,
  });

// Riwayat pribadi (halaman Koreksi Absensi): selalu milik user yang login sendiri,
// berlaku sama untuk semua role, setara useMyLeaveRequests/useMyOvertimeRequests.
// `userId` hanya menamespace cache per akun (mis. saat berganti user tanpa reload penuh).
const myCorrectionKey = (userId, params) => ["attendance-corrections", "mine", userId, params];

export const useMyAttendanceCorrections = (userId, params, enabled = true) =>
  useQuery({
    queryKey: myCorrectionKey(userId, params),
    queryFn: ({ signal }) => myAttendanceCorrectionsRequest(params, signal),
    enabled,
    placeholderData: keepPreviousData,
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
