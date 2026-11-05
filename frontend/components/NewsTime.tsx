"use client";

import { ageMinutes, dayKey, formatClock, formatShortDay, formatTime } from "@/lib/format";
import { useNow } from "@/lib/useNow";

/**
 * The time of a news item. The server renders the clock time; in the browser,
 * news from the last hour reads "jetzt" or "12 Min" ("vor 12 Min" for screen
 * readers), short enough for the time column. The full time is in the title.
 * showDay: where no day heading groups the rows (tiles), older days show "03.11.";
 * serverNow (ISO) is "today" until the browser has its own clock.
 */
export default function NewsTime({ iso, showDay = false, serverNow }: { iso: string; showDay?: boolean; serverNow?: string }) {
  const now = useNow();
  const minutes = now === null ? null : ageMinutes(iso, new Date(now));
  const ref = now ?? serverNow;
  const older = showDay && ref !== undefined && dayKey(iso) !== dayKey(new Date(ref));
  return (
    <time dateTime={iso} title={formatTime(iso)}>
      {minutes === null ? (
        older ? formatShortDay(iso) : formatClock(iso)
      ) : minutes === 0 ? (
        "jetzt"
      ) : (
        <>
          <span className="sr-only">vor </span>
          {minutes} Min
        </>
      )}
    </time>
  );
}
