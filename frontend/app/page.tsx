import Filters from "@/components/Filters";
import InfiniteNews from "@/components/InfiniteNews";
import SectionTabs from "@/components/SectionTabs";
import { getNews, getSources } from "@/lib/api";
import { feedQuery } from "@/lib/feed";
import { filterHref, parseFilters, toNewsParams, type SearchParams } from "@/lib/filters";

export default async function Home({ searchParams }: { searchParams: Promise<SearchParams> }) {
  const filters = parseFilters(await searchParams);
  const now = new Date();
  const [page, sources] = await Promise.all([getNews(toNewsParams(filters, now)), getSources()]);
  const query = feedQuery(filters);

  return (
    <>
      <h1>Aktuelle Meldungen</h1>
      <p className="sub">Ad-hoc-Mitteilungen und Nachrichten zu ATX-, DAX- und US-Aktien.</p>
      <SectionTabs value={filters} />
      <Filters sources={sources} value={filters} />
      {/* Keyed so a new filter starts a fresh list instead of appending to the old one. */}
      <InfiniteNews
        key={`${query}:${page.next_cursor}`}
        items={page.items}
        nextCursor={page.next_cursor}
        query={query}
        moreHref={filterHref("/", { ...filters, cursor: page.next_cursor ?? "" }, { cursor: true })}
        now={now.toISOString()}
      />
    </>
  );
}
