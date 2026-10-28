import type { Source } from "@/lib/api";
import { INDICES, RANGES, type Filters as FilterValues } from "@/lib/filters";

type Props = {
  sources: Source[];
  value: FilterValues;
  /** Hide the index select on a stock page. */
  showIndex?: boolean;
};

// A plain GET form: works without JavaScript and keeps filters in the URL.
// The pagination cursor is deliberately not carried over.
export default function Filters({ sources, value, showIndex = true }: Props) {
  return (
    <form className="filters" method="get">
      {showIndex && (
        <label>
          Index
          <select name="index" defaultValue={value.index}>
            <option value="">Alle</option>
            {INDICES.map((i) => (
              <option key={i.key} value={i.key}>
                {i.label}
              </option>
            ))}
          </select>
        </label>
      )}
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
