# Plan Website: Ideen aus Alltop und Upstract

Entwurf vom 2026-11-05. Grundlage ist die Analyse von alltop.com und upstract.com am 2026-11-05
(HTML und CSS ausgewertet, nicht im Browser angesehen).

## Ausgangslage

Beide Seiten sammeln Überschriften aus vielen Quellen und verlinken auf das Original, also dasselbe Prinzip wie wir.

- **Alltop**: Themen statt Quellen. Je Geschichte eine Hauptmeldung mit „· 23 sources“ und „More: Quelle · Quelle“,
  KI-Zusammenfassung und „Why it matters“. Ruhige, redaktionelle Typografie: Papier-Beige `#F4F3EE`, Tinte `#111`,
  ein Akzent (Rost `#D13D1F`), Inter plus JetBrains Mono mit `tabular-nums` für Zeiten und Zahlen,
  Haarlinien (10 % Deckkraft), kleine Labels in Großbuchstaben (10.5px, Sperrung .08em). Relative Zeiten,
  „refreshes every 5 min“, Ansichten `stories`/`columns`/`list`, Tastenkürzel `j`/`k`/`v`/`/`, Dark Mode.
- **Upstract**: Raster mit einer Kachel je Quelle (rund 60 Quellen, je 8–10 Titel, „More“ lädt in der Kachel nach).
  Ansichten „Grid“, „Briefing“ und „Wire“ (chronologisch). Hover-Vorschau mit Feed-Beschreibung und Vorschaubild,
  Teilen-Button je Meldung, Dark Mode über eine eigene CSS-Datei mit `prefers-color-scheme`. Ohne Login führen
  547 von 652 Links auf `/login` statt zur Quelle.

Unser Feed zeigt schon eine Meldung je Cluster (`ListNews`). In den letzten 7 Tagen hatten aber nur 27 von
10.096 Clustern mehr als eine Quelle. Die Anzeige „N Quellen“ bringt also erst mit mehr überlappenden
Nachrichtenquellen (EU-Plan, weitere Newsfeeds) spürbar etwas.

## Rahmen für alle Aufgaben

- Öffentlich bleibt es bei Titel, Quelle, Link, Zeit und Tickern, dazu Zählungen. Kein Text, kein Sentiment,
  kein Event-Typ (AGENTS.md, Rechtsregeln 1 und 4).
- Keine Anfragen an Dritte aus dem Browser der Besucher (keine Google-Favicons, keine fremden Fonts zur Laufzeit;
  Fonts über `next/font`, das sie selbst ausliefert).
- Der Browser spricht nur mit Next-Routen, nie direkt mit der Go-API. Alles funktioniert ohne JavaScript.
- Aussehen nur über `app/theme.css`.
- Jede Aufgabe auf eigenem Branch von `develop` per TDD, Merge mit `--no-ff`. Gates:
  `docker compose --profile test run --rm backend-test sh -c "gofmt -l .; go vet ./... && go test ./..."` und
  `docker compose --profile test run --rm web-test`.

## Entscheidungen (2026-11-05)

1. **Look:** Das Dunkelblau der Marke und der weiße Inhalt bleiben. Übernommen wird nur die ruhigere
   Typografie von Alltop (Haarlinien, Mono-Zahlen, kleine Labels in Großbuchstaben), keine neue Farbwelt.
2. **Dark Mode:** ja, er folgt der Systemeinstellung. Damit entfällt „always light“ in `theme.css`.

## A. Darstellung

- [x] **1. Ruhigere Typografie** — `feature/web-typography`
  Files: `frontend/app/theme.css`, `frontend/app/globals.css`, `frontend/app/theme.test.ts`
  What: Neue Tokens `--font-mono` (für Zeiten, Zählungen, Badges, mit `font-variant-numeric: tabular-nums`),
  `--hairline` (Trennlinien mit ca. 10 % Deckkraft statt `--border`), `--micro-*` für Tagesköpfe
  („HEUTE“, „GESTERN“) in Großbuchstaben mit Sperrung. Mono-Font als System-Stack (`ui-monospace`, SF Mono, Menlo, Consolas …).
  Keine neuen Farbwerte (Entscheidung 1).
  Umsetzungsnotiz: statt `next/font` ein System-Stack. Es wird nichts geladen, und der Build braucht keinen
  Download von Google Fonts (der hinter der Firmen-Firewall scheitern kann). Mono bekommen Zeiten, Ticker-Chips,
  Zählungen und Badges; Haarlinien trennen Meldungen, Tage, Suchtreffer und den Footer. Eingabefelder, Tabs und
  Segmente behalten `--border`. Live geprüft auf 1280 px und 375 px (Playwright, keine horizontale Scrollleiste).
  Done: Zeitspalte fluchtet; `theme.test.ts` prüft die neuen Tokens; Screenshot Desktop und 375 px.

