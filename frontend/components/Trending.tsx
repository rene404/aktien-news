import { getTopTickers } from "@/lib/api";
import { trendingParams } from "@/lib/feed";
import ScrollRow from "./ScrollRow";
import TickerChips from "./TickerChips";

const LABEL = "Heute im Gespräch";

// The most-mentioned stocks of the section in the last 24 hours, one row of
// chips above the feed. Renders nothing when there are none (or the API fails).
export default async function Trending({ section }: { section: string }) {
  const items = await getTopTickers(trendingParams(section, new Date()));
  if (items.length === 0) return null;
  return (
    <div className="trending">
      <span className="trending-label" aria-hidden="true">
        {LABEL}
      </span>
      <ScrollRow>
        <TickerChips items={items} label={LABEL} compact />
      </ScrollRow>
    </div>
  );
}
