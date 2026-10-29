import type { Source } from "@/lib/api";
import { RANGES, type Filters as FilterValues } from "@/lib/filters";

type Props = {
  sources: Source[];
  value: FilterValues;
};

// A plain GET form: works without JavaScript and keeps filters in the URL.
// The section rides along hidden; the pagination cursor is deliberately not carried over.
export default function Filters({ sources, value }: Props) {
  return (
    <form className="filters" method="get">
      {value.section && <input type="hidden" name="section" value={value.section} />}
      <label>
        Quelle
        <select name="source" defaultValue={value.source}>
          <option value="">Alle</option>
          {sources.map((s) => (
            <option key={s.key} value={s.key}>
              {s.name}
            </option>
          ))}
        </select>
      </label>
      <label>
        Zeitraum
        <select name="range" defaultValue={value.range}>
          {RANGES.map((r) => (
            <option key={r.key} value={r.key}>
              {r.label}
            </option>
          ))}
        </select>
      </label>
      <button type="submit">Filtern</button>
    </form>
  );
}
