import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { Suspense } from "react";
import Filters from "@/components/Filters";
import NewsFeed from "@/components/NewsFeed";
import NewsSkeleton from "@/components/NewsSkeleton";
import { getInstrument, getSources } from "@/lib/api";
import { feedQuery, STOCK_DEFAULT_RANGE } from "@/lib/feed";
import { parseFilters, type SearchParams } from "@/lib/filters";

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
  const sources = await getSources();
  const path = `/aktie/${encodeURIComponent(inst.ticker)}`;

  return (
    <>
      <h1>
        {inst.name} <span className="chip">{inst.ticker}</span>
      </h1>
      <p className="sub">
        {[inst.isin, MARKETS[inst.mic] ?? inst.mic, ...inst.indices].filter(Boolean).join(" · ")}
      </p>
      <Filters path={path} sources={sources} value={filters} />
      <Suspense key={feedQuery(filters, inst.ticker)} fallback={<NewsSkeleton />}>
        <NewsFeed filters={filters} path={path} ticker={inst.ticker} />
      </Suspense>
    </>
  );
}
