"use client";

import { useRef } from "react";
import { useScrollRow } from "./useScrollRow";

/** A row that swipes sideways with faded edges; the content stays server-rendered. */
export default function ScrollRow({ className, children }: { className?: string; children: React.ReactNode }) {
  const ref = useRef<HTMLDivElement>(null);
  useScrollRow(ref);
  return (
    <div ref={ref} className={className ? `scroll-row ${className}` : "scroll-row"}>
      {children}
    </div>
  );
}
