import Filters from "@/components/Filters";
import MoreLink from "@/components/MoreLink";
import NewsList from "@/components/NewsList";
import { getNews, getSources } from "@/lib/api";
import { parseFilters, toNewsParams, type SearchParams } from "@/lib/filters";

export default async function Home({ searchParams }: { searchParams: Promise<SearchParams> }) {
  const filters = parseFilters(await searchParams);
  const [page, sources] = await Promise.all([getNews(toNewsParams(filters, new Date())), getSources()]);

  return (
    <>
      <h1>Aktuelle Meldungen</h1>
      <p className="sub">Ad-hoc-Mitteilungen und Nachrichten zu ATX-, DAX- und US-Aktien.</p>
      <Filters sources={sources} value={filters} />
      <NewsList items={page.items} />
      <MoreLink path="/" filters={filters} cursor={page.next_cursor} />
    </>
  );
}