- [x] **2. Relative Zeit für frische Meldungen** — `feature/web-relative-time`
  Files: `frontend/lib/format.ts`, `frontend/lib/format.test.ts`, `frontend/lib/useNow.ts`, `frontend/components/NewsTime.tsx` (+ Test), `frontend/components/NewsList.tsx`
  What: Unter 60 Minuten „vor 12 Min“, darüber die Uhrzeit wie bisher. `<time datetime>` mit der absoluten
  Zeit im `title`. Der Server rendert die Uhrzeit; der Client ersetzt sie nach dem Mounten und
  aktualisiert minütlich (der Feed fragt ohnehin jede Minute nach).
  Done: Tabellentest für die Grenzen (0, 59, 60 Min, Tageswechsel in Wien).
  Umsetzungsnotiz: Angezeigt wird kompakt „jetzt“ bzw. „12 Min“, weil „vor 12 Min“ in der Zeitspalte umbrach;
  Screenreader lesen „vor 12 Min“ (`sr-only`), der Tooltip zeigt Datum und Uhrzeit. Ein gemeinsamer Takt
  (`useNow`, `useSyncExternalStore`, alle 30 s) für alle Zeilen; auf dem Server `null`, daher bleibt das
  gecachte HTML gleich und es gibt keinen Hydration-Unterschied. Bis 5 Min in der Zukunft gilt als „jetzt“.

- [x] **3. Logos der Quellen** — `feature/web-source-icons`
  Files: `frontend/public/sources/<key>.png|svg`, `frontend/components/SourceIcon.tsx` (+ Test), `frontend/components/NewsList.tsx`
  What: Kleines Icon (16 px) vor dem Quellennamen, aus `public/` ausgeliefert, einmalig von Hand abgelegt.
  Fehlt ein Icon, ein Kreis mit dem Anfangsbuchstaben. Kein Laden von Google oder der Quelle zur Laufzeit.
  Done: Test: bekannter Key → `<img>`, unbekannter → Buchstabe.
  Umsetzungsnotiz: Zuordnung über den Anbieter, das Präfix des Keys vor dem ersten „_“ (`fn_dax` → `fn`),
  in `frontend/lib/sources.ts`; 7 Icons (fn, oekb, ots, prn, sec, telekom, voestalpine) als 64-px-PNG, angezeigt
  in 14 px. Der Ersatz-Buchstabe kommt per CSS (`::before`), damit die Zeile keinen Text dazubekommt.
  Dabei gefunden: Das Docker-Image (`output: "standalone"`) kopierte `public/` nicht; das `Dockerfile` kopiert
  es jetzt, ein Regressionstest prüft das. Live geprüft: alle Icons laden, keine Anfragen an fremde Hosts.

- [x] **4. Aktualitätszeile** — `feature/api-status`
  Files: `backend/internal/store/queries/*.sql`, `backend/internal/service/status.go` (+ Test),
  `backend/internal/httpapi/status.go` (+ Test), `backend/internal/httpapi/dto.go` (+ `publicDTOs`),
  `frontend/components/FreshnessLine.tsx` (+ Test)
  What: `GET /api/v1/status` mit Anzahl aktiver Quellen und Zeit des letzten erfolgreichen Abrufs
  (aus `ingest_runs`), nur Zählungen und Zeitpunkte. Darunter unter „Aktuelle Meldungen“:
  „16 Quellen · zuletzt aktualisiert vor 2 Min“.
  Done: DTO-Test grün; Zeile erscheint auf Startseite und Aktienseite.
  Umsetzungsnotiz: kein eigener Endpunkt `/api/v1/status`, sondern `last_ok_at` je Quelle in
  `GET /api/v1/sources` (Ende des letzten Laufs mit `ok`/`partial`, sonst `null`; Lateral-Join über
  `ingest_runs_source_idx`). Die Website lädt die Quellen ohnehin je Bereich und zeigt so die Aktualität
  passend zum Tab bzw. zur gewählten Quelle (`shownSources`): Alle 13, ATX 8, DAX 6, S&P 500 5, Ad-hoc 2,
  SEC-Filings 3 Quellen. Server rendert „um 21:55“, der Browser „vor 2 Min“/„gerade eben“; nicht heute → Datum.

