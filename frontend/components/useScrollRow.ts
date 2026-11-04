"use client";

import { useEffect, type RefObject } from "react";

// A row that scrolls sideways on narrow screens (tabs, ticker chips). Marks the
// edges with more content (data-more-start / data-more-end) so CSS can fade
// them out, and brings the current item (aria-current) into view. Sets the
// attributes directly, so scrolling never re-renders.
export function useScrollRow(ref: RefObject<HTMLElement | null>) {
  useEffect(() => {
    const el = ref.current;
    if (!el) return;

    const current = el.querySelector<HTMLElement>("[aria-current]");
    if (current && current.offsetLeft + current.offsetWidth > el.clientWidth) {
      el.scrollLeft = current.offsetLeft + current.offsetWidth / 2 - el.clientWidth / 2;
    }

    const update = () => {
      const max = el.scrollWidth - el.clientWidth;
      el.toggleAttribute("data-more-start", el.scrollLeft > 1);
      el.toggleAttribute("data-more-end", el.scrollLeft < max - 1);
    };
    update();
    el.addEventListener("scroll", update, { passive: true });
    window.addEventListener("resize", update);
    return () => {
      el.removeEventListener("scroll", update);
      window.removeEventListener("resize", update);
    };
  }, [ref]);
}
