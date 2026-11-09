# aktien-news.at

News-Hub für Aktien aus Österreich (ATX), Deutschland (DAX), Frankreich (CAC 40),
Spanien (IBEX 35), den Niederlanden (AEX), Schweden, Finnland, Dänemark
(OMXS30, OMXH25, OMXC25), Großbritannien (FTSE 100) und den USA (S&P 500).
Meldungen werden automatisch eingesammelt, dedupliziert und Unternehmen
zugeordnet. Öffentlich gibt es eine filterbare Liste (Titel, Quelle, Link);
intern liefert ein MCP-Server Nachrichten und Sentiment an boerseonline.

> Keine Anlageberatung. Angezeigt werden nur Überschriften mit Links zur Originalquelle.

## Schnellstart

Voraussetzung: Docker (mit Compose). Go und Node werden nicht lokal benötigt.

```bash
cp .env.example .env      # optional
docker compose --profile tools run --rm --build seed   # Schema + Stammdaten
docker compose up -d --build                          # db, api, worker, web
```

US-Stammdaten (optional, alle NYSE-/Nasdaq-Titel) und SEC-8-K-Meldungen brauchen
`SEC_USER_AGENT` mit Kontaktadresse in `.env` (sec.gov ist im AustriaCard-Netz gesperrt):

```bash
docker compose --profile tools run --rm -v <ordner>:/sec:ro seed -sec-file /sec/company_tickers_exchange.json
```

- Webseite: http://localhost:3020 (über devproxy: http://aktien.localhost)
- API: http://localhost:8090/api/v1/news
- Ingest-Log: `docker compose logs -f worker`

Tests:

```bash
docker compose --profile test run --rm backend-test   # Go
docker compose --profile test run --rm web-test       # Frontend: lint, test, build
```

## Märkte, Quellen und Tabs

| Markt | Index-Tab | Börse | Pflichtmeldungen | Weitere Quellen |
|---|---|---|---|---|
| AT | ATX | Wiener Börse (XWBO) | OeKB OAM | APA-OTS, FinanzNachrichten, voestalpine IR |
| DE | DAX | Xetra (XETR) | – (Ad-hoc über FinanzNachrichten) | FinanzNachrichten, Telekom IR |
| FR | CAC 40 | Euronext Paris (XPAR) | AMF über DILA Open Data | – |
| ES | IBEX 35 | Bolsa de Madrid (XMAD) | CNMV | – |
| NL | AEX | Euronext Amsterdam (XAMS) | AFM Directors' Dealings | – |
| NORDIC | OMXS30, OMXH25, OMXC25 | Nasdaq Stockholm, Helsinki, Copenhagen (XSTO, XHEL, XCSE) | Nasdaq Nordic Company Announcements | Investor News (Nasdaq Nordic) |
| UK | FTSE 100 | London Stock Exchange (XLON) | FCA National Storage Mechanism | – |
| US | S&P 500 | NYSE, Nasdaq (XNYS, XNAS) | SEC EDGAR 8-K, 6-K | PR Newswire, FinanzNachrichten |

Dazu die Tabs „Alle“, „Ad-hoc“ und „Pflichtmeldungen“ (Quellenart `filing`).
Index-Tabs zeigen Nachrichten, Pressemitteilungen und Ad-hoc-Meldungen ihrer Werte, aber keine Filings;
die stehen unter „Pflichtmeldungen“ und auf den Aktienseiten. Die Aktienliste kennt dieselben Indizes.

Nasdaq Nordic und die FCA laufen nur zum privaten Testen: Vor einem öffentlichen Betrieb müssen die
Anfrage bei Nasdaq und die Rechtsprüfung der FCA-Nutzung erledigt sein (`docs/plan-eu-sources.md`, Entscheidung 3).
Die FCA-Quelle geht nicht ans LLM (`text_policy = no_llm`).

## API (öffentlich, ohne Auth)

