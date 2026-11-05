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

- [ ] **1. Ruhigere Typografie** — `feature/web-typography`
  Files: `frontend/app/theme.css`, `frontend/app/globals.css`, `frontend/app/layout.tsx`, `frontend/app/theme.test.ts`
  What: Neue Tokens `--font-mono` (für Zeiten, Zählungen, Badges, mit `font-variant-numeric: tabular-nums`),
  `--hairline` (Trennlinien mit ca. 10 % Deckkraft statt `--border`), `--micro-*` für Tagesköpfe
  („HEUTE“, „GESTERN“) in Großbuchstaben mit Sperrung. Mono-Font über `next/font` (selbst ausgeliefert).
  Keine neuen Farbwerte (Entscheidung 1).
  Done: Zeitspalte fluchtet; `theme.test.ts` prüft die neuen Tokens; Screenshot Desktop und 375 px.

- [ ] **2. Relative Zeit für frische Meldungen** — `feature/web-relative-time`
  Files: `frontend/lib/format.ts`, `frontend/lib/format.test.ts`, `frontend/components/NewsList.tsx`
  What: Unter 60 Minuten „vor 12 Min“, darüber die Uhrzeit wie bisher. `<time datetime>` mit der absoluten
  Zeit im `title`. Der Server rendert die Uhrzeit; der Client ersetzt sie nach dem Mounten und
  aktualisiert minütlich (der Feed fragt ohnehin jede Minute nach).
  Done: Tabellentest für die Grenzen (0, 59, 60 Min, Tageswechsel in Wien).

- [ ] **3. Logos der Quellen** — `feature/web-source-icons`
  Files: `frontend/public/sources/<key>.png|svg`, `frontend/components/SourceIcon.tsx` (+ Test), `frontend/components/NewsList.tsx`
  What: Kleines Icon (16 px) vor dem Quellennamen, aus `public/` ausgeliefert, einmalig von Hand abgelegt.
  Fehlt ein Icon, ein Kreis mit dem Anfangsbuchstaben. Kein Laden von Google oder der Quelle zur Laufzeit.
  Done: Test: bekannter Key → `<img>`, unbekannter → Buchstabe.

- [ ] **4. Aktualitätszeile** — `feature/api-status`
  Files: `backend/internal/store/queries/*.sql`, `backend/internal/service/status.go` (+ Test),
  `backend/internal/httpapi/status.go` (+ Test), `backend/internal/httpapi/dto.go` (+ `publicDTOs`),
  `frontend/components/FreshnessLine.tsx` (+ Test)
  What: `GET /api/v1/status` mit Anzahl aktiver Quellen und Zeit des letzten erfolgreichen Abrufs
  (aus `ingest_runs`), nur Zählungen und Zeitpunkte. Darunter unter „Aktuelle Meldungen“:
  „16 Quellen · zuletzt aktualisiert vor 2 Min“.
  Done: DTO-Test grün; Zeile erscheint auf Startseite und Aktienseite.

- [ ] **5. Dark Mode** (Entscheidung 2) — `feature/web-dark-mode`
  Files: `frontend/app/theme.css`, `frontend/app/theme.test.ts`
  What: Zweiter Token-Satz unter `@media (prefers-color-scheme: dark)`; `color-scheme: light dark`.
  Done: Kontrast ≥ 4.5:1 für Text und Links in beiden Modi (Test über die Token-Werte).

## B. Ansichten und Bedienung

- [ ] **6. Ansicht „Nach Quelle“** — `feature/web-by-source`
  Files: `frontend/app/quellen/page.tsx`, `frontend/components/SourceGrid.tsx` (+ Test), `frontend/components/SiteHeader.tsx`
  What: Wie das Raster von Upstract: eine Kachel je Quelle des gewählten Bereichs mit den 5 neuesten Titeln,
  „Mehr“ führt auf den Feed mit `?source=<key>`. Bereichs-Tabs wie auf der Startseite. Der bestehende Feed
  bleibt die Standardansicht (bei Upstract „Wire“). Erst mit der vorhandenen API (eine Anfrage je Quelle,
  serverseitig parallel); ein gebündelter Endpunkt nur, wenn das messbar zu langsam ist.
  Done: Test: je Quelle eine Kachel, leere Quellen ausgeblendet; Umschalter Feed/Quellen im Kopf.

- [ ] **7. Tastenkürzel** — `feature/web-shortcuts`
  Files: `frontend/components/Shortcuts.tsx` (+ Test), `frontend/app/layout.tsx`
  What: `j`/`k` nächste/vorige Meldung (Fokus und Markierung), `o` oder `Enter` öffnet die Quelle in einem neuen
  Fenster, `/` springt in die Suche, `?` zeigt die Hilfe. Nicht aktiv, solange ein Eingabefeld den Fokus hat.
  Done: Test mit simulierten Tasten; Hilfe per `?` erreichbar.

- [ ] **8. Teilen** — `feature/web-share`
  Files: `frontend/components/ShareButton.tsx` (+ Test), `frontend/components/NewsList.tsx`
  What: Kleiner Button je Meldung: auf dem Handy `navigator.share` mit Titel und Original-Link,
  sonst Link kopieren mit kurzer Bestätigung. Ohne JavaScript nicht sichtbar.
  Done: Test für beide Wege.

## C. Cluster sichtbar machen (später)

- [ ] **9. „Auch bei …“** — `feature/news-cluster-sources`
  Files: `backend/internal/store/queries/news.sql`, sqlc, `backend/internal/service/news.go`,
  `backend/internal/httpapi/dto.go`, `frontend/components/NewsList.tsx`
  What: `ListNews` liefert je Cluster zusätzlich die anderen Mitglieder (Quelle, Titel, Link), höchstens 5.
  Die Website zeigt bei mehr als einer Quelle „· 3 Quellen“ und darunter „Auch bei: APA-OTS · FinanzNachrichten“.
  Titel und Links sind öffentlich erlaubt (Rechtsregel 1). Optional eine Liste „Meistberichtet heute“ nach
  Zahl der Quellen, ohne Wertung.
  Wann: sobald mehr Cluster mehrere Quellen haben (nach dem EU-Plan oder weiteren Newsfeeds).
  Done: DTO-Test grün; Store-Test mit einem Cluster aus 3 Quellen.

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
