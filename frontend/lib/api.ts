// Server-side client for the Go REST API. Only called from server
// components; the browser never talks to the API directly.

export type SourceKind = "adhoc" | "press_release" | "news_api" | "ir" | "filing";

export type NewsItem = {
  id: number;
  title: string;
  url: string;
  source: { key: string; name: string; kind: SourceKind };
  published_at: string;
  tickers: string[];
  /** The same news at other sources, oldest first (API: at most 5). */
  also?: AlsoAt[];
  /** Further items of the same source that this row stands for (e.g. several directors' dealings). */
  same_source?: number;
};

export type AlsoAt = { title: string; url: string; source: { key: string; name: string; kind: SourceKind }; published_at: string };

export type NewsPage = { items: NewsItem[]; next_cursor: string | null };

export type Instrument = { ticker: string; mic: string; name: string; isin?: string };

export type InstrumentDetail = Instrument & { country: string; indices: string[] };

export type Source = {
  key: string;
  name: string;
  market: string;
  kind: SourceKind;
  /** Last successful fetch (ISO, UTC); null before the first. */
  last_ok_at: string | null;
};

/** A stock with the number of news items it appears in. */
export type TickerCount = { ticker: string; name: string; count: number };

export type NewsParams = {
  ticker?: string;
  index?: string;
  source?: string;
  /** Comma-separated source kinds. */
  kind?: string;
  from?: string;
  to?: string;
  cursor?: string;
  limit?: number;
};

const API_URL = process.env.API_URL ?? "http://localhost:8090";

// Mirrors the API's own Cache-Control (60 s).
const REVALIDATE = 60;

/** "?a=1&b=2" for the set parameters, "" when none is set. */
function query(params: Record<string, string | number | undefined>): string {
  const qs = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== "") qs.set(k, String(v));
  }
  const s = qs.toString();
  return s ? `?${s}` : "";
}

export function newsUrl(base: string, params: NewsParams): string {
  return `${base}/api/v1/news${query(params)}`;
}

async function getJSON<T>(url: string): Promise<{ status: number; body: T | null }> {
  const res = await fetch(url, { next: { revalidate: REVALIDATE } });
  if (res.status === 404 || res.status === 400) return { status: res.status, body: null };
  if (!res.ok) throw new Error(`API ${res.status} for ${url}`);
  return { status: res.status, body: (await res.json()) as T };
}

export async function getNews(params: NewsParams): Promise<NewsPage> {
  const { status, body } = await getJSON<NewsPage>(newsUrl(API_URL, params));
  if (!body) throw new Error(`API ${status} for news`);
  return body;
}

/** null when the ticker is unknown or malformed. */
export async function getInstrument(ticker: string): Promise<InstrumentDetail | null> {
  const { body } = await getJSON<InstrumentDetail>(`${API_URL}/api/v1/instruments/${encodeURIComponent(ticker)}`);
  return body;
}

export async function searchInstruments(q: string): Promise<Instrument[]> {
  const { body } = await getJSON<{ items: Instrument[] }>(`${API_URL}/api/v1/instruments?q=${encodeURIComponent(q)}`);
  return body?.items ?? [];
}

export async function getSources(): Promise<Source[]> {
  const { body } = await getJSON<{ items: Source[] }>(`${API_URL}/api/v1/sources`);
  return body?.items ?? [];
}

/**
 * The most-mentioned stocks for the feed's filters. Empty on any error: the
 * row only decorates the page and must never break it.
 */
export async function getTopTickers(params: { index?: string; kind?: string; from: string; limit?: number }): Promise<TickerCount[]> {
  try {
    const { body } = await getJSON<{ items: TickerCount[] }>(`${API_URL}/api/v1/tickers/top${query(params)}`);
    return body?.items ?? [];
  } catch {
    return [];
  }
}

/** Every member of an index with its count; null for an unknown index. */
export async function getIndexInstruments(index: string, params: { from?: string; kind?: string }): Promise<TickerCount[] | null> {
  const { body } = await getJSON<{ items: TickerCount[] }>(
    `${API_URL}/api/v1/indices/${encodeURIComponent(index)}/instruments${query(params)}`,
  );
  return body?.items ?? null;
}