| Endpoint | Beschreibung |
|---|---|
| `GET /api/v1/news?ticker=&index=&source=&from=&to=&cursor=&limit=` | Meldungen mit sicher verknüpfter Firma, neueste zuerst, ein Eintrag pro Cluster gleicher Meldungen (Titel/Link/Quelle der wichtigsten Quelle, Ticker aller Mitglieder; `also`: dieselbe Meldung bei bis zu 5 anderen Quellen, je Titel/Link/Quelle/Zeit). `source`/`from`/`to` beziehen sich auf diesen Eintrag. `from`/`to` RFC 3339, `limit` 1–100 (Default 25), `cursor` aus `next_cursor` |
| `GET /api/v1/instruments?q=` | Suche nach Ticker, Name oder ISIN (max. 10; exakter Ticker, dann Namensanfang, dann Index-Werte zuerst) |
| `GET /api/v1/instruments/{ticker}` | Name, ISIN, Börse, Indizes einer Aktie (404 wenn unbekannt) |
| `GET /api/v1/sources` | aktive Quellen mit letztem erfolgreichem Abruf (`last_ok_at`) |
| `GET /api/v1/tickers/top?from=&index=&kind=&limit=` | meistgenannte Aktien seit `from` (Pflicht, RFC 3339): Ticker, Name, Anzahl Meldungen; `index` nur dessen Werte, `kind` Quellenarten, `limit` 1–20 (Default 10) |
| `GET /api/v1/indices/{index}/instruments?from=&kind=` | alle aktuellen Werte eines Index mit Anzahl Meldungen (auch 0), nach Name (404 wenn unbekannt) |
| `GET /healthz` | Health-Check |

Gezählt wird wie im Feed: ein Cluster pro Aktie, Zeit und Quellenart des wichtigsten Artikels.
Antworten enthalten bewusst nur Titel, Quelle, Link, Zeit, Ticker und Zählungen – keine Textauszüge, kein Sentiment.

## MCP (intern, für mcp-hub)

Mit `MCP_AUTH_TOKEN` in `.env` startet die API zusätzlich einen MCP-Server (Streamable HTTP, JSON-Antworten,
zustandslos) auf Port 8081. Er ist nicht auf den Host gemappt und verlangt `Authorization: Bearer <token>`.
mcp-hub erreicht ihn als `http://aktien-news-api:8081/mcp` über das Netz `aktien-news-mcp`, dem nur der
boerseonline-Client-Proxy beitritt. Dort heißen die Tools `news_search_news` und `news_ticker_sentiment`;
der Proxy liest den Token aus `NEWS_MCP_TOKEN` (`mcp-hub/clients/boerseonline/.env`, gleicher Wert wie hier).
aktien-news muss laufen, bevor der Client-Proxy startet, weil er das Netz als `external` erwartet.

| Tool | Beschreibung |
|---|---|
| `search_news` | `since` (Pflicht), `until`, `ticker`, `query`, `event_type`, `min_confidence` (0.85), `as_of`, `limit` (1–50, 20). Ein Eintrag pro Artikel und verlinkter Firma |
| `ticker_sentiment` | `ticker` (Pflicht), `window` (`1d`/`7d`/`30d`), `as_of`. Zählungen und gewichteter Score |

`as_of` liefert den Wissensstand zu diesem Zeitpunkt (Links, Klassifikationen, Ingest). Solange nichts
klassifiziert ist, sind Event-Typ, Sentiment und Scores `null`.

## Doku

- [AGENTS.md](AGENTS.md) – Regeln, Befehle, Architektur
- [docs/design.md](docs/design.md) – Design des MVP
- [docs/research-sources.md](docs/research-sources.md) – Datenquellen und Lizenzen
- [docs/plan-eu-sources.md](docs/plan-eu-sources.md) – Pflichtmeldungen aus FR, ES, NL, Nordics und UK (abgeschlossen)
- [docs/plan-phase2.md](docs/plan-phase2.md) – Phase 2 (abgeschlossen): mehr Quellen, US-Stammdaten, Clustering
- [docs/plan-phase1.md](docs/plan-phase1.md) – Phase 1
