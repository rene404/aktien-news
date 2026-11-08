import type { NewsParams, Source } from "./api";

// Source kinds of the API. Filings (SEC 8-K/6-K, OeKB, CNMV, AMF, AFM) come in
// by the hundred and would bury everything else, so they get a section of
// their own: "Pflichtmeldungen".
export const NEWS_KINDS = ["adhoc", "press_release", "news_api", "ir"];

export const SECTIONS = [
  { key: "", label: "Alle", kinds: NEWS_KINDS },
  { key: "ATX", label: "ATX", index: "ATX", kinds: NEWS_KINDS, market: "AT" },
  { key: "DAX", label: "DAX", index: "DAX", kinds: NEWS_KINDS, market: "DE" },
  { key: "CAC40", label: "CAC 40", index: "CAC40", kinds: NEWS_KINDS, market: "FR" },
  { key: "IBEX35", label: "IBEX 35", index: "IBEX35", kinds: NEWS_KINDS, market: "ES" },
  { key: "AEX", label: "AEX", index: "AEX", kinds: NEWS_KINDS, market: "NL" },
  { key: "SP500", label: "S&P 500", index: "SP500", kinds: NEWS_KINDS, market: "US" },
  { key: "adhoc", label: "Ad-hoc", kinds: ["adhoc"] },
  { key: "filings", label: "Pflichtmeldungen", kinds: ["filing"] },
] as const satisfies readonly { key: string; label: string; index?: string; kinds: readonly string[]; market?: string }[];

/** Exchanges (MIC) as readers know them, and the source market they belong to. */
export const EXCHANGES: Record<string, { name: string; market: string }> = {
  XWBO: { name: "Wiener Börse", market: "AT" },
  XETR: { name: "Xetra", market: "DE" },
  XNYS: { name: "NYSE", market: "US" },
  XNAS: { name: "Nasdaq", market: "US" },
  XPAR: { name: "Euronext Paris", market: "FR" },
  XMAD: { name: "Bolsa de Madrid", market: "ES" },
  XAMS: { name: "Euronext Amsterdam", market: "NL" },
};

/**
 * The sources worth offering: those of the section's kinds and market (a
 * section page) or of the exchange's market (a stock page). Sources covering
 * several markets ("MULTI") always fit a market. A source already picked in the
 * URL stays in the list, so the select shows what is filtered.
 */
export function sourcesFor(sources: Source[], opts: { section?: string; mic?: string; selected?: string }): Source[] {
  let kinds: readonly string[] | undefined;
  let market: string | undefined;
  if (opts.mic !== undefined) {
    market = EXCHANGES[opts.mic]?.market;
  } else {
    const section = SECTIONS.find((s) => s.key === (opts.section ?? "")) ?? SECTIONS[0];
    kinds = section.kinds;
    market = "market" in section ? section.market : undefined;
  }
  return sources.filter(
    (s) =>
      s.key === opts.selected ||
      ((!kinds || kinds.includes(s.kind)) && (!market || s.market === market || s.market === "MULTI")),
  );
}

/** The sources behind the list on screen: all offered ones, or just the one picked in the URL. */
export function shownSources(sources: Source[], selected: string): Source[] {
  return selected ? sources.filter((s) => s.key === selected) : sources;
}

export const RANGES = [
  { key: "24h", label: "24 Stunden", hours: 24 },
  { key: "7d", label: "7 Tage", hours: 24 * 7 },
  { key: "30d", label: "30 Tage", hours: 24 * 30 },
  { key: "all", label: "Alle", hours: 0 },
] as const;

/** Range of the start page when the URL names none. */
export const DEFAULT_RANGE = "7d";

/** `adhoc` (stock page only): ad-hoc news only; present only when on. */
export type Filters = { section: string; source: string; range: string; cursor: string; adhoc?: boolean };

export type SearchParams = Record<string, string | string[] | undefined>;

export const PAGE_SIZE = 25;

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
    ...(first(sp.adhoc) === "1" ? { adhoc: true } : {}),
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

/** News per tile in the view by source (/quellen). */
export const TILE_SIZE = 5;

/** The newest few news of one source, within the section's index and the range. */
export function tileParams(f: Filters, sourceKey: string, now: Date): NewsParams {
  const { index, from } = toNewsParams({ ...f, source: sourceKey, cursor: "" }, now);
  return { ...(index ? { index } : {}), source: sourceKey, ...(from ? { from } : {}), limit: TILE_SIZE };
}

/** A link to `path` with these filters; defaults are left out of the URL. */
export function filterHref(path: string, f: Filters, opts: { cursor?: boolean; defaultRange?: string } = {}): string {
  const qs = new URLSearchParams();
  if (f.section) qs.set("section", f.section);
  if (f.source) qs.set("source", f.source);
  if (f.range !== (opts.defaultRange ?? DEFAULT_RANGE)) qs.set("range", f.range);
  if (f.adhoc) qs.set("adhoc", "1");
  if (opts.cursor && f.cursor) qs.set("cursor", f.cursor);
  const s = qs.toString();
  return s ? `${path}?${s}` : path;
}
