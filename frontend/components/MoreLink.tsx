import Link from "next/link";
import type { Filters } from "@/lib/filters";

/** "Ältere Meldungen": same filters, next cursor. */
export default function MoreLink({ path, filters, cursor }: { path: string; filters: Filters; cursor: string | null }) {
  if (!cursor) return null;
  const qs = new URLSearchParams();
  if (filters.index) qs.set("index", filters.index);
  if (filters.source) qs.set("source", filters.source);
  qs.set("range", filters.range);
  qs.set("cursor", cursor);
  return (
    <Link className="more" href={`${path}?${qs}`}>
      Ältere Meldungen →
    </Link>
  );
}
