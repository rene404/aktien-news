const fmt = new Intl.DateTimeFormat("de-AT", {
  timeZone: "Europe/Vienna",
  day: "2-digit",
  month: "2-digit",
  year: "numeric",
  hour: "2-digit",
  minute: "2-digit",
});

/** "28.10.2026, 10:30" in Vienna time, independent of the server's TZ. */
export function formatTime(iso: string): string {
  return fmt.format(new Date(iso));
}
