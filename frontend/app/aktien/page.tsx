import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import TabNav from "@/components/TabNav";
import TickerChips from "@/components/TickerChips";
import { getIndexInstruments } from "@/lib/api";
import { NEWS_KINDS, type SearchParams } from "@/lib/filters";
import { LIST_INDICES, LIST_RANGES, LIST_SORTS, parseStockList, sortTickers, stockListHref, type StockList } from "@/lib/stocklist";

type Props = { searchParams: Promise<SearchParams> };

export async function generateMetadata({ searchParams }: Props): Promise<Metadata> {
  const label = indexLabel(parseStockList(await searchParams).index);
  return {
    title: `${label}-Aktien – News je Wert`,
    description: `Alle ${label}-Werte mit der Zahl ihrer aktuellen Meldungen und Ad-hoc-Mitteilungen.`,
  };
}

export default async function StocksPage({ searchParams }: Props) {
  const list = parseStockList(await searchParams);
  const hours = LIST_RANGES.find((r) => r.key === list.range)!.hours;
  const items = await getIndexInstruments(list.index, { from: since(hours), kind: NEWS_KINDS.join(",") });
  if (!items) notFound();
  const label = indexLabel(list.index);

  return (
    <>
      <h1>Aktien</h1>
      <p className="sub">Die Werte von ATX, DAX und S&amp;P 500 mit der Zahl ihrer Meldungen.</p>
      <TabNav
        label="Indizes"
        tabs={LIST_INDICES.map((i) => ({ href: stockListHref({ ...list, index: i.key }), label: i.label, current: i.key === list.index }))}
      />
      <div className="options">
        <Segmented label="Zeitraum" list={list} field="range" options={LIST_RANGES} />
        <Segmented label="Sortierung" list={list} field="sort" options={LIST_SORTS} />
      </div>
      <TickerChips items={sortTickers(items, list.sort)} label={`${label}-Aktien`} />
    </>
  );
}

// A choice of links, the current one highlighted; works without JavaScript.
function Segmented({ label, list, field, options }: {
  label: string;
  list: StockList;
  field: "range" | "sort";
  options: readonly { key: string; label: string }[];
}) {
  return (
    <nav className="segmented" aria-label={label}>
      {options.map((o) => (
        <Link key={o.key} href={stockListHref({ ...list, [field]: o.key })} aria-current={o.key === list[field] ? "true" : undefined}>
          {o.label}
        </Link>
      ))}
    </nav>
  );
}

function indexLabel(key: string): string {
  return LIST_INDICES.find((i) => i.key === key)?.label ?? key;
}

/** Start of the range, to the minute, so the API cache can be reused. */
function since(hours: number): string {
  const t = new Date(Date.now() - hours * 3600_000);
  t.setUTCSeconds(0, 0);
  return t.toISOString();
}
