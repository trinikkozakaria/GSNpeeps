import { apiClient } from "../../../lib/api/client";
import { myProfileSchema, personalMetricsSchema } from "../schemas/profile-schema";

export const myProfileRequest = async (signal) => {
  const envelope = await apiClient.get("/profil/saya", { signal });
  return myProfileSchema.parse(envelope.data);
};

export const personalMetricsRequest = async (signal) => {
  const envelope = await apiClient.get("/profil/saya/metrik", { signal });
  return personalMetricsSchema.parse(envelope.data);
};
