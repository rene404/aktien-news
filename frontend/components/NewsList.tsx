import Link from "next/link";
import type { NewsItem, SourceKind } from "@/lib/api";
import { dayKey, dayLabel, formatClock, formatTime } from "@/lib/format";

type Props = {
  items: NewsItem[];
  /** On a stock page, the chip of that stock is redundant. */
  hideTicker?: string;
  /** Reference for "Heute"/"Gestern". */
  now?: Date;
};

// The kind of the source, not of the article: plain news gets no badge.
export const KIND_LABELS: Partial<Record<SourceKind, string>> = {
  adhoc: "Ad-hoc",
  press_release: "Presse",
  ir: "IR",
  filing: "Filing",
};

// Renders only what the public API delivers: title (linked to the original
// source), source, time and tickers. No snippets — see AGENTS.md, legal invariants.
export default function NewsList({ items, hideTicker, now = new Date() }: Props) {
  if (items.length === 0) {
    return <p className="empty">Keine Meldungen für diese Auswahl.</p>;
  }

  const days: { key: string; items: NewsItem[] }[] = [];
  for (const item of items) {
    const key = dayKey(item.published_at);
    if (days.at(-1)?.key !== key) days.push({ key, items: [] });
    days.at(-1)!.items.push(item);
  }

  return (
    <>
      {days.map((day) => {
        const label = dayLabel(day.key, now);
        return (
          <section key={day.key} className="day" aria-label={label}>
            <h2>{label}</h2>
            <ol className="news">
              {day.items.map((item) => (
                <NewsRow key={item.id} item={item} hideTicker={hideTicker} />
              ))}
            </ol>
          </section>
        );
      })}
    </>
  );
}

function NewsRow({ item, hideTicker }: { item: NewsItem; hideTicker?: string }) {
  const kind = KIND_LABELS[item.source.kind];
  return (
    <li className={item.source.kind === "adhoc" ? "adhoc" : undefined}>
      <time dateTime={item.published_at} title={formatTime(item.published_at)}>
        {formatClock(item.published_at)}
      </time>
      <div>
        <a className="title" href={item.url} target="_blank" rel="noopener noreferrer">
          {item.title}{" "}
          <span className="ext" aria-hidden="true">
            ↗
          </span>
          <span className="sr-only">(öffnet in neuem Fenster)</span>
        </a>
        <div className="meta">
          {kind && <span className={`badge badge-${item.source.kind}`}>{kind}</span>}
          <span>{item.source.name}</span>
          {item.tickers
            .filter((t) => t !== hideTicker)
            .map((t) => (
              <Link key={t} className="chip" href={`/aktie/${encodeURIComponent(t)}`}>
                {t}
              </Link>
            ))}
        </div>
      </div>
    </li>
  );
}
