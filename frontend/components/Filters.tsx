"use client";

import Form from "next/form";
import { useSyncExternalStore } from "react";
import type { Source } from "@/lib/api";
import { RANGES, type Filters as FilterValues } from "@/lib/filters";

type Props = {
  /** The page to filter, e.g. "/" or "/aktie/EBS". */
  path: string;
  sources: Source[];
  value: FilterValues;
  /** Offer "Nur Ad-hoc" (stock page). */
  showAdhoc?: boolean;
};

// A GET form: keeps filters in the URL and works without JavaScript. With
// JavaScript, next/form navigates client-side and a changed select applies at once.
// The section rides along hidden; the pagination cursor is deliberately not carried over.
// The submit button is only for browsers without JavaScript: the server renders
// it, the client hides it once hydrated.
export default function Filters({ path, sources, value, showAdhoc }: Props) {
  const hydrated = useSyncExternalStore(noop, () => true, () => false);
  const submit = (e: React.ChangeEvent<HTMLSelectElement | HTMLInputElement>) => e.currentTarget.form?.requestSubmit();
  return (
    <Form className="filters" action={path}>
      {value.section && <input type="hidden" name="section" value={value.section} />}
      <label>
        Quelle
        <select name="source" defaultValue={value.source} onChange={submit}>
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
        <select name="range" defaultValue={value.range} onChange={submit}>
          {RANGES.map((r) => (
            <option key={r.key} value={r.key}>
              {r.label}
            </option>
          ))}
        </select>
      </label>
      {showAdhoc && (
        <label className="check">
          <input type="checkbox" name="adhoc" value="1" defaultChecked={value.adhoc} onChange={submit} />
          Nur Ad-hoc
        </label>
      )}
      <button type="submit" hidden={hydrated}>
        Filtern
      </button>
    </Form>
  );
}

const noop = () => () => {};
