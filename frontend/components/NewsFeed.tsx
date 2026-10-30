import { getNews } from "@/lib/api";
import { feedQuery, STOCK_DEFAULT_RANGE, stockNewsParams } from "@/lib/feed";
import { DEFAULT_RANGE, filterHref, toNewsParams, type Filters } from "@/lib/filters";
import InfiniteNews from "./InfiniteNews";

type Props = {
  filters: Filters;
  /** The page the list lives on, for the no-JavaScript link to the next page. */
  path: string;
  /** Set on a stock page: news of this stock only. */
  ticker?: string;
};

// The news list of the start page and the stock pages: the first page comes
// from the API here, further pages from /feed in the browser.
export default async function NewsFeed({ filters, path, ticker }: Props) {
  const now = new Date();
  const page = await getNews(ticker ? stockNewsParams(filters, ticker, now) : toNewsParams(filters, now));
  const query = feedQuery(filters, ticker);
  const defaultRange = ticker ? STOCK_DEFAULT_RANGE : DEFAULT_RANGE;

  return (
    // Keyed so fresh data after revalidation starts a new list instead of appending to the old one.
    <InfiniteNews
      key={`${query}:${page.next_cursor}`}
      items={page.items}
      nextCursor={page.next_cursor}
      query={query}
      moreHref={filterHref(path, { ...filters, cursor: page.next_cursor ?? "" }, { cursor: true, defaultRange })}
      hideTicker={ticker}
      now={now.toISOString()}
    />
  );
}
