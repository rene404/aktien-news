import { filterHref, SECTIONS, type Filters } from "@/lib/filters";
import TabNav from "./TabNav";

// The sections of the start page. Switching a section keeps the range but
// drops the source and the cursor.
export default function SectionTabs({ value }: { value: Filters }) {
  return (
    <TabNav
      label="Bereiche"
      tabs={SECTIONS.map((s) => ({
        href: filterHref("/", { ...value, section: s.key, source: "", cursor: "" }),
        label: s.label,
        current: s.key === value.section,
      }))}
    />
  );
}
