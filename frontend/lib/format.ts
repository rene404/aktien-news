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
  // Calendar arithmetic on noon UTC, which lies on the same day in Vienna;
  // now − 24 h would miss on the days the clocks change.
  const yesterday = new Date(`${dayKey(now)}T12:00:00Z`);
  yesterday.setUTCDate(yesterday.getUTCDate() - 1);
  if (key === dayKey(yesterday)) return "Gestern";
  return longDay.format(new Date(`${key}T12:00:00Z`));
}

const MINUTE = 60_000;
/** Sources whose clock runs a little ahead still read "gerade eben". */
const FUTURE_TOLERANCE = 5 * MINUTE;

/**
 * Whole minutes since iso while it is less than an hour old (0 = just now),
 * else null: older news shows the clock time instead.
 */
export function ageMinutes(iso: string, now: Date): number | null {
  const age = now.getTime() - new Date(iso).getTime();
  if (age < -FUTURE_TOLERANCE || age >= 60 * MINUTE) return null;
  return Math.floor(Math.max(age, 0) / MINUTE);
}
