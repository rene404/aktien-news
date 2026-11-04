"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import type { NewsItem, NewsPage } from "@/lib/api";
import { PAGE_SIZE } from "@/lib/filters";
import { mergeNews, unseen } from "@/lib/news";
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

/** How often the list asks for news newer than it shows; matches the 60 s cache. */
export const NEW_ITEMS_POLL_MS = 60_000;

export default function InfiniteNews({ items: first, nextCursor, query, moreHref, hideTicker, now }: Props) {
  const [items, setItems] = useState(first);
  const [cursor, setCursor] = useState(nextCursor);
  const [status, setStatus] = useState<"idle" | "loading" | "error">("idle");
  const sentinel = useRef<HTMLDivElement>(null);
  // News that came in after the page was loaded; shown on request, not pushed into the list.
  const [fresh, setFresh] = useState<NewsItem[]>([]);
  const shown = useRef(items);
  useEffect(() => {
    shown.current = items;
  }, [items]);

  // While the tab is visible, ask for the first page now and then. Failures stay silent.
  useEffect(() => {
    const id = setInterval(async () => {
      if (document.visibilityState !== "visible") return;
      try {
        const res = await fetch(`/feed?${query}`);
        if (!res.ok) return;
        const page = (await res.json()) as NewsPage;
        setFresh(unseen(shown.current, page.items));
      } catch {
        // offline: try again next time
      }
    }, NEW_ITEMS_POLL_MS);
    return () => clearInterval(id);
  }, [query]);

  const showFresh = () => {
    setItems((prev) => mergeNews(prev, fresh));
    setFresh([]);
    window.scrollTo({ top: 0, behavior: "smooth" });
  };
  const freshLabel =
    fresh.length === 0
      ? ""
      : `${fresh.length >= PAGE_SIZE ? `${PAGE_SIZE}+` : fresh.length} ${fresh.length === 1 ? "neue Meldung" : "neue Meldungen"}`;

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
      {freshLabel && (
        <button type="button" className="new-items" onClick={showFresh}>
          <span aria-hidden="true">↑</span> {freshLabel}
        </button>
      )}
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
        {status === "loading" ? "Ältere Meldungen werden geladen" : freshLabel}
      </p>
    </>
  );
}
