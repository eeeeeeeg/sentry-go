export function formatDate(value?: string): string {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

export function compactNumber(value: number | undefined): string {
  return new Intl.NumberFormat("zh-CN", { notation: "compact" }).format(value ?? 0);
}

export function levelClass(level?: string): string {
  switch (level) {
    case "fatal":
      return "badge bg-red-100 text-red-700";
    case "error":
      return "badge bg-red-100 text-red-700";
    case "warning":
      return "badge bg-amber-100 text-amber-800";
    case "info":
      return "badge bg-sky-100 text-sky-800";
    default:
      return "badge bg-slate-100 text-slate-700";
  }
}

export function prettyJSON(raw?: string) {
  if (!raw) return "-";
  try {
    return JSON.stringify(JSON.parse(raw), null, 2);
  } catch {
    return raw;
  }
}

export function datetimeLocalToRFC3339(value: string) {
  if (!value) {
    return undefined;
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return undefined;
  }
  return date.toISOString();
}
