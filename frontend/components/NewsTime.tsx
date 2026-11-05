"use client";

import { ageMinutes, formatClock, formatTime } from "@/lib/format";
import { useNow } from "@/lib/useNow";

/**
 * The time of a news item. The server renders the clock time; in the browser,
 * news from the last hour reads "jetzt" or "12 Min" ("vor 12 Min" for screen
 * readers), short enough for the time column. The full time is in the title.
 */
export default function NewsTime({ iso }: { iso: string }) {
  const now = useNow();
  const minutes = now === null ? null : ageMinutes(iso, new Date(now));
  return (
    <time dateTime={iso} title={formatTime(iso)}>
      {minutes === null ? (
        formatClock(iso)
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
