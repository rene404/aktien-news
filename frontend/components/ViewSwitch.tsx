import Link from "next/link";
import { filterHref, type Filters } from "@/lib/filters";

const VIEWS = [
  { key: "feed", label: "Feed", path: "/" },
  { key: "sources", label: "Nach Quelle", path: "/quellen" },
] as const;

/** Feed (newest first) or one tile per source; section and range carry over, the source pick does not. */
export default function ViewSwitch({ current, value }: { current: (typeof VIEWS)[number]["key"]; value: Filters }) {
  return (
    <nav className="segmented view-switch" aria-label="Ansicht">
      {VIEWS.map((v) => (
        <Link
          key={v.key}
          href={filterHref(v.path, { ...value, source: "", cursor: "" })}
          aria-current={v.key === current ? "page" : undefined}
        >
          {v.label}
        </Link>
      ))}
    </nav>
  );
}
