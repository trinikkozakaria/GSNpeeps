import axios from "axios";

import { config } from "../config";
import { configureDownloadBoundary } from "./download";
import { AppError, normalizeApiError } from "./errors";

let readAccessToken = () => null;
let handleUnauthorized = () => {};

export const configureAuthBoundary = ({ getAccessToken, onUnauthorized }) => {
  readAccessToken = getAccessToken ?? (() => null);
  handleUnauthorized = onUnauthorized ?? (() => {});
  // Unduhan berkas memakai fetch terpisah tetapi harus memakai token yang sama.
  configureDownloadBoundary({ getAccessToken });
};

export const apiClient = axios.create({
  baseURL: config.apiBaseUrl,
  timeout: config.requestTimeoutMs,
  headers: {
    Accept: "application/json",
  },
});

export const protectedMediaRequest = async (path, signal) => {
  const token = readAccessToken();
  const response = await axios.get(`${config.apiBaseUrl}/media`, {
    params: { path }, signal, responseType: "blob",
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
  });
  return response.data;
};

apiClient.interceptors.request.use((request) => {
  const token = readAccessToken();
  if (token) {
    request.headers.Authorization = `Bearer ${token}`;
  }
  return request;
});

apiClient.interceptors.response.use(
  (response) => {
    if (!response.data || response.data.success !== true || !("data" in response.data)) {
      throw new AppError({
        status: response.status,
        code: "INVALID_RESPONSE",
        message: "Format respons layanan tidak sesuai kontrak.",
      });
    }
    return response.data;
  },
  (error) => {
    const normalized = normalizeApiError(error);
    if (normalized.status === 401) {
      handleUnauthorized();
    }
    return Promise.reject(normalized);
  },
);

