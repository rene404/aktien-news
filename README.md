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
docker compose up -d --build
curl localhost:8090/healthz
```

Tests:

```bash
docker compose --profile test run --rm backend-test
```

## API (öffentlich, ohne Auth)

| Endpoint | Beschreibung |
|---|---|
| `GET /api/v1/news?ticker=&index=&source=&from=&to=&cursor=&limit=` | Meldungen mit sicher verknüpfter Firma, neueste zuerst. `from`/`to` RFC 3339, `limit` 1–100 (Default 25), `cursor` aus `next_cursor` |
| `GET /api/v1/instruments?q=` | Suche nach Ticker, Name oder ISIN (max. 10) |
| `GET /api/v1/instruments/{ticker}` | Name, ISIN, Börse, Indizes einer Aktie (404 wenn unbekannt) |
| `GET /api/v1/sources` | aktive Quellen |
| `GET /healthz` | Health-Check |

Antworten enthalten bewusst nur Titel, Quelle, Link, Zeit und Ticker – keine Textauszüge, kein Sentiment.

## Doku

- [AGENTS.md](AGENTS.md) – Regeln, Befehle, Architektur
- [docs/design.md](docs/design.md) – Design des MVP
- [docs/research-sources.md](docs/research-sources.md) – Datenquellen und Lizenzen
- [docs/plan-phase1.md](docs/plan-phase1.md) – aktueller Umsetzungsplan
