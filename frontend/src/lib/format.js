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
