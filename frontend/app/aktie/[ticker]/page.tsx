import type { Metadata } from "next";
import { notFound } from "next/navigation";
import Filters from "@/components/Filters";
import InfiniteNews from "@/components/InfiniteNews";
import { getInstrument, getNews, getSources } from "@/lib/api";
import { feedQuery, STOCK_DEFAULT_RANGE, stockNewsParams } from "@/lib/feed";
import { filterHref, parseFilters, type SearchParams } from "@/lib/filters";

type Props = { params: Promise<{ ticker: string }>; searchParams: Promise<SearchParams> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { ticker } = await params;
  const inst = await getInstrument(decodeURIComponent(ticker));
  if (!inst) return { title: "Aktie nicht gefunden" };
  return {
    title: `${inst.name} (${inst.ticker}) Aktie News`,
    description: `Aktuelle Meldungen und Ad-hoc-Mitteilungen zur ${inst.name}-Aktie (${inst.isin ?? inst.ticker}).`,
  };
}

// Market identifier codes as readers know them.
const MARKETS: Record<string, string> = { XWBO: "Wiener Börse", XETR: "Xetra", XNYS: "NYSE", XNAS: "Nasdaq" };

export default async function StockPage({ params, searchParams }: Props) {
  const { ticker } = await params;
  const inst = await getInstrument(decodeURIComponent(ticker));
  if (!inst) notFound();

  const filters = { ...parseFilters(await searchParams, STOCK_DEFAULT_RANGE), section: "" };
  const now = new Date();
  const [page, sources] = await Promise.all([
    getNews(stockNewsParams(filters, inst.ticker, now)),
    getSources(),
  ]);
  const path = `/aktie/${encodeURIComponent(inst.ticker)}`;
  const query = feedQuery(filters, inst.ticker);

  return (
    <>
      <h1>
        {inst.name} <span className="chip">{inst.ticker}</span>
      </h1>
      <p className="sub">
        {[inst.isin, MARKETS[inst.mic] ?? inst.mic, ...inst.indices].filter(Boolean).join(" · ")}
      </p>
      <Filters sources={sources} value={filters} />
      <InfiniteNews
        key={`${query}:${page.next_cursor}`}
        items={page.items}
        nextCursor={page.next_cursor}
        query={query}
        moreHref={filterHref(path, { ...filters, cursor: page.next_cursor ?? "" }, { cursor: true, defaultRange: STOCK_DEFAULT_RANGE })}
        hideTicker={inst.ticker}
        now={now.toISOString()}
      />
    </>
  );
}
