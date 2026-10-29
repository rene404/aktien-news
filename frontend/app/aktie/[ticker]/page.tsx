import type { Metadata } from "next";
import { notFound } from "next/navigation";
import Filters from "@/components/Filters";
import MoreLink from "@/components/MoreLink";
import NewsList from "@/components/NewsList";
import { getInstrument, getNews, getSources } from "@/lib/api";
import { parseFilters, toNewsParams, type SearchParams } from "@/lib/filters";

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

// Single stocks have few headlines, so the stock page shows all of them by
// default, filings included.
const DEFAULT_RANGE = "all";

export default async function StockPage({ params, searchParams }: Props) {
  const { ticker } = await params;
  const inst = await getInstrument(decodeURIComponent(ticker));
  if (!inst) notFound();

  const filters = { ...parseFilters(await searchParams, DEFAULT_RANGE), section: "" };
  const now = new Date();
  const [page, sources] = await Promise.all([
    getNews({ ...toNewsParams(filters, now), kind: undefined, ticker: inst.ticker }),
    getSources(),
  ]);
  const path = `/aktie/${encodeURIComponent(inst.ticker)}`;

  return (
    <>
      <h1>
        {inst.name} <span className="chip">{inst.ticker}</span>
      </h1>
      <p className="sub">
        {[inst.isin, MARKETS[inst.mic] ?? inst.mic, ...inst.indices].filter(Boolean).join(" · ")}
      </p>
      <Filters sources={sources} value={filters} />
      <NewsList items={page.items} hideTicker={inst.ticker} now={now} />
      <MoreLink path={path} filters={filters} cursor={page.next_cursor} defaultRange={DEFAULT_RANGE} />
    </>
  );
}
