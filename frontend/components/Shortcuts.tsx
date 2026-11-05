"use client";

import { useEffect, useId, useRef, useState, useSyncExternalStore } from "react";

const KEYS: [string, string][] = [
  ["j", "nächste Meldung"],
  ["k", "vorige Meldung"],
  ["o", "Meldung bei der Quelle öffnen (neues Fenster)"],
  ["/", "Suche"],
  ["?", "diese Hilfe"],
];

const noop = () => () => {};
const titles = () => [...document.querySelectorAll<HTMLAnchorElement>("ol.news a.title")];

function typing(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  return target.isContentEditable || ["INPUT", "TEXTAREA", "SELECT"].includes(target.tagName);
}

// Keyboard shortcuts for readers who go through many headlines: j/k move the
// focus through the titles, o opens the focused one at its source, / jumps
// into the search, ? explains it all. Inactive while a field has the focus.
export default function Shortcuts() {
  const [help, setHelp] = useState(false);
  // true once hydrated, false on the server (same pattern as Filters)
  const mounted = useSyncExternalStore(noop, () => true, () => false);
  const headingId = useId();
  const close = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (help) close.current?.focus();
  }, [help]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setHelp(false);
        return;
      }
      if (e.metaKey || e.ctrlKey || e.altKey || typing(e.target)) return;
      const list = titles();
      const at = list.indexOf(document.activeElement as HTMLAnchorElement);
      switch (e.key) {
        case "j":
        case "k": {
          if (list.length === 0) return;
          const next = at < 0 ? 0 : Math.min(Math.max(at + (e.key === "j" ? 1 : -1), 0), list.length - 1);
          list[next].focus();
          list[next].scrollIntoView?.({ block: "nearest" });
          break;
        }
        case "o":
          if (at >= 0) window.open(list[at].href, "_blank", "noopener,noreferrer");
          break;
        case "/":
          document.querySelector<HTMLInputElement>("form.search input")?.focus();
          break;
        case "?":
          setHelp((h) => !h);
          break;
        default:
          return;
      }
      e.preventDefault();
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, []);

  return (
    <>
      {/* Without JavaScript there are no shortcuts, so no button either. */}
      <button type="button" className="link-button" hidden={!mounted} onClick={() => setHelp(true)}>
        Tastenkürzel
      </button>
      {help && (
        <div className="shortcuts" role="dialog" aria-labelledby={headingId}>
          <h2 id={headingId}>Tastenkürzel</h2>
          <dl>
            {KEYS.map(([key, what]) => (
              <div key={key}>
                <dt>
                  <kbd>{key}</kbd>
                </dt>
                <dd>{what}</dd>
              </div>
            ))}
          </dl>
          <button type="button" ref={close} onClick={() => setHelp(false)}>
            Schließen
          </button>
        </div>
      )}
    </>
  );
}
