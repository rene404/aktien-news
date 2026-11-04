import Link from "next/link";
import type { TickerCount } from "@/lib/api";

type Props = {
  items: TickerCount[];
  /** Accessible name of the list. */
  label: string;
  /** Ticker and count only, for a row above the feed. */
  compact?: boolean;
};

// Stocks as chips, each linked to its page: name, ticker and number of news.
// Stocks without news are dimmed.
export default function TickerChips({ items, label, compact }: Props) {
  return (
    <ul className={compact ? "tickers compact" : "tickers"} aria-label={label}>
      {items.map((t) => (
        <li key={t.ticker}>
          <Link
            className={t.count === 0 ? "tchip zero" : "tchip"}
            href={`/aktie/${encodeURIComponent(t.ticker)}`}
            aria-label={`${t.name} (${t.ticker}), ${mentions(t.count)}`}
            title={compact ? t.name : undefined}
          >
            {!compact && <span className="tchip-name">{t.name}</span>}
            <span className="tchip-ticker">{t.ticker}</span>
            <span className="tchip-count">{t.count}</span>
          </Link>
        </li>
      ))}
    </ul>
  );
}

function mentions(n: number): string {
  if (n === 0) return "keine Meldungen";
  return n === 1 ? "1 Meldung" : `${n} Meldungen`;
}
