const rupiah = new Intl.NumberFormat("id-ID", {
  style: "currency",
  currency: "IDR",
  maximumFractionDigits: 0,
});

const compact = new Intl.NumberFormat("id-ID", {
  notation: "compact",
  maximumFractionDigits: 1,
});

/** Money is always integer rupiah; never format it as a float. */
export function formatIDR(value: number): string {
  return rupiah.format(Math.round(value));
}

export function formatCompact(value: number): string {
  return `Rp${compact.format(Math.round(value))}`;
}

export function formatSignedIDR(value: number): string {
  const sign = value > 0 ? "+" : value < 0 ? "−" : "";
  return `${sign}${formatIDR(Math.abs(value))}`;
}

const dayMonth = new Intl.DateTimeFormat("id-ID", {
  day: "numeric",
  month: "short",
});

const dayMonthYear = new Intl.DateTimeFormat("id-ID", {
  day: "numeric",
  month: "short",
  year: "numeric",
});

const monthYear = new Intl.DateTimeFormat("id-ID", {
  month: "long",
  year: "numeric",
});

function safeDate(iso: string): Date | null {
  if (!iso) return null;
  const date = new Date(iso);
  return Number.isNaN(date.getTime()) ? null : date;
}

export function formatDayMonth(iso: string): string {
  const date = safeDate(iso);
  return date ? dayMonth.format(date) : "–";
}

export function formatDayMonthYear(iso: string): string {
  const date = safeDate(iso);
  return date ? dayMonthYear.format(date) : "–";
}

export function formatMonthYear(iso: string): string {
  const date = safeDate(iso);
  return date ? monthYear.format(date) : "–";
}

export function formatMonthLabel(month: string): string {
  // "2026-11" (optionally "2026-11-01") → "Nov 2026"
  const date = safeDate(`${month.slice(0, 7)}-01T00:00:00`);
  return date
    ? new Intl.DateTimeFormat("id-ID", { month: "short", year: "numeric" }).format(date)
    : month;
}

export function formatDateInput(iso: string): string {
  const date = new Date(iso);
  const offset = date.getTimezoneOffset();
  return new Date(date.getTime() - offset * 60_000).toISOString().slice(0, 10);
}

export function formatPercent(value: number, digits = 0): string {
  return `${(value * 100).toFixed(digits)}%`;
}

export function titleCase(value: string): string {
  return value
    .split(" ")
    .map((word) => (word ? word[0].toUpperCase() + word.slice(1) : word))
    .join(" ");
}

export function transactionSign(type: string): "+" | "−" | "" {
  if (type === "income") return "+";
  if (type === "expense") return "−";
  return "";
}

/**
 * Accepts the way people actually type rupiah in Indonesia: "25rb", "1,5jt",
 * "10.000", "50000". Returns null when the input is not a positive amount.
 */
export function parseAmountInput(raw: string): number | null {
  const text = raw.trim().toLowerCase().replace(/\s+/g, "");
  if (!text) return null;
  const match = /^(?:rp\.?)?([0-9][0-9.,]*)(rb|ribu|k|jt|juta)?$/.exec(text);
  if (!match) return null;
  const multiplier = ["rb", "ribu", "k"].includes(match[2])
    ? 1_000
    : ["jt", "juta"].includes(match[2])
      ? 1_000_000
      : 1;
  // Indonesian: "." thousands separator, "," decimal separator.
  const cleaned = match[1].replace(/\./g, "").replace(/,/g, ".");
  const value = Number(cleaned) * multiplier;
  if (!Number.isFinite(value) || value <= 0) return null;
  return Math.round(value);
}

export function relativeTime(iso: string): string {
  const parsed = safeDate(iso);
  if (!parsed) return "–";
  const date = parsed.getTime();
  const diff = Date.now() - date;
  const minutes = Math.round(diff / 60_000);
  if (minutes < 1) return "baru saja";
  if (minutes < 60) return `${minutes} menit lalu`;
  const hours = Math.round(minutes / 60);
  if (hours < 24) return `${hours} jam lalu`;
  const days = Math.round(hours / 24);
  if (days < 30) return `${days} hari lalu`;
  return formatDayMonthYear(iso);
}
