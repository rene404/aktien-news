# aktien-news.at

News-Hub für Aktien aus Österreich (ATX), Deutschland (DAX) und den USA.
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

## API (öffentlich, ohne Auth)

| Endpoint | Beschreibung |
|---|---|
| `GET /api/v1/news?ticker=&index=&source=&from=&to=&cursor=&limit=` | Meldungen mit sicher verknüpfter Firma, neueste zuerst, ein Eintrag pro Cluster gleicher Meldungen (Titel/Link/Quelle der wichtigsten Quelle, Ticker aller Mitglieder). `source`/`from`/`to` beziehen sich auf diesen Eintrag. `from`/`to` RFC 3339, `limit` 1–100 (Default 25), `cursor` aus `next_cursor` |
| `GET /api/v1/instruments?q=` | Suche nach Ticker, Name oder ISIN (max. 10) |
| `GET /api/v1/instruments/{ticker}` | Name, ISIN, Börse, Indizes einer Aktie (404 wenn unbekannt) |
| `GET /api/v1/sources` | aktive Quellen |
| `GET /healthz` | Health-Check |

Antworten enthalten bewusst nur Titel, Quelle, Link, Zeit und Ticker – keine Textauszüge, kein Sentiment.

## MCP (intern, für mcp-hub)

Mit `MCP_AUTH_TOKEN` in `.env` startet die API zusätzlich einen MCP-Server (Streamable HTTP, JSON-Antworten,
zustandslos) unter `http://api:8081/mcp`. Er ist nur im Docker-Netz erreichbar und verlangt
`Authorization: Bearer <token>`.

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
- [docs/plan-phase2.md](docs/plan-phase2.md) – Phase 2 (abgeschlossen): mehr Quellen, US-Stammdaten, Clustering
- [docs/plan-phase1.md](docs/plan-phase1.md) – Phase 1
