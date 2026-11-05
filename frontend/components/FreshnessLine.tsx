"use client";

import type { Source } from "@/lib/api";
import { ageMinutes, dayKey, formatClock, formatTime } from "@/lib/format";
import { useNow } from "@/lib/useNow";

/**
 * "16 Quellen · zuletzt aktualisiert vor 2 Min": how many sources feed the
 * current view and when the newest of them last fetched successfully. The
 * server renders the clock time; the browser switches to minutes for the last hour.
 */
export default function FreshnessLine({ sources }: { sources: Source[] }) {
  const now = useNow();
  if (sources.length === 0) return null;

  const count = `${sources.length} ${sources.length === 1 ? "Quelle" : "Quellen"}`;
  const last = sources
    .map((s) => s.last_ok_at)
    .filter((t): t is string => t !== null)
    .reduce<string | null>((a, b) => (a === null || Date.parse(b) > Date.parse(a) ? b : a), null);
  if (last === null) return <p className="freshness">{count}</p>;

  return (
    <p className="freshness">
      {count} · zuletzt aktualisiert{" "}
      <time dateTime={last} title={formatTime(last)}>
        {when(last, now)}
      </time>
    </p>
  );
}

function when(iso: string, now: number | null): string {
  if (now !== null) {
    const minutes = ageMinutes(iso, new Date(now));
    if (minutes === 0) return "gerade eben";
    if (minutes !== null) return `vor ${minutes} Min`;
  }
  return dayKey(iso) === dayKey(new Date(now ?? Date.now())) ? `um ${formatClock(iso)}` : `am ${formatTime(iso)}`;
}
