import Link from "next/link";
import { filterHref, SECTIONS, type Filters } from "@/lib/filters";

// Plain links, so the sections work without JavaScript. Switching a section
// keeps the range but drops the source and the cursor.
export default function SectionTabs({ value }: { value: Filters }) {
  return (
    <nav className="tabs" aria-label="Bereiche">
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
