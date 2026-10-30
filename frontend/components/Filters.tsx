"use client";

import Form from "next/form";
import type { Source } from "@/lib/api";
import { RANGES, type Filters as FilterValues } from "@/lib/filters";

type Props = {
  /** The page to filter, e.g. "/" or "/aktie/EBS". */
  path: string;
  sources: Source[];
  value: FilterValues;
};

// A GET form: keeps filters in the URL and works without JavaScript. With
// JavaScript, next/form navigates client-side and a changed select applies at once.
// The section rides along hidden; the pagination cursor is deliberately not carried over.
export default function Filters({ path, sources, value }: Props) {
  const submit = (e: React.ChangeEvent<HTMLSelectElement>) => e.currentTarget.form?.requestSubmit();
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
      <button type="submit">Filtern</button>
    </Form>
  );
}
