import dayjs from "dayjs";

// formatDateTime renders a unix-seconds timestamp for the console.
export function formatDateTime(ts?: number): string {
  return ts ? dayjs.unix(ts).format("YYYY-MM-DD HH:mm") : "never";
}

// grantExpiry renders a grant's expiry and whether it has lapsed.
export function grantExpiry(ts?: number): { label: string; expired: boolean } {
  if (!ts) return { label: "never", expired: false };
  const at = dayjs.unix(ts);
  const date = at.format("YYYY-MM-DD");
  return at.isBefore(dayjs())
    ? { label: `expired ${date}`, expired: true }
    : { label: `until ${date}`, expired: false };
}
