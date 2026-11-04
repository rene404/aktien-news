import type { TickerCount } from "./api";
import { RANGES, SECTIONS, type SearchParams } from "./filters";

// The stock list (/aktien): the members of an index with their news count.

/** The indices with a section of their own, in the same order. */
export const LIST_INDICES = SECTIONS.flatMap((s) => ("index" in s ? [{ key: s.index, label: s.label }] : []));

/** A list counts recent news: no "Alle"/30 days, that says little about today. */
export const LIST_RANGES = RANGES.filter((r) => r.key === "24h" || r.key === "7d");

export const LIST_SORTS = [
  { key: "count", label: "Meiste Meldungen" },
  { key: "name", label: "A–Z" },
] as const;

export type StockList = { index: string; range: string; sort: string };

const DEFAULTS: StockList = { index: "ATX", range: "7d", sort: "count" };

function first(v: string | string[] | undefined): string {
  return (Array.isArray(v) ? v[0] : v) ?? "";
}

export function parseStockList(sp: SearchParams): StockList {
  const index = first(sp.index);
  const range = first(sp.range);
  const sort = first(sp.sort);
  return {
    index: LIST_INDICES.some((i) => i.key === index) ? index : DEFAULTS.index,
    range: LIST_RANGES.some((r) => r.key === range) ? range : DEFAULTS.range,
    sort: LIST_SORTS.some((s) => s.key === sort) ? sort : DEFAULTS.sort,
  };
}

export function stockListHref(l: StockList): string {
  const qs = new URLSearchParams();
  for (const k of ["index", "range", "sort"] as const) {
    if (l[k] !== DEFAULTS[k]) qs.set(k, l[k]);
  }
  const s = qs.toString();
  return s ? `/aktien?${s}` : "/aktien";
}

const byName = new Intl.Collator("de-AT", { sensitivity: "base" });

/** Sorted copy: most news first (ties by name), or by name. */
export function sortTickers(items: TickerCount[], sort: string): TickerCount[] {
  const name = (a: TickerCount, b: TickerCount) => byName.compare(a.name, b.name);
  return [...items].sort(sort === "name" ? name : (a, b) => b.count - a.count || name(a, b));
}
