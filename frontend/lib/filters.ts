import type { NewsParams } from "./api";

// Source kinds of the API. Filings (SEC 8-K/6-K) come in by the hundred and
// would bury everything else, so they get a section of their own.
const NEWS_KINDS = ["adhoc", "press_release", "news_api", "ir"];

export const SECTIONS = [
  { key: "", label: "Alle", kinds: NEWS_KINDS },
  { key: "ATX", label: "ATX", index: "ATX", kinds: NEWS_KINDS },
  { key: "DAX", label: "DAX", index: "DAX", kinds: NEWS_KINDS },
  { key: "SP500", label: "S&P 500", index: "SP500", kinds: NEWS_KINDS },
  { key: "adhoc", label: "Ad-hoc", kinds: ["adhoc"] },
  { key: "filings", label: "SEC-Filings", kinds: ["filing"] },
] as const satisfies readonly { key: string; label: string; index?: string; kinds: readonly string[] }[];

export const RANGES = [
  { key: "24h", label: "24 Stunden", hours: 24 },
  { key: "7d", label: "7 Tage", hours: 24 * 7 },
  { key: "30d", label: "30 Tage", hours: 24 * 30 },
  { key: "all", label: "Alle", hours: 0 },
] as const;

/** Range of the start page when the URL names none. */
export const DEFAULT_RANGE = "7d";

export type Filters = { section: string; source: string; range: string; cursor: string };

export type SearchParams = Record<string, string | string[] | undefined>;

const PAGE_SIZE = 25;

function first(v: string | string[] | undefined): string {
  return (Array.isArray(v) ? v[0] : v) ?? "";
}

/** Reads the URL's filter params; anything unexpected falls back to the default. */
export function parseFilters(sp: SearchParams, defaultRange = DEFAULT_RANGE): Filters {
  const section = first(sp.section) || first(sp.index); // ?index= is the pre-section URL
  const source = first(sp.source);
  const range = first(sp.range);
  const cursor = first(sp.cursor);
  return {
    section: SECTIONS.some((s) => s.key === section) ? section : "",
    source: /^[a-z0-9_]{1,32}$/.test(source) ? source : "",
    range: RANGES.some((r) => r.key === range) ? range : defaultRange,
    cursor: /^[A-Za-z0-9_-]{1,200}$/.test(cursor) ? cursor : "",
  };
}

export function toNewsParams(f: Filters, now: Date): NewsParams {
  const params: NewsParams = {};
  const section = SECTIONS.find((s) => s.key === f.section) ?? SECTIONS[0];
  if ("index" in section) params.index = section.index;
  // A source picked by hand is shown even if its kind is outside the section.
  if (f.source) params.source = f.source;
  else params.kind = section.kinds.join(",");
  const hours = RANGES.find((r) => r.key === f.range)?.hours ?? 0;
  if (hours > 0) params.from = new Date(now.getTime() - hours * 3600_000).toISOString();
  if (f.cursor) params.cursor = f.cursor;
  params.limit = PAGE_SIZE;
  return params;
}

/** A link to `path` with these filters; defaults are left out of the URL. */
export function filterHref(path: string, f: Filters, opts: { cursor?: boolean; defaultRange?: string } = {}): string {
  const qs = new URLSearchParams();
  if (f.section) qs.set("section", f.section);
  if (f.source) qs.set("source", f.source);
  if (f.range !== (opts.defaultRange ?? DEFAULT_RANGE)) qs.set("range", f.range);
  if (opts.cursor && f.cursor) qs.set("cursor", f.cursor);
  const s = qs.toString();
  return s ? `${path}?${s}` : path;
}
