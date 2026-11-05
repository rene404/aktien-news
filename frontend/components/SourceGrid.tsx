import type { NewsItem, Source } from "@/lib/api";
import NewsTime from "./NewsTime";
import SourceIcon from "./SourceIcon";

export type Tile = { source: Source; items: NewsItem[]; moreHref: string };

// The view by source: one tile per source with its newest titles, like a
// wall of feeds. Only title, time and link, as everywhere (AGENTS.md, legal invariants).
/** now: server time (ISO), so "today" matches between server and browser. */
export default function SourceGrid({ tiles, now }: { tiles: Tile[]; now: string }) {
  const shown = tiles.filter((t) => t.items.length > 0);
  if (shown.length === 0) return <p className="empty">Keine Meldungen für diese Auswahl.</p>;
  return (
    <div className="source-grid">
      {shown.map(({ source, items, moreHref }) => (
        <section key={source.key} className="tile" aria-labelledby={`tile-${source.key}`}>
          <h2 id={`tile-${source.key}`}>
            <SourceIcon sourceKey={source.key} name={source.name} />
            {source.name}
          </h2>
          <ol>
            {items.map((item) => (
              <li key={item.id}>
                <NewsTime iso={item.published_at} showDay serverNow={now} />
                <a className="title" href={item.url} target="_blank" rel="noopener noreferrer">
                  {item.title}{" "}
                  <span className="ext" aria-hidden="true">
                    ↗
                  </span>
                  <span className="sr-only">(öffnet in neuem Fenster)</span>
                </a>
              </li>
            ))}
          </ol>
          <a className="tile-more" href={moreHref}>
            Alle Meldungen <span className="sr-only">von {source.name}</span>
          </a>
        </section>
      ))}
    </div>
  );
}
