const rawApiBaseUrl = import.meta.env.VITE_API_BASE_URL ?? "/api/v1";

if (!rawApiBaseUrl.startsWith("/") && !URL.canParse(rawApiBaseUrl)) {
  throw new Error("VITE_API_BASE_URL harus berupa URL valid atau path same-origin.");
}

export const config = Object.freeze({
  apiBaseUrl: rawApiBaseUrl.replace(/\/$/, ""),
  requestTimeoutMs: Number(import.meta.env.VITE_REQUEST_TIMEOUT_MS ?? 15000),
});

