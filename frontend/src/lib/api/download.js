import { config } from "../config";
import { AppError, normalizeApiError } from "./errors";

// Endpoint export mengembalikan file stream, bukan envelope JSON, sehingga tidak dapat
// melewati interceptor response apiClient. Helper ini memakai fetch dengan Bearer token
// yang sama dan menormalkan error ke bentuk AppError.
let readAccessToken = () => null;

export const configureDownloadBoundary = ({ getAccessToken }) => {
  readAccessToken = getAccessToken ?? (() => null);
};

const filenameFromDisposition = (headerValue, fallback) => {
  if (!headerValue) return fallback;
  const encoded = /filename\*=UTF-8''([^;]+)/i.exec(headerValue);
  const plain = /filename="?([^";]+)"?/i.exec(headerValue);
  const raw = encoded ? decodeURIComponent(encoded[1]) : plain?.[1];
  if (!raw) return fallback;
  // Buang komponen path apa pun yang dikirim server dan batasi ke karakter aman.
  const base = raw.split(/[\\/]/).pop()?.trim() ?? "";
  const safe = base.replace(/[^A-Za-z0-9._-]/g, "-").replace(/^[-.]+/, "");
  return safe || fallback;
};

/**
 * Mengunduh berkas terautentikasi dan mengembalikan blob beserta nama berkas yang aman.
 * Pemanggil bertanggung jawab membuat dan mencabut object URL.
 */
export const downloadFile = async (path, params, { signal, fallbackFileName } = {}) => {
  const query = new URLSearchParams();
  Object.entries(params ?? {}).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== "") {
      query.set(key, String(value));
    }
  });
  const suffix = query.toString() ? `?${query}` : "";
  const token = readAccessToken();

  let response;
  try {
    response = await fetch(`${config.apiBaseUrl}${path}${suffix}`, {
      method: "GET",
      signal,
      headers: token ? { Authorization: `Bearer ${token}` } : undefined,
    });
  } catch (error) {
    if (error?.name === "AbortError") {
      throw new AppError({ code: "REQUEST_ABORTED", message: "Permintaan dibatalkan.", cause: error });
    }
    throw new AppError({ code: "NETWORK_ERROR", message: "Tidak dapat terhubung ke layanan.", cause: error });
  }

  if (!response.ok) {
    let payload = null;
    try {
      payload = await response.json();
    } catch {
      payload = null;
    }
    throw normalizeApiError({
      response: { status: response.status, data: payload },
    });
  }

  return {
    blob: await response.blob(),
    fileName: filenameFromDisposition(
      response.headers.get("Content-Disposition"),
      fallbackFileName,
    ),
  };
};

export const __testing = { filenameFromDisposition };
