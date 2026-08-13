// Backend belum mengirim Set-Cookie untuk token akses, sehingga sesi dipertahankan
// lewat cookie yang ditulis dari sisi browser. Cookie tidak dapat HttpOnly karena
// nilainya masih harus dibaca JavaScript untuk mengisi header Authorization.
const COOKIE_NAME = "gsnpeeps_session";

const isBrowser = () => typeof document !== "undefined";

const cookieAttributes = (maxAgeSeconds) => {
  const attributes = [`path=/`, `SameSite=Strict`, `Max-Age=${maxAgeSeconds}`];
  if (typeof window !== "undefined" && window.location.protocol === "https:") {
    attributes.push("Secure");
  }
  return attributes.join("; ");
};

const rawCookieValue = () => {
  const prefix = `${COOKIE_NAME}=`;
  const entry = document.cookie
    .split(";")
    .map((part) => part.trim())
    .find((part) => part.startsWith(prefix));
  return entry ? entry.slice(prefix.length) : null;
};

/**
 * Membaca sesi tersimpan. Mengembalikan null bila cookie tidak ada, rusak, atau kedaluwarsa.
 */
export const readStoredSession = () => {
  if (!isBrowser()) return null;
  const raw = rawCookieValue();
  if (!raw) return null;

  let parsed;
  try {
    parsed = JSON.parse(decodeURIComponent(raw));
  } catch {
    clearStoredSession();
    return null;
  }

  const token = parsed?.token;
  const expiresAt = parsed?.expires_at;
  if (typeof token !== "string" || token.length === 0 || typeof expiresAt !== "number") {
    clearStoredSession();
    return null;
  }
  if (expiresAt <= Date.now()) {
    clearStoredSession();
    return null;
  }
  return { token, expiresAt };
};

/**
 * Menyimpan token beserta waktu kedaluwarsa absolut (epoch ms).
 * Umur cookie mengikuti sisa masa berlaku token agar hilang otomatis saat token mati.
 */
export const writeStoredSession = ({ token, expiresAt }) => {
  if (!isBrowser()) return;
  const maxAgeSeconds = Math.floor((expiresAt - Date.now()) / 1000);
  if (!token || !Number.isFinite(maxAgeSeconds) || maxAgeSeconds <= 0) {
    clearStoredSession();
    return;
  }
  const value = encodeURIComponent(JSON.stringify({ token, expires_at: expiresAt }));
  document.cookie = `${COOKIE_NAME}=${value}; ${cookieAttributes(maxAgeSeconds)}`;
};

export const clearStoredSession = () => {
  if (!isBrowser()) return;
  document.cookie = `${COOKIE_NAME}=; ${cookieAttributes(0)}`;
};

export const __testing = { COOKIE_NAME };
