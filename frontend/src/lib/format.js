// Format tampilan memakai locale Indonesia. Nilai wire dari API tidak pernah diubah;
// formatting hanya terjadi pada lapisan presentasi.
const dateFormatter = new Intl.DateTimeFormat("id-ID", {
  day: "numeric",
  month: "long",
  year: "numeric",
  timeZone: "Asia/Jakarta",
});

const currencyFormatter = new Intl.NumberFormat("id-ID", {
  style: "currency",
  currency: "IDR",
  maximumFractionDigits: 0,
});

const monthFormatter = new Intl.DateTimeFormat("id-ID", {
  month: "long",
  year: "numeric",
  timeZone: "Asia/Jakarta",
});

const dateTimeFormatter = new Intl.DateTimeFormat("id-ID", {
  day: "numeric",
  month: "short",
  year: "numeric",
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
  timeZone: "Asia/Jakarta",
});

const timeFormatter = new Intl.DateTimeFormat("id-ID", {
  hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false,
  timeZone: "Asia/Jakarta",
});

/** Memformat tanggal `YYYY-MM-DD` dari API tanpa pergeseran timezone. */
export const formatDate = (value) => {
  if (!value) return "";
  const parsed = new Date(`${value}T00:00:00+07:00`);
  return Number.isNaN(parsed.getTime()) ? value : dateFormatter.format(parsed);
};

/** Memformat periode `YYYY-MM` menjadi nama bulan dan tahun. */
export const formatPeriod = (value) => {
  if (!value) return "";
  const parsed = new Date(`${value}-01T00:00:00+07:00`);
  return Number.isNaN(parsed.getTime()) ? value : monthFormatter.format(parsed);
};

/**
 * Memformat timestamp ISO dari API. Zona ditetapkan Asia/Jakarta agar seluruh pengguna
 * membaca waktu yang sama dengan waktu server, bukan zona perangkat masing-masing.
 */
export const formatDateTime = (value) => {
  if (!value) return "";
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? value : `${dateTimeFormatter.format(parsed)} WIB`;
};

export const formatCurrency = (value) =>
  typeof value === "number" ? currencyFormatter.format(value) : "";

export const formatNumber = (value) =>
  typeof value === "number" ? value.toLocaleString("id-ID") : "";

export const formatPercent = (value) =>
  typeof value === "number"
    ? `${value.toLocaleString("id-ID", { maximumFractionDigits: 2 })}%`
    : "";

export const formatHours = (value) =>
  typeof value === "number"
    ? `${value.toLocaleString("id-ID", { maximumFractionDigits: 1 })} jam`
    : "";

/** Menghitung usia penuh berdasarkan tanggal hari ini di zona Asia/Jakarta. */
export const calculateAge = (birthDate, now = new Date()) => {
  if (!birthDate) return null;
  const [birthYear, birthMonth, birthDay] = birthDate.slice(0, 10).split("-").map(Number);
  if (![birthYear, birthMonth, birthDay].every(Number.isInteger)) return null;
  const parts = new Intl.DateTimeFormat("en-CA", {
    timeZone: "Asia/Jakarta", year: "numeric", month: "2-digit", day: "2-digit",
  }).formatToParts(now);
  const today = Object.fromEntries(parts.map((part) => [part.type, Number(part.value)]));
  let age = today.year - birthYear;
  if (today.month < birthMonth || (today.month === birthMonth && today.day < birthDay)) age -= 1;
  return Math.max(0, age);
};

export const formatTime = (value) => {
  if (!value) return "";
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? value : `${timeFormatter.format(parsed)} WIB`;
};
