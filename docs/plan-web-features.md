# Plan Website: Review-Fixes und Leser-Features

Entwurf vom 2026-11-04. Grundlage: das Review der Website vom 2026-11-04 (Quellen-Filter, Filtern-Button,
Tab-Leiste, „Gestern“) und die daraus gewählten Features: Aktienliste mit Chips, „Heute im Gespräch“,
Suchvorschläge, Hinweis „N neue Meldungen“, Schalter „Nur Ad-hoc“.

Rahmen, der für alle Aufgaben gilt:

- Öffentlich bleibt es bei Titel, Quelle, Link, Zeit und Tickern. Neu dazu kommen nur **Zählungen**
  (Anzahl Meldungen je Ticker) und Firmennamen. Kein Sentiment, kein Event-Typ (AGENTS.md, Rechtsregeln 1 und 4).
- Der Browser spricht nie direkt mit der Go-API, nur mit Next-Routen (wie `/feed`).
- Alles funktioniert auch ohne JavaScript, JavaScript macht es nur bequemer.
- Vorbedingung: `feature/brand-colors` (inkl. Scrollbalken-Fix `d3befcc`) ist nach `develop` gemergt.
- Jede Aufgabe läuft auf einem eigenen Branch von `develop` per TDD und wird mit `--no-ff` zurückgemergt.
  Gates: Backend `docker compose --profile test run --rm backend-test sh -c "gofmt -l .; go vet ./... && go test ./..."`,
  Frontend `docker compose --profile test run --rm web-test`.

## A. Review-Fixes (nur Frontend)

- [x] **1. Quellen-Liste passt zum Bereich** — `fix/web-sources-per-section`
  Files: `frontend/lib/filters.ts`, `frontend/lib/filters.test.ts`, `frontend/app/page.tsx`, `frontend/app/aktie/[ticker]/page.tsx`
  What: Jeder Bereich in `SECTIONS` bekommt `markets` (ATX → `AT`, DAX → `DE`, S&P 500 → `US`, jeweils plus `MULTI`;
  Alle/Ad-hoc/SEC-Filings ohne Einschränkung). `sourcesFor(section, sources)` liefert nur Quellen, deren `market`
  und `kind` zum Bereich passen. Die Aktienseite filtert nach dem Markt der Börse (`XWBO` → AT, `XETR` → DE,
  `XNYS`/`XNAS` → US, plus `MULTI`), dieselbe Tabelle wie `MARKETS`, nach `lib/` verschoben.
  Eine per URL gewählte Quelle, die nicht in der Liste steht, bleibt als Option erhalten, damit der Select nicht lügt.
  Done: Im ATX-Tab stehen weder SEC- noch PR-Newswire-Quellen; im SEC-Filings-Tab nur die beiden SEC-Quellen;
  auf `/aktie/VOE` keine US-Quellen. Tabellentest für `sourcesFor`.

- [x] **2. „Filtern“-Button nur ohne JavaScript** — `fix/web-filter-button`
  Files: `frontend/components/Filters.tsx`, `frontend/components/Filters.test.tsx`
  What: Der Button wird gerendert (Fallback ohne JS) und nach dem Mounten per `hidden` ausgeblendet, weil eine
  Änderung im Select ohnehin sofort greift.
  Done: Test: nach dem Render ist der Button nicht sichtbar; im Server-HTML ist er vorhanden.

- [x] **3. Hinweis auf weitere Tabs** — `fix/web-tabs-hint`
  Files: `frontend/components/SectionTabs.tsx`, `frontend/components/SectionTabs.test.tsx`, `frontend/app/globals.css`, `frontend/app/theme.css`
  What: Hat die Tab-Leiste rechts noch Inhalt, blendet ein Verlauf (`mask-image`) den Rand aus; links ebenso,
  sobald gescrollt wurde. Gesteuert über `data-more-start`/`data-more-end`, gesetzt von einem kleinen Scroll-Listener.
  Der aktive Tab wird beim Laden in den sichtbaren Bereich gescrollt (`scrollIntoView({inline: "nearest"})`).
  Die Breite des Verlaufs kommt als Token aus `theme.css`.
  Done: Test: Attribute folgen `scrollLeft`/`scrollWidth`. Live auf 375 px Breite geprüft: Verlauf rechts,
  nach dem Wischen auch links, auf Desktop keiner.

