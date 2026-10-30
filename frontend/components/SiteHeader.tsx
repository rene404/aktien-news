"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";

/** How long the page must rest before the header slides back in. */
export const SCROLL_IDLE_MS = 200;

// A sticky header that makes room while reading: it slides away when
// scrolling down and comes back when scrolling stops or goes up. It stays
// near the top of the page and while the search has focus. Without
// JavaScript it is a plain sticky header.
export default function SiteHeader() {
  const ref = useRef<HTMLElement>(null);
  const [hidden, setHidden] = useState(false);
  const [floating, setFloating] = useState(false);

  useEffect(() => {
    let lastY = window.scrollY;
    let idle: ReturnType<typeof setTimeout> | undefined;
    const onScroll = () => {
      const y = window.scrollY;
      const header = ref.current;
      const pinned = !header || y <= header.offsetHeight || header.contains(document.activeElement);
      setHidden(!pinned && y > lastY);
      setFloating(y > 0);
      lastY = y;
      clearTimeout(idle);
      idle = setTimeout(() => setHidden(false), SCROLL_IDLE_MS);
    };
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => {
      window.removeEventListener("scroll", onScroll);
      clearTimeout(idle);
    };
  }, []);

  return (
    <header
      ref={ref}
      className="site"
      data-hidden={hidden ? "" : undefined}
      data-floating={floating ? "" : undefined}
      onFocusCapture={() => setHidden(false)}
    >
      <div className="wrap">
        <Link className="brand" href="/">
          aktien-news<span>.at</span>
        </Link>
        <form className="search" action="/suche" method="get" role="search">
          <input name="q" type="search" placeholder="Aktie, Ticker oder ISIN" aria-label="Aktie suchen" required maxLength={64} />
          <button type="submit">Suchen</button>
        </form>
      </div>
    </header>
  );
}
