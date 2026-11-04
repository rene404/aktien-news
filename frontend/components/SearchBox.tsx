"use client";

import { useRouter } from "next/navigation";
import { useEffect, useId, useRef, useState } from "react";

/** Pause in typing before suggestions are fetched. */
export const SUGGEST_DELAY_MS = 150;
const MIN_CHARS = 2;

type Suggestion = { ticker: string; mic: string; name: string };

// The header search: a GET form to /suche (works without JavaScript) whose
// field suggests stocks while typing (ARIA combobox). Arrow keys pick, Enter
// or a click opens the stock, Escape closes; Enter without a pick searches.
export default function SearchBox() {
  const router = useRouter();
  const listId = useId();
  const [q, setQ] = useState("");
  const [items, setItems] = useState<Suggestion[]>([]);
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(-1);
  const latest = useRef(0);

  useEffect(() => {
    const term = q.trim();
    const request = ++latest.current;
    if (term.length < MIN_CHARS) return;
    const ctrl = new AbortController();
    const timer = setTimeout(async () => {
      try {
        const res = await fetch(`/suggest?q=${encodeURIComponent(term)}`, { signal: ctrl.signal });
        if (!res.ok) return;
        const body = (await res.json()) as { items: Suggestion[] };
        if (request !== latest.current) return; // a newer request is under way
        setItems(body.items);
        setActive(-1);
        setOpen(true);
      } catch {
        // aborted or offline: the plain form still works
      }
    }, SUGGEST_DELAY_MS);
    return () => {
      clearTimeout(timer);
      ctrl.abort();
    };
  }, [q]);

  const expanded = open && items.length > 0;

  const go = (s: Suggestion) => {
    setOpen(false);
    router.push(`/aktie/${encodeURIComponent(s.ticker)}`);
  };

  const onKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    const n = items.length;
    switch (e.key) {
      case "ArrowDown":
        if (n === 0) return;
        e.preventDefault();
        setOpen(true);
        setActive((a) => (a + 1) % n);
        break;
      case "ArrowUp":
        if (n === 0) return;
        e.preventDefault();
        setOpen(true);
        setActive((a) => (a <= 0 ? n - 1 : a - 1));
        break;
      case "Enter":
        if (expanded && active >= 0) {
          e.preventDefault();
          go(items[active]);
        }
        break;
      case "Escape":
        if (expanded) {
          e.preventDefault();
          setOpen(false);
        }
        break;
    }
  };

  return (
    <form className="search" action="/suche" method="get" role="search">
      <div className="suggest">
        <input
          name="q"
          type="search"
          placeholder="Aktie, Ticker oder ISIN"
          aria-label="Aktie suchen"
          required
          maxLength={64}
          autoComplete="off"
          role="combobox"
          aria-autocomplete="list"
          aria-expanded={expanded}
          aria-controls={listId}
          aria-activedescendant={expanded && active >= 0 ? `${listId}-${active}` : undefined}
          value={q}
          onChange={(e) => {
            setQ(e.target.value);
            if (e.target.value.trim().length < MIN_CHARS) {
              setItems([]);
              setOpen(false);
            }
          }}
          onKeyDown={onKeyDown}
          onFocus={() => setOpen(true)}
          onBlur={() => setOpen(false)}
        />
        <ul id={listId} role="listbox" aria-label="Vorschläge" className="suggestions" hidden={!expanded}>
          {items.map((s, i) => (
            <li
              key={`${s.mic}:${s.ticker}`}
              id={`${listId}-${i}`}
              role="option"
              aria-selected={i === active}
              // Keep the focus in the field, so blur does not close the list before the click.
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => go(s)}
            >
              <span>{s.name}</span>
              <span className="suggest-ticker">{s.ticker}</span>
            </li>
          ))}
        </ul>
      </div>
      <button type="submit">Suchen</button>
    </form>
  );
}