- [x] **5. Dark Mode** (Entscheidung 2) — `feature/web-dark-mode`
  Files: `frontend/app/theme.css`, `frontend/app/theme.test.ts`
  What: Zweiter Token-Satz unter `@media (prefers-color-scheme: dark)`; `color-scheme: light dark`.
  Done: Kontrast ≥ 4.5:1 für Text und Links in beiden Modi (Test über die Token-Werte).
  Umsetzungsnotiz: Der Header behält das Markenblau; Links, Tabs und Chips bekommen hellere Blautöne
  (`--link` `#8fb8ff`, `--chip-bg` `#1f4c8f`), Hintergrund `#0e131a`. Ein Test verlangt, dass jede wörtliche
  Farbe des hellen Satzes im dunklen neu gesetzt ist, damit keine vergessen wird; dazu neue Kontrastpaare
  (Hover-Link, Suchvorschlag, Button-Hover). Live geprüft mit `colorScheme: "dark"`: Startseite, Aktienseite,
  `/aktien`, Suchvorschläge, 375 px.

## B. Ansichten und Bedienung

- [x] **6. Ansicht „Nach Quelle“** — `feature/web-by-source`
  Files: `frontend/app/quellen/page.tsx`, `frontend/components/SourceGrid.tsx` (+ Test), `frontend/components/SiteHeader.tsx`
  What: Wie das Raster von Upstract: eine Kachel je Quelle des gewählten Bereichs mit den 5 neuesten Titeln,
  „Mehr“ führt auf den Feed mit `?source=<key>`. Bereichs-Tabs wie auf der Startseite. Der bestehende Feed
  bleibt die Standardansicht (bei Upstract „Wire“). Erst mit der vorhandenen API (eine Anfrage je Quelle,
  serverseitig parallel); ein gebündelter Endpunkt nur, wenn das messbar zu langsam ist.
  Done: Test: je Quelle eine Kachel, leere Quellen ausgeblendet; Umschalter Feed/Quellen im Kopf.
  Umsetzungsnotiz: `/quellen` mit Bereichs-Tabs, Umschalter „Feed | Nach Quelle“ (`ViewSwitch`, unter den Tabs,
  Bereich und Zeitraum bleiben), Aktualitätszeile und Kacheln (`SourceGrid`, `tileParams`: 5 Meldungen je Quelle,
  Index und Zeitraum des Bereichs). Eine fehlerhafte Quelle zeigt nur keine Kachel. Weil Tagesköpfe fehlen,
  zeigen Kacheln für ältere Tage das Datum („04.11.“) statt der Uhrzeit. `j`/`k` laufen auch durch die Kacheln.
  Live: 100–560 ms je Seite (eine API-Anfrage je Quelle, parallel), daher vorerst kein gebündelter Endpunkt.

- [x] **7. Tastenkürzel** — `feature/web-shortcuts`
  Files: `frontend/components/Shortcuts.tsx` (+ Test), `frontend/app/layout.tsx`
  What: `j`/`k` nächste/vorige Meldung (Fokus und Markierung), `o` oder `Enter` öffnet die Quelle in einem neuen
  Fenster, `/` springt in die Suche, `?` zeigt die Hilfe. Nicht aktiv, solange ein Eingabefeld den Fokus hat.
  Done: Test mit simulierten Tasten; Hilfe per `?` erreichbar.
  Umsetzungsnotiz: `o` öffnet die Meldung mit dem Fokus (Enter tut das ohnehin über den Link). Die Hilfe ist
  ein kleines Feld unten rechts (`role="dialog"`, nicht modal), dazu ein Button „Tastenkürzel“ im Footer, der erst
  nach der Hydration erscheint. Die Zeile mit dem Fokus ist hinterlegt (`li:has(.title:focus-visible)`).
  Live mit echten Tasten geprüft (Playwright).

