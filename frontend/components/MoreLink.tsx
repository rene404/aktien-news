import Link from "next/link";
import { filterHref, type Filters } from "@/lib/filters";

/** "Ältere Meldungen": same filters, next cursor. */
export default function MoreLink({ path, filters, cursor, defaultRange }: { path: string; filters: Filters; cursor: string | null; defaultRange?: string }) {
  if (!cursor) return null;
  return (
    <Link className="more" href={filterHref(path, { ...filters, cursor }, { cursor: true, defaultRange })}>
      Ältere Meldungen →
    </Link>
  );
}
