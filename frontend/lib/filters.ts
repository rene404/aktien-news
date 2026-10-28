import type { NewsParams } from "./api";

export const INDICES = [
  { key: "ATX", label: "ATX" },
  { key: "DAX", label: "DAX" },
  { key: "SP500", label: "S&P 500" },
] as const;

export const RANGES = [
  { key: "24h", label: "24 Stunden", hours: 24 },
  { key: "7d", label: "7 Tage", hours: 24 * 7 },
  { key: "30d", label: "30 Tage", hours: 24 * 30 },
  { key: "all", label: "Alle", hours: 0 },
] as const;

export type Filters = { index: string; source: string; range: string; cursor: string };

export type SearchParams = Record<string, string | string[] | undefined>;

const PAGE_SIZE = 25;

function first(v: string | string[] | undefined): string {
  return (Array.isArray(v) ? v[0] : v) ?? "";
}

/** Reads the URL's filter params; anything unexpected falls back to the default. */
export function parseFilters(sp: SearchParams): Filters {
  const index = first(sp.index);
  const source = first(sp.source);
  const range = first(sp.range);
  const cursor = first(sp.cursor);
  return {
    index: INDICES.some((i) => i.key === index) ? index : "",
    source: /^[a-z0-9_]{1,32}$/.test(source) ? source : "",
    range: RANGES.some((r) => r.key === range) ? range : "7d",
    cursor: /^[A-Za-z0-9_-]{1,200}$/.test(cursor) ? cursor : "",
  };
}

export function toNewsParams(f: Filters, now: Date): NewsParams {
  const params: NewsParams = {};
  if (f.index) params.index = f.index;
  if (f.source) params.source = f.source;
  const hours = RANGES.find((r) => r.key === f.range)?.hours ?? 0;
  if (hours > 0) params.from = new Date(now.getTime() - hours * 3600_000).toISOString();
  if (f.cursor) params.cursor = f.cursor;
  params.limit = PAGE_SIZE;
  return params;
}
