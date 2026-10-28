import Link from "next/link";
import type { NewsItem } from "@/lib/api";
import { formatTime } from "@/lib/format";

type Props = {
  items: NewsItem[];
  /** On a stock page, the chip of that stock is redundant. */
  hideTicker?: string;
};

// Renders only what the public API delivers: title (linked to the original
// source), source, time and tickers. No snippets — see AGENTS.md, legal invariants.
export default function NewsList({ items, hideTicker }: Props) {
  if (items.length === 0) {
    return <p className="empty">Keine Meldungen für diese Auswahl.</p>;
  }
  return (
    <ol className="news">
      {items.map((item) => (
        <li key={item.id}>
          <a className="title" href={item.url} target="_blank" rel="noopener noreferrer">
            {item.title}
          </a>
          <div className="meta">
            <time dateTime={item.published_at}>{formatTime(item.published_at)}</time>
            <span aria-hidden="true"> · </span>
            <span>{item.source.name}</span>
            {item.tickers
              .filter((t) => t !== hideTicker)
              .map((t) => (
                <Link key={t} className="chip" href={`/aktie/${encodeURIComponent(t)}`}>
                  {t}
                </Link>
              ))}
          </div>
        </li>
      ))}
    </ol>
  );
}
