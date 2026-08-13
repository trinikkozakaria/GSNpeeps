import { apiClient } from "../../../lib/api/client";

const data = (envelope) => envelope.data;
export const feedsRequest = async (signal) => data(await apiClient.get("/company-feed", { signal }));
export const createFeedRequest = async (payload) => data(await apiClient.post("/company-feed", payload));
export const holidaysRequest = async (year, signal) => data(await apiClient.get("/kalender/libur", { params: { tahun: year }, signal }));
export const upsertHolidaysRequest = async (items) => data(await apiClient.put("/kalender/libur/bulk", { items }));
export const documentTypesRequest = async (signal) => data(await apiClient.get("/master/jenis-dokumen", { signal }));
export const createDocumentTypeRequest = async (payload) => data(await apiClient.post("/master/jenis-dokumen", payload));
