import { getTopTickers } from "@/lib/api";
import { trendingParams } from "@/lib/feed";
import TickerChips from "./TickerChips";

const LABEL = "Heute im Gespräch";

// The most-mentioned stocks of the section in the last 24 hours, chips above
// the feed that wrap, so all of them are visible. Renders nothing when there
// are none (or the API fails).
export default async function Trending({ section }: { section: string }) {
  const items = await getTopTickers(trendingParams(section, new Date()));
  if (items.length === 0) return null;
  return (
    <div className="trending">
      <span className="trending-label" aria-hidden="true">
        {LABEL}
      </span>
      <TickerChips items={items} label={LABEL} compact />
    </div>
  );
}