- [x] **4. „Gestern“ am Tag der Zeitumstellung** — `fix/web-yesterday-dst`
  Files: `frontend/lib/format.ts`, `frontend/lib/format.test.ts`
  What: `dayLabel` rechnet „Gestern“ über den Kalendertag (Tag von `now` in Wien minus ein Kalendertag,
  Mittag UTC) statt über `now − 24 h`.
  Done: Regressionstest: `now` = 2026-10-25 23:30 Wien (25-Stunden-Tag), Meldung vom 24.10. → „Gestern“.
  Ebenso für den 23-Stunden-Tag im März.

## B. Aktienliste mit Chips

- [x] **5. API: Ticker mit Meldungszahl** — `feature/api-ticker-counts`
  Files: `backend/internal/store/queries/news.sql`, `backend/internal/store/db/*` (sqlc), `backend/internal/service/tickers.go`,
  `backend/internal/service/tickers_test.go`, `backend/internal/httpapi/tickers.go`, `backend/internal/httpapi/tickers_test.go`,
  `backend/internal/httpapi/dto.go`, `backend/internal/httpapi/dto_test.go`, `backend/internal/httpapi/router.go`
  What: Zwei Queries mit denselben Regeln wie `ListNews` (zählt **Cluster**, nur Links mit Status `linked`,
  Index-Mitgliedschaft zum heutigen Tag):
  - `ListIndexMembers(index_key, from, kinds?)`: alle aktiven Werte des Index mit Name und Anzahl Meldungen seit `from`,
    auch mit 0.
  - `TopTickers(index?, kinds?, from, limit 1–20)`: die meistgenannten Ticker, sortiert nach Anzahl, dann Ticker.
  Endpunkte: `GET /api/v1/indices/{index}/instruments?from=&kind=` und `GET /api/v1/tickers/top?index=&kind=&from=&limit=`,
  Antwort `{items: [{ticker, name, count}]}`, `Cache-Control` 60 s wie `/news`. Unbekannter Index → 404,
  ungültige Parameter → 400. Die neuen DTOs kommen in den Rechtstest der DTOs.
  Done: Mit festen Daten: Ein Cluster mit drei Artikeln zählt einmal; `pending`/`rejected` zählt nicht;
  ein ausgeschiedenes Index-Mitglied fehlt; Werte ohne Meldung erscheinen mit 0. Store-Tests laufen up → down → up
  (keine Migration nötig; falls ein Index für die Zählung fehlt, eigene goose-Datei).

- [x] **6. Seite `/aktien`** — `feature/web-stock-list`
  Files: `frontend/app/aktien/page.tsx`, `frontend/lib/api.ts`, `frontend/lib/api.test.ts`, `frontend/components/TickerChips.tsx`,
  `frontend/components/TickerChips.test.tsx`, `frontend/components/SiteHeader.tsx`, `frontend/app/globals.css`
  What: Tabs ATX · DAX · S&P 500 (`?index=`), Zeitraum 24 Stunden / 7 Tage (`?range=`), Sortierung „Meiste Meldungen“
  oder „A–Z“ (`?sort=`). Jeder Wert ist ein Chip „Name · Ticker · Anzahl“ und verlinkt auf `/aktie/<ticker>`; Werte mit 0
  sind blasser. Alles per Link/GET-Form, also ohne JS bedienbar. Der Header bekommt einen Link „Aktien“.
  Metadaten für SEO (Titel „ATX-Aktien – News je Wert“).
  Done: Komponententest für Chips (Link, Anzahl, 0-Darstellung). Live: `/aktien?index=ATX` zeigt alle ATX-Werte,
  S&P 500 bleibt bei ~500 Chips flüssig.

- [x] **7. „Heute im Gespräch“ über dem Feed** — `feature/web-trending`
  Files: `frontend/components/Trending.tsx`, `frontend/components/Trending.test.tsx`, `frontend/app/page.tsx`, `frontend/app/globals.css`
  What: Eine Zeile Chips mit den 10 meistgenannten Tickern der letzten 24 h, passend zum gewählten Bereich
  (Index und Quellenarten wie der Feed). Eigenes `Suspense`, damit der Feed nicht wartet. Ohne Treffer wird
  nichts gerendert. Auf dem Handy wischbar wie die Tabs.
  Done: Test: rendert nichts bei leerer Liste, sonst Chips in API-Reihenfolge mit Anzahl. Live im ATX- und DAX-Tab geprüft.

## C. Für Leser

