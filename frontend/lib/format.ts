// All times are shown in Vienna time, independent of the server's TZ.
const TZ = "Europe/Vienna";

const full = new Intl.DateTimeFormat("de-AT", {
  timeZone: TZ,
  day: "2-digit",
  month: "2-digit",
  year: "numeric",
  hour: "2-digit",
  minute: "2-digit",
});
const clock = new Intl.DateTimeFormat("de-AT", { timeZone: TZ, hour: "2-digit", minute: "2-digit" });
const isoDay = new Intl.DateTimeFormat("en-CA", { timeZone: TZ, year: "numeric", month: "2-digit", day: "2-digit" });
const longDay = new Intl.DateTimeFormat("de-AT", { timeZone: TZ, weekday: "long", day: "2-digit", month: "2-digit", year: "numeric" });

/** "28.10.2026, 10:30" */
export function formatTime(iso: string): string {
  return full.format(new Date(iso));
}

/** "10:30" */
export function formatClock(iso: string): string {
  return clock.format(new Date(iso));
}

/** The Vienna calendar day, "2026-10-28". */
export function dayKey(d: Date | string): string {
  return isoDay.format(new Date(d));
}

/** "Heute", "Gestern" or "Montag, 26.10.2026" for a dayKey. */
export function dayLabel(key: string, now: Date): string {
  if (key === dayKey(now)) return "Heute";
  if (key === dayKey(new Date(now.getTime() - 24 * 3600_000))) return "Gestern";
  // Noon UTC lies on the same calendar day in Vienna.
  return longDay.format(new Date(`${key}T12:00:00Z`));
}
