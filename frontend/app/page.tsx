import { Suspense } from "react";
import Filters from "@/components/Filters";
import NewsFeed from "@/components/NewsFeed";
import NewsSkeleton from "@/components/NewsSkeleton";
import SectionTabs from "@/components/SectionTabs";
import { getSources } from "@/lib/api";
import { feedQuery } from "@/lib/feed";
import { parseFilters, sourcesFor, type SearchParams } from "@/lib/filters";

export default async function Home({ searchParams }: { searchParams: Promise<SearchParams> }) {
  const filters = parseFilters(await searchParams);
  const sources = sourcesFor(await getSources(), { section: filters.section, selected: filters.source });

  return (
    <>
      <h1>Aktuelle Meldungen</h1>
      <p className="sub">Ad-hoc-Mitteilungen und Nachrichten zu ATX-, DAX- und US-Aktien.</p>
      <SectionTabs value={filters} />
      <Filters path="/" sources={sources} value={filters} />
      {/* Keyed so a new filter shows the placeholder instead of the old list. */}
      <Suspense key={feedQuery(filters)} fallback={<NewsSkeleton />}>
        <NewsFeed filters={filters} path="/" />
      </Suspense>
    </>
  );
}