- [x] **8. Suchvorschläge im Header** — `feature/web-search-suggest`
  Files: `frontend/app/suggest/route.ts`, `frontend/components/SearchBox.tsx`, `frontend/components/SearchBox.test.tsx`,
  `frontend/components/SiteHeader.tsx`, `frontend/app/globals.css`
  What: Next-Route `/suggest?q=` reicht `searchInstruments` durch (max. 10, `Cache-Control` 60 s). Das Suchfeld wird
  eine Combobox nach ARIA-Muster (`role="combobox"`, `listbox`, `aria-activedescendant`): ab 2 Zeichen,
  150 ms Debounce, veraltete Antworten werden verworfen (`AbortController`). Pfeiltasten wählen, Enter öffnet die Aktie,
  Escape schließt, ein Klick daneben auch. Ohne JS bleibt es das heutige GET-Formular auf `/suche`.
  Done: Tests: Debounce, Tastatursteuerung, Enter ohne Auswahl schickt das Formular ab, alte Antwort überschreibt neue nicht.
  Live mit Tastatur und auf dem Handy geprüft.

- [x] **9. Hinweis „N neue Meldungen“** — `feature/web-new-items`
  Files: `frontend/components/InfiniteNews.tsx`, `frontend/components/InfiniteNews.test.tsx`, `frontend/app/globals.css`
  What: Solange der Tab sichtbar ist (`visibilitychange`), holt der Feed alle 60 s die erste Seite über `/feed`
  (gleiche Query, ohne Cursor; das deckt sich mit dem 60-s-Cache). Neu ist, was noch nicht in der Liste steht.
  Gibt es Neues, erscheint oben ein Button „3 neue Meldungen“ (ab 25 „25+ neue Meldungen“). Ein Klick fügt sie
  sortiert nach (`published_at`, `id`) ein, ohne die schon geladenen älteren Seiten zu verlieren, und scrollt nach oben.
  Fehler beim Abfragen bleiben still. `aria-live="polite"` meldet die Anzahl.
  Done: Tests mit Fake-Timern: kein Abruf bei verstecktem Tab; doppelte IDs zählen nicht; Klick fügt ein und
  setzt den Zähler zurück; nachträglich eingetroffene ältere Meldung landet an der richtigen Stelle.

- [x] **10. Schalter „Nur Ad-hoc“ auf der Aktienseite** — `feature/web-adhoc-only`
  Files: `frontend/lib/filters.ts`, `frontend/lib/filters.test.ts`, `frontend/lib/feed.ts`, `frontend/lib/feed.test.ts`,
  `frontend/components/Filters.tsx`, `frontend/components/Filters.test.tsx`, `frontend/app/aktie/[ticker]/page.tsx`
  What: `Filters` bekommt `adhoc: boolean` (`?adhoc=1`). `stockNewsParams` setzt dann `kind=adhoc`; eine zusätzlich
  gewählte Quelle gilt weiter (die API verknüpft beides mit UND). Die Checkbox erscheint nur auf der Aktienseite
  (Prop `showAdhoc`) und wirkt sofort wie die Selects. `feedQuery`/`feedParams` tragen den Wert, damit das
  Nachladen dieselbe Auswahl liefert.
  Done: Tests: `?adhoc=1` → `kind=adhoc`, fehlt im URL wenn aus, Nachladen über `/feed` behält ihn.
  Live: `/aktie/VOE?adhoc=1` zeigt nur Ad-hoc-Meldungen.

Umsetzungsnotizen (2026-11-04, alle zehn Aufgaben gemergt):
- Die Live-Prüfung zu 8 zeigte, dass „ers“ Erste Group nicht unter den 10 Treffern fand (Tausende SEC-Firmen,
  alphabetisch). `SearchInstruments` sortiert jetzt: exakter Ticker, Namensanfang, aktuelle Index-Werte,
  Ticker-Anfang, Name (mit Regressionstest).
- 9: Der Button schwebt unten mittig statt oben, damit er nicht mit dem einblendenden Header kollidiert.
- 3 und 7 teilen sich `useScrollRow` (Verlauf an den Rändern, aktueller Eintrag im Blick); Tabs über `TabNav`.

## Reihenfolge

1–4 zuerst (klein, unabhängig). Dann 5 → 6 → 7 (6 und 7 brauchen die API aus 5). 8, 9 und 10 sind voneinander
und von B unabhängig und können jederzeit dazwischen.

## Bewusst nicht in diesem Plan

- Watchlist, RSS-Feed, IR-Links, Sitemap/OpenGraph: später, eigener Plan.
- Push oder Live-Updates per SSE/WebSocket: Polling alle 60 s reicht und passt zum Cache.
- Sentiment oder Event-Typ in Chips oder Hinweisen: bleibt intern.
