"use client";

import Link from "next/link";
import { useRef } from "react";
import { filterHref, SECTIONS, type Filters } from "@/lib/filters";
import { useScrollRow } from "./useScrollRow";

// Plain links, so the sections work without JavaScript. Switching a section
// keeps the range but drops the source and the cursor. On narrow screens the
// row scrolls sideways with faded edges (useScrollRow).
export default function SectionTabs({ value }: { value: Filters }) {
  const ref = useRef<HTMLElement>(null);
  useScrollRow(ref);
  return (
    <nav ref={ref} className="tabs scroll-row" aria-label="Bereiche">
      {SECTIONS.map((s) => (
        <Link
          key={s.key}
          href={filterHref("/", { ...value, section: s.key, source: "", cursor: "" })}
          aria-current={s.key === value.section ? "page" : undefined}
        >
          {s.label}
        </Link>
      ))}
    </nav>
  );
}
