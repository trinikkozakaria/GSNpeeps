import { useQuery } from "@tanstack/react-query";

import { myProfileRequest, personalMetricsRequest } from "../api/profile-api";

// Scope memakai user ID sehingga cache tidak pernah dipakai ulang lintas identitas.
export const profileKeys = {
  all: ["profile"],
  me: (userId) => ["profile", "me", userId],
  metrics: (userId) => ["profile", "metrics", userId],
};

export const useMyProfile = (userId) =>
  useQuery({
    queryKey: profileKeys.me(userId),
    queryFn: ({ signal }) => myProfileRequest(signal),
    enabled: Boolean(userId),
  });

export const usePersonalMetrics = (userId, enabled = true) =>
  useQuery({
    queryKey: profileKeys.metrics(userId),
    queryFn: ({ signal }) => personalMetricsRequest(signal),
    enabled: Boolean(userId) && enabled,
  });
