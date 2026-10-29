import type { NewsParams } from "./api";
import { parseFilters, toNewsParams, type Filters } from "./filters";

// Single stocks have few headlines, so the stock page shows all of them by
// default, filings included.
export const STOCK_DEFAULT_RANGE = "all";

const tickerPattern = /^[A-Za-z0-9.-]{1,12}$/;

/** API params for a stock page. */
export function stockNewsParams(f: Filters, ticker: string, now: Date): NewsParams {
  const params = toNewsParams({ ...f, section: "" }, now);
  delete params.kind;
  return { ...params, ticker };
}

/** The query the client sends to /feed for further pages (cursor appended there). */
export function feedQuery(f: Filters, ticker?: string): string {
  const qs = new URLSearchParams();
  if (f.section) qs.set("section", f.section);
  if (f.source) qs.set("source", f.source);
  qs.set("range", f.range);
  if (ticker) qs.set("ticker", ticker);
  return qs.toString();
}

/** Reads a /feed query back into API params; null for a malformed ticker. */
export function feedParams(sp: URLSearchParams, now: Date): NewsParams | null {
  const raw = Object.fromEntries(sp);
  const ticker = sp.get("ticker");
  if (ticker === null) return toNewsParams(parseFilters(raw), now);
  if (!tickerPattern.test(ticker)) return null;
  return stockNewsParams(parseFilters(raw, STOCK_DEFAULT_RANGE), ticker, now);
}
