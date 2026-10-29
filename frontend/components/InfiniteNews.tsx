"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import type { NewsItem, NewsPage } from "@/lib/api";
import NewsList from "./NewsList";

type Props = {
  items: NewsItem[];
  nextCursor: string | null;
  /** Filter query for /feed, without the cursor (see lib/feed.ts). */
  query: string;
  /** The next page as a plain link: the fallback without JavaScript. */
  moreHref: string;
  hideTicker?: string;
  /** Server time as ISO string, so "Heute" matches between server and client. */
  now: string;
};

// Start loading well before the reader reaches the end of the list.
const PRELOAD_MARGIN = "800px";

export default function InfiniteNews({ items: first, nextCursor, query, moreHref, hideTicker, now }: Props) {
  const [items, setItems] = useState(first);
  const [cursor, setCursor] = useState(nextCursor);
  const [status, setStatus] = useState<"idle" | "loading" | "error">("idle");
  const sentinel = useRef<HTMLDivElement>(null);

  const loadMore = useCallback(async () => {
    if (!cursor) return;
    setStatus("loading");
    try {
      const res = await fetch(`/feed?${query}&cursor=${encodeURIComponent(cursor)}`);
      if (!res.ok) throw new Error(`feed ${res.status}`);
      const page = (await res.json()) as NewsPage;
      setItems((prev) => {
        const seen = new Set(prev.map((i) => i.id));
        return [...prev, ...page.items.filter((i) => !seen.has(i.id))];
      });
      setCursor(page.next_cursor);
      setStatus("idle");
    } catch {
      setStatus("error");
    }
  }, [cursor, query]);

  // Re-observed after every page, so a list shorter than the screen keeps filling.
  useEffect(() => {
    const el = sentinel.current;
    if (!el || !cursor || status !== "idle" || typeof IntersectionObserver === "undefined") return;
    const io = new IntersectionObserver((entries) => {
      if (entries.some((e) => e.isIntersecting)) void loadMore();
    }, { rootMargin: PRELOAD_MARGIN });
    io.observe(el);
    return () => io.disconnect();
  }, [cursor, status, loadMore]);

  return (
    <>
      <NewsList items={items} hideTicker={hideTicker} now={new Date(now)} />
      {cursor && (
        <div ref={sentinel}>
          <a
            className="more"
            href={moreHref}
            aria-disabled={status === "loading"}
            onClick={(e) => {
              e.preventDefault();
              if (status !== "loading") void loadMore();
            }}
          >
            {status === "loading" ? "Lädt …" : status === "error" ? "Laden fehlgeschlagen – erneut versuchen" : "Ältere Meldungen →"}
          </a>
        </div>
      )}
      <p className="sr-only" aria-live="polite">
        {status === "loading" ? "Ältere Meldungen werden geladen" : ""}
      </p>
    </>
  );
}
