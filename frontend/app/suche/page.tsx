import type { Metadata } from "next";
import Link from "next/link";
import { redirect } from "next/navigation";
import { searchInstruments } from "@/lib/api";
import type { SearchParams } from "@/lib/filters";

export const metadata: Metadata = { title: "Suche", robots: { index: false } };

export default async function SearchPage({ searchParams }: { searchParams: Promise<SearchParams> }) {
  const raw = (await searchParams).q;
  const q = (Array.isArray(raw) ? raw[0] : raw)?.trim().slice(0, 64) ?? "";
  const results = q ? await searchInstruments(q) : [];

  // An exact ticker hit goes straight to the stock page.
  const exact = results.find((r) => r.ticker.toUpperCase() === q.toUpperCase());
  if (exact) redirect(`/aktie/${encodeURIComponent(exact.ticker)}`);

  return (
    <>
      <h1>Suche</h1>
      <p className="sub">{q ? `Ergebnisse für „${q}“` : "Bitte einen Suchbegriff eingeben."}</p>
      {q && results.length === 0 && <p className="empty">Keine passende Aktie gefunden.</p>}
      <ul className="results">
        {results.map((r) => (
          <li key={`${r.mic}:${r.ticker}`}>
            <Link href={`/aktie/${encodeURIComponent(r.ticker)}`}>{r.name}</Link>{" "}
            <span className="chip">{r.ticker}</span> <span className="sub">{r.isin}</span>
          </li>
        ))}
      </ul>
    </>
  );
}
