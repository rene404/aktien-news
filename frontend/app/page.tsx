import Filters from "@/components/Filters";
import MoreLink from "@/components/MoreLink";
import NewsList from "@/components/NewsList";
import SectionTabs from "@/components/SectionTabs";
import { getNews, getSources } from "@/lib/api";
import { parseFilters, toNewsParams, type SearchParams } from "@/lib/filters";

export default async function Home({ searchParams }: { searchParams: Promise<SearchParams> }) {
  const filters = parseFilters(await searchParams);
  const now = new Date();
  const [page, sources] = await Promise.all([getNews(toNewsParams(filters, now)), getSources()]);

  return (
    <>
      <h1>Aktuelle Meldungen</h1>
      <p className="sub">Ad-hoc-Mitteilungen und Nachrichten zu ATX-, DAX- und US-Aktien.</p>
      <SectionTabs value={filters} />
      <Filters sources={sources} value={filters} />
      <NewsList items={page.items} now={now} />
      <MoreLink path="/" filters={filters} cursor={page.next_cursor} />
    </>
  );
}
