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

export default async function StockPage({ params, searchParams }: Props) {
  const { ticker } = await params;
  const inst = await getInstrument(decodeURIComponent(ticker));
  if (!inst) notFound();

  const filters = { ...parseFilters(await searchParams), index: "" };
  const [page, sources] = await Promise.all([
    getNews({ ...toNewsParams(filters, new Date()), ticker: inst.ticker }),
    getSources(),
  ]);

  return (
    <>
      <h1>
        {inst.name} <span className="chip">{inst.ticker}</span>
      </h1>
      <p className="sub">
        {[inst.isin, inst.mic, ...inst.indices].filter(Boolean).join(" · ")}
      </p>
      <Filters sources={sources} value={filters} showIndex={false} />
      <NewsList items={page.items} hideTicker={inst.ticker} />
      <MoreLink path={`/aktie/${encodeURIComponent(inst.ticker)}`} filters={filters} cursor={page.next_cursor} />
    </>
  );
}
