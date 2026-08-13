import { apiClient } from "../../../lib/api/client";
import { dashboardMetricsSchema } from "../schemas/dashboard-schema";

export const dashboardMetricsRequest = async ({ periode, tanggalAcuan }, signal) => {
  const params = {};
  if (periode) params.periode = periode;
  if (tanggalAcuan) params.tanggal_acuan = tanggalAcuan;
  const envelope = await apiClient.get("/dashboard/metrik", { params, signal });
  return dashboardMetricsSchema.parse(envelope.data);
};
