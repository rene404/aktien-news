import type { Metadata } from "next";
import FreshnessLine from "@/components/FreshnessLine";
import SectionTabs from "@/components/SectionTabs";
import SourceGrid, { type Tile } from "@/components/SourceGrid";
import ViewSwitch from "@/components/ViewSwitch";
import { getNews, getSources } from "@/lib/api";
import { filterHref, parseFilters, sourcesFor, tileParams, type SearchParams } from "@/lib/filters";

export const metadata: Metadata = {
  title: "Meldungen nach Quelle",
  description: "Die neuesten Meldungen zu österreichischen, deutschen, europäischen und US-Aktien, nach Quelle geordnet – mit Link zur Originalquelle.",
};

export default async function BySourcePage({ searchParams }: { searchParams: Promise<SearchParams> }) {
  const filters = { ...parseFilters(await searchParams), source: "", cursor: "" };
  const sources = sourcesFor(await getSources(), { section: filters.section });
  const now = new Date();
  // One request per source, in parallel; a failing source just shows no tile.
  const tiles: Tile[] = await Promise.all(
    sources.map(async (source) => ({
      source,
      items: await getNews(tileParams(filters, source.key, now)).then(
        (p) => p.items,
        () => [],
      ),
      moreHref: filterHref("/", { ...filters, source: source.key }),
    })),
  );

  return (
    <>
      <h1>Meldungen nach Quelle</h1>
      <p className="sub">Die neuesten Meldungen jeder Quelle auf einen Blick.</p>
      <SectionTabs value={filters} path="/quellen" />
      <ViewSwitch current="sources" value={filters} />
      <FreshnessLine sources={sources} />
      <SourceGrid tiles={tiles} now={now.toISOString()} />
    </>
  );
}
