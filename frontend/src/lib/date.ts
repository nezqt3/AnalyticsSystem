const DAY_MS = 24 * 60 * 60 * 1000;

export function formatISODate(date: Date): string {
  return date.toISOString().slice(0, 10);
}

export function getDefaultDateRange(): { from: string; to: string } {
  const now = new Date();
  return {
    from: formatISODate(new Date(now.getTime() - 7 * DAY_MS)),
    to: formatISODate(now),
  };
}

export function formatDateTime(value: string): string {
  if (!value) return "-";
  const parsed = new Date(value.replace(" ", "T"));
  if (Number.isNaN(parsed.getTime())) return value;
  return new Intl.DateTimeFormat("ru-RU", {
    day: "2-digit",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
  }).format(parsed);
}

export function formatPercent(value: number): string {
  return `${Math.round(value * 100)}%`;
}
