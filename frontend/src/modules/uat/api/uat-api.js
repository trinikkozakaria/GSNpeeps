import { apiClient } from "../../../lib/api/client";

const data = (envelope) => envelope.data;

/** GET /company-feed mendukung pagination (page/limit); default backend 20 per halaman. */
export const feedsRequest = async (params, signal) => {
  const envelope = await apiClient.get("/company-feed", { params, signal });
  return { items: envelope.data, meta: envelope.meta };
};
export const createFeedRequest = async (payload) => data(await apiClient.post("/company-feed", payload));
export const updateFeedRequest = async (id, payload) => data(await apiClient.put(`/company-feed/${id}`, payload));
export const deleteFeedRequest = async (id) => data(await apiClient.delete(`/company-feed/${id}`));
export const holidaysRequest = async (year, signal) => data(await apiClient.get("/kalender/libur", { params: { tahun: year }, signal }));
export const upsertHolidaysRequest = async (items) => data(await apiClient.put("/kalender/libur/bulk", { items }));
export const documentTypesRequest = async (signal) => data(await apiClient.get("/master/jenis-dokumen", { signal }));
export const createDocumentTypeRequest = async (payload) => data(await apiClient.post("/master/jenis-dokumen", payload));
export const homeSummaryRequest = async (signal) => data(await apiClient.get("/beranda", { signal }));