- [x] **8. Teilen** — `feature/web-share`
  Files: `frontend/components/ShareButton.tsx` (+ Test), `frontend/components/NewsList.tsx`
  What: Kleiner Button je Meldung: auf dem Handy `navigator.share` mit Titel und Original-Link,
  sonst Link kopieren mit kurzer Bestätigung. Ohne JavaScript nicht sichtbar.
  Done: Test für beide Wege.
  Umsetzungsnotiz: Geteilt wird der Original-Link. Teilen und Zwischenablage gibt es nur im sicheren Kontext
  (https oder localhost); über reines http, z. B. `http://aktien.<LAN_IP>.sslip.io/` auf dem Handy, bleibt der
  Button daher verborgen. Abbruch des Teilen-Dialogs bleibt still. Live geprüft unter localhost (Zwischenablage).

## C. Cluster sichtbar machen (später)

- [x] **9. „Auch bei …“** — `feature/news-cluster-sources`
  Files: `backend/internal/store/queries/news.sql`, sqlc, `backend/internal/service/news.go`,
  `backend/internal/httpapi/dto.go`, `frontend/components/NewsList.tsx`
  What: `ListNews` liefert je Cluster zusätzlich die anderen Mitglieder (Quelle, Titel, Link), höchstens 5.
  Die Website zeigt bei mehr als einer Quelle „· 3 Quellen“ und darunter „Auch bei: APA-OTS · FinanzNachrichten“.
  Titel und Links sind öffentlich erlaubt (Rechtsregel 1). Optional eine Liste „Meistberichtet heute“ nach
  Zahl der Quellen, ohne Wertung.
  Wann: sobald mehr Cluster mehrere Quellen haben (nach dem EU-Plan oder weiteren Newsfeeds).
  Done: DTO-Test grün; Store-Test mit einem Cluster aus 3 Quellen.
  Umsetzungsnotiz: vorgezogen (Freigabe 2026-11-05). Eine zweite Query `ListAlsoAt` für alle Einträge einer Seite
  (je andere Quelle das früheste Mitglied, nie die Quelle des Repräsentanten), der Service sortiert nach Zeit und
  kappt bei 5. Die Website zeigt nur „Auch bei: Quelle · Quelle“, verlinkt auf die Kopien (neues Fenster, Titel als
  Tooltip); auf „· N Quellen“ und „Meistberichtet heute“ verzichtet, solange Überschneidungen selten sind.
  Live: `/aktie/LNZ` zeigt bei 4 Meldungen „Auch bei: OeKB OAM Pflichtmeldungen“ (APA-OTS + OeKB + FN ATX).

## Bewusst nicht übernommen

- **KI-Zusammenfassungen und „Why it matters“ (Alltop):** Text über den Titel hinaus, Rechtsregeln 1 und 2;
  eine Einordnung zu einer Aktie grenzt an Anlageberatung (MAR).
- **Hover-Vorschau mit Feed-Beschreibung und Vorschaubild (Upstract):** Beschreibung verstößt gegen Rechtsregel 1,
  Bilder sind urheberrechtlich geschützt.
- **Gruppierung nach Ereignis (Earnings, Launches):** Der Ereignistyp ist intern (Rechtsregel 4).
  Die Trennung nach Art der Quelle (Ad-hoc, SEC-Filings) haben wir bereits.
- **„Story of the Day“ mit „Hot“-Badge:** wirkt bei Aktien wie eine Empfehlung.
- **Favicons über `google.com/s2/favicons`:** schickt die IP der Besucher an Google (DSGVO).
- **Links hinter Login (Upstract):** Wir verlinken immer direkt auf die Quelle.
- **Trends von X und Google:** keine passende, lizenzklare Quelle; „Heute im Gespräch“ deckt das ab.

## Reihenfolge

1 → 2 → 3 (sichtbarster Gewinn, nur Frontend), dann 5, 4, 7, 6, 8. Aufgabe 9 nach dem EU-Plan.
