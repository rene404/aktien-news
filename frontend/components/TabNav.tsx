"use client";

import Link from "next/link";
import { useRef } from "react";
import { useScrollRow } from "./useScrollRow";

export type Tab = { href: string; label: string; current: boolean };

// Tabs as plain links (work without JavaScript). On narrow screens the row
// scrolls sideways with faded edges (useScrollRow).
export default function TabNav({ tabs, label }: { tabs: Tab[]; label: string }) {
  const ref = useRef<HTMLElement>(null);
  useScrollRow(ref);
  return (
    <nav ref={ref} className="tabs scroll-row" aria-label={label}>
      {tabs.map((t) => (
        <Link key={t.href} href={t.href} aria-current={t.current ? "page" : undefined}>
          {t.label}
        </Link>
      ))}
    </nav>
  );
}
