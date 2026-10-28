# Design — aktien-news.at News-Hub (MVP)

Status: Entwurf, 2026-10-28. Input für `/plan`.

## 1. Ziel

Ein Hub, der Nachrichten zu Aktien aus ATX/Wiener Börse, DAX/Deutschland und
US-Märkten einsammelt, dedupliziert, Firmen zuordnet und intern klassifiziert
(Event-Typ, Sentiment). Er liefert die Daten öffentlich als filterbare
Link-Liste und intern über MCP an boerseonline — rechtssicher und point-in-time.

### Akzeptanzkriterien

1. Ingest läuft automatisch für: FinanzNachrichten.de (Ad-hoc + Index-Feeds DE/AT/US),
   APA-OTS RSS (`/rss/finanzen`, `/rss/wirtschaft`), SEC EDGAR 8-K, Business Wire / PR Newswire RSS,
   IR-RSS-Feeds der Emittenten, wo vorhanden. Alpha Vantage NEWS_SENTIMENT ist optional (Baseline).
   Die Wiener Börse ist ausgeschlossen (ToS-Verbot); siehe `docs/research-sources.md`.
2. Dieselbe Meldung aus mehreren Quellen erscheint als **ein** Cluster.
3. Jeder Artikel ist mit 0..n Firmen verknüpft (Confidence, Status).
4. Jeder abgeleitete Fakt (Link, Klassifikation) hat einen eigenen
   Wissenszeitpunkt. Eine Abfrage mit `as_of=T` liefert exakt das, was zum
   Zeitpunkt T bekannt war.
5. Die öffentliche Seite zeigt **nur** Titel, Quelle, Link, Zeitpunkt und Ticker.
   Sentiment, Event-Typ und Text erscheinen nicht.
6. boerseonline ruft `search_news` und `ticker_sentiment` über den mcp-hub auf.
7. `docker compose up` startet das komplette System lokal. Die App bootet ohne
   API-Keys; key-abhängige Quellen werden übersprungen.

### Out of scope

Kauf- und Verkaufssignale, Volltext oder Snippets öffentlich, Backtest (läuft in
boerseonline), E-Mail-Digest, User-Accounts, Admin-UI (der Review-Queue reicht
im MVP SQL/CLI), boersennews.at außer einer 301-Weiterleitung.

## 2. Ist-Zustand

Das Repo ist leer, der alte Python-Stand liegt in der Git-History (`f7735dd`).
Übernommen werden **Konzepte**, kein Code:
- die Confidence-basierte Matching-Engine (Schwellen 0.85 und 0.40, `rejected`
  wird nie wiederbelebt),
- „Adapter sind reine Parser“: Fixtures-Tests, kein Live-Netz in Tests.

boerseonline (`src/aistock/mcp_client.py`) spricht JSON-RPC über Streamable
HTTP mit einer **mcp-hub-Proxy-Instanz** (`clients/boerseonline/` im mcp-hub-Repo).
Der Client erwartet im Body JSON (`resp.json()`), kein SSE. Tools tragen ein
Child-Präfix (`db_`, `email_` …). → aktien-news wird ein **weiterer Child-Server**
dieser Proxy-Instanz, z. B. mit dem Präfix `news` (`news_search_news`, `news_ticker_sentiment`).

## 3. Architektur

```
docker-compose
├── db        postgres:16 (pg_trgm, citext)
├── worker    Go  cmd/worker  – Scheduler: fetch → dedupe → match → classify
├── api       Go  cmd/api     – öffentliche REST-API (:8080) + MCP (:8081, nur internes Netz)
└── web       Next.js (TS)    – SSR für SEO, liest nur die öffentliche REST-API
(+ Reverse Proxy/TLS beim Deployment, z. B. Caddy; boersennews.at → 301)
```

Go-Layout (ein Modul):

```
backend/
  cmd/api, cmd/worker, cmd/seed
  internal/
    config/            env-basiert
    store/             pgx + sqlc-generierte Queries, goose-Migrationen
    ingest/<source>/   reine Parser: Parse(payload) -> []RawArticle
    ingest/fetch/      HTTP/RSS-Fetching, robots/TDM-Check, Rate-Limit
    dedupe/            URL-Normalisierung, Content-Hash, SimHash-Clustering
    matching/          Entity-Linking (Ticker/ISIN/Name)
    classify/          Classifier-Interface + LLM-Implementierung
    httpapi/           öffentliche REST-Handler (dünn)
    mcpserver/         MCP-Tools (dünn, rufen dieselben Services)
    service/           Abfrage-Logik (news search, sentiment aggregate) – geteilt von REST + MCP
frontend/              Next.js App Router
```

Layering wie bisher: Handler und MCP-Tools sind dünn, Logik liegt in `service/`, DB-Zugriff nur über `store/`.

### 3.1 Architekturstil

**Modularer Monolith:** ein Go-Modul, eine Datenbank, mehrere Binaries
(`cmd/api`, `cmd/worker`, `cmd/seed`). Bewusst ohne Microservices und ohne Message Queue.

- **Schichten:** Handler/MCP-Tool → `service` → `store`. Handler übersetzen nur
  zwischen Transport und Service, SQL steht ausschließlich in `store`.
- **Pakete nach Fachthema** (`catalog`, `ingest`, `dedupe`, `matching`, `classify`).
  Nach Technik geschnitten sind nur die Ränder (`httpapi`, `mcpserver`, `store`).
- **Funktionaler Kern, Seiteneffekte außen:** Parser (`Parse([]byte)`) und
  Matcher (`Index.Match`) sind reine Funktionen. Netzwerk und Datenbank
  liegen in dünnen Schichten außen (`ingest/fetch`, `matching.Persist`,
  `matching.LoadIndex`). Der Kern wird mit Fixtures getestet, ohne Mocks.
- **Trennung von Lesen und Schreiben über Prozesse:** Nur der Worker schreibt
  Nachrichtendaten, `cmd/seed` die Stammdaten. API und MCP lesen nur. Es gibt
  keine getrennten Lese- und Schreibmodelle (kein volles CQRS).

### 3.2 Eingesetzte Patterns

| Pattern | Wo | Wozu |
|---|---|---|
| Adapter | ein Parser pro Quelle → gemeinsames `ingest.RawArticle` | neue Quellen ohne Änderung der Pipeline |
| Pipes & Filters | fetch → parse → dedupe → match → classify | Stufen einzeln testbar, Fehler pro Quelle isoliert |
| Strategy | `Fetcher`, `Classifier` (Fake im Test, OpenRouter produktiv) | Implementierung austauschbar |
| Table Data Gateway | sqlc-generierte `db.Queries` | typsicheres SQL ohne ORM |
| Unit of Work | `pgx.BeginFunc` (Seed, Persist) | alles oder nichts |
| Append-only / Temporal Data | `article_entities`, `classifications`, `as_of`-Abfragen | Backtests ohne Look-ahead-Bias; leichter als Event Sourcing |
| Idempotente Operationen | Upserts, `ON CONFLICT DO NOTHING` | Läufe beliebig oft wiederholbar |
| Invarianten in der Datenbank | Trigger (append-only, unveränderliches `ingested_at`), Check-Constraints | Regeln gelten auch für SQL von Hand |
| Pessimistisches Locking | Advisory-Lock pro Artikel in `matching.Persist` | keine Duplikate bei parallelen Läufen |
| Constructor Injection | `catalog.New(conn)`; Funktionen nehmen `db.DBTX` | ohne DI-Framework; Pool und Transaktion sind austauschbar |
| 12-Factor-Konfiguration | Umgebungsvariablen mit lokalen Defaults | bootet ohne API-Keys |

### 3.3 Bewusst nicht eingesetzt

| Verworfen | Grund |
|---|---|
| Microservices, Message Queue | mehr Betrieb ohne Nutzen bei diesem Volumen |
| ORM (GORM o. ä.) | Point-in-time-Abfragen brauchen kontrolliertes SQL |
| Vollständiges Hexagonal bzw. Clean Architecture (Interface pro Repository) | Tests laufen gegen echtes Postgres statt gegen Mocks; Interfaces nur, wo es mehrere Implementierungen gibt |
| DDD mit Aggregates und Domain Events | die Fachlichkeit ist überwiegend Datenverarbeitung, keine komplexe Geschäftslogik |

Regel für neuen Code: Funktionen mit DB-Zugriff nehmen `db.DBTX` (Pool oder
Transaktion), keinen konkreten `*pgxpool.Pool`. Ausnahme ist eine Funktion,
die ihre Transaktion selbst öffnet; sie nimmt einen `TxBeginner`.

## 4. Datenmodell

Alle Zeitstempel sind `timestamptz` (UTC).

**sources**
`id, key (unique), name, kind (adhoc|press_release|news_api|ir), market (AT|DE|US),
weight numeric (Ad-hoc 1.0 > Agentur 0.8 > Presse 0.6 > Blog 0.3), enabled,
fetch_interval (Minimum je Quelle, z. B. FinanzNachrichten ≥ 10 min),
text_policy (none|feed_fields|full_text), tdm_reserved bool, tdm_checked_at, terms_url, notes`
— `text_policy` legt fest, was ins LLM darf. Default ist `none` (nur Titel).
`full_text` nur für gemeinfreie oder lizenzierte Quellen (SEC, Emittenten-IR ohne Vorbehalt).
Ein erkannter TDM-Vorbehalt erzwingt `none`, unabhängig von der Konfiguration.

**companies**
`id, name, legal_name, lei null, country, created_at`

**listings**
`id, company_id → companies, isin, ticker, mic (XWBO|XETR|XNYS|XNAS …), currency,
active` — unique `(mic, ticker)`, index `isin`

**index_memberships**
`listing_id, index_key (ATX|ATX_PRIME|DAX|MDAX|SP500 …), valid_from, valid_to null`
(point-in-time, damit Filter auf historische Zusammensetzung später möglich sind)

**company_aliases**
`company_id, alias, kind (core_name|former_name|brand)` — Trigram-Index

**clusters**
`id, first_seen_at, representative_article_id, size`

**articles**
`id, source_id, external_id, url, canonical_url, title, lang (de|en),
published_at, ingested_at default now(), content_hash, simhash bigint,
cluster_id → clusters, issuer_hint null (vom Feed geliefertes Emittentenfeld)`
- unique `(source_id, external_id)`, unique `canonical_url`
- index `(published_at desc)`, index `cluster_id`

**article_texts** (nur intern, Retention)
`article_id pk, body text, fetched_at, purge_after`
— wird nur befüllt, wenn `sources.text_policy = full_text` und `tdm_reserved = false`. Ein Job löscht nach
`purge_after` (Default 30 Tage). Nie über eine API ausgeliefert.

**article_entities** (append-only für Statuswechsel)
`id, article_id, company_id, confidence, method (issuer|isin|cashtag|ticker|full_name|core_name),
status (linked|pending|rejected), decided_at, decided_by (engine|admin)`
— „aktueller Status“ = jüngste Zeile pro `(article_id, company_id)` mit `decided_at <= as_of`.

**classifications** (append-only)
`id, article_id, company_id, event_type, sentiment numeric(-1..1), confidence numeric(0..1),
novelty numeric(0..1), model, prompt_version, classified_at`
- `event_type` enum: earnings, guidance, m_and_a, rating_change, litigation,
  insider, management_change, capital_markets, dividend, regulatory, adhoc_other, other
- Neuklassifikation = neue Zeile, nie UPDATE.

**external_sentiment**
`id, provider ('alphavantage'), article_id null, listing_id, score, label, relevance,
provider_published_at, fetched_at` — Baseline, getrennt von den eigenen Scores.

**ingest_runs**
`id, source_id, started_at, finished_at, status (ok|partial|failed|skipped), fetched, new, error`

Migrationen: goose, jede mit `up` und `down`. Die CI testet up → down → up gegen einen frischen Postgres.

## 5. Pipeline (worker)

1. **Fetch** pro Quelle im eigenen Intervall, isoliert. Ein Fehler wird als WARNING
   und als `ingest_runs.status=failed` festgehalten, der Rest des Batches läuft weiter.
   Vor dem Text-Fetch werden robots.txt und der TDM-Vorbehalt (TDMRep-Header/`tdmrep.json`)
   geprüft und in `sources.tdm_reserved` gecacht.
2. **Parse** über den reinen Parser zu `RawArticle`.
3. **Dedupe exakt** über normalisierte URL (ohne utm/Tracking) und `content_hash` des
   normalisierten Titels.
4. **Cluster** per SimHash über Titel und Lead, Hamming-Distanz ≤ 3, im 72-h-Fenster.
   Repräsentant ist die Quelle mit dem höchsten `weight`, bei Gleichstand die früheste.
5. **Match** (Confidence pro Firma, höchste gewinnt):

   | Signal | Confidence |
   |---|---|
   | Emittentenfeld (`issuer_hint`: OTS `author`, Emittenten-IR-Feed, SEC-CIK) → Firma | 1.00 |
   | ISIN-Feld (FinanzNachrichten `fn:isin`) oder ISIN im Text | 0.98 |
   | `$TICKER`-Cashtag | 0.95 |
   | Ticker als ganzes Wort in Großbuchstaben (≥ 2 Zeichen, nicht auf Stoppliste) | 0.90 |
   | Voller Firmenname inkl. Rechtsform (AG, SE, N.V., Inc., plc …) | 0.90 |
   | Kernname (≥ 3 Zeichen) | 0.55 → pending |

   ≥ 0.85 wird `linked`, ≥ 0.40 wird `pending`, darunter wird der Treffer verworfen.
   Pending-Links erscheinen weder öffentlich noch über MCP.
6. **Classify** nur für den Cluster-Repräsentanten und je verlinkter Firma. Andere
   Artikel des Clusters erben das Ergebnis mit `novelty` = 1 / Rang im Cluster.
   Der Input richtet sich nach `sources.text_policy`; im MVP ist das für die meisten Quellen **nur der Titel**.
   Das LLM gibt bei reinem Titel-Input eine entsprechend niedrigere `confidence` an. Die Ausgabe ist
   strukturiert (`event_type`, `sentiment`, `confidence`).
7. **Baseline** ruft Alpha Vantage NEWS_SENTIMENT für US-Ticker ab und schreibt nach `external_sentiment`.

## 6. Endpoint-Verträge

### Öffentliche REST-API (`/api/v1`, Auth: keine)

`GET /api/v1/news`
- Query: `ticker?` (string), `index?` (ATX|DAX|…), `source?` (key), `from?`/`to?` (RFC 3339),
  `cursor?` (opak), `limit?` (1–100, Default 25)
- 200: `{ items: [{ id, title, url, source: {key, name}, published_at, tickers: [string] }], next_cursor }`
- 400: ungültiger Parameter. Ein unbekannter Ticker ist kein Fehler und liefert eine leere Liste.
- Der Response-Typ enthält **keine** Felder für Text, Snippet, Sentiment oder Event-Typ (Rechts-Invariante).

`GET /api/v1/instruments?q=` → 200 `{ items: [{ ticker, mic, name, isin }] }` (Autocomplete, max. 10)

`GET /api/v1/sources` → 200 `{ items: [{ key, name, market }] }`

`GET /healthz` → 200 `{ status: "ok" }`

Rate-Limit pro IP am Reverse Proxy, Antwort 429.

### MCP-Server (Child des mcp-hub, internes Netz)

- Transport: Streamable HTTP unter `/mcp` mit JSON-Antworten (kein SSE nötig), offizielles Go-SDK
  `github.com/modelcontextprotocol/go-sdk`.
- Auth: statisches Bearer-Token (`MCP_AUTH_TOKEN`), das der Hub-Proxy mitsendet.
  Fehlt das Token oder ist es falsch, kommt **401**. Rollen gibt es nicht, daher entfällt 403 im MVP.
- Tool-Fehler werden als `isError: true` mit Text gemeldet (so erwartet es `McpHubClient`).

`search_news`
- Args: `ticker?`, `query?` (Volltextsuche über Titel), `since` (RFC 3339, Pflicht), `until?`,
  `event_type?`, `min_confidence?` (Default 0.85), `as_of?` (Default jetzt), `limit?` (1–50, Default 20)
- Ergebnis: `[{ id, cluster_id, title, url, source, source_weight, published_at, ingested_at,
  tickers, event_type, sentiment, confidence, novelty, classified_at }]`
- Point-in-time: Es zählen nur Zeilen mit `ingested_at`, `decided_at` und `classified_at` ≤ `as_of`.

`ticker_sentiment`
- Args: `ticker` (Pflicht), `window` (`1d|7d|30d`), `as_of?`
- Ergebnis: `{ ticker, window, as_of, n_articles, n_clusters, score_weighted, score_mean,
  by_event_type: {type: {n, score}}, baseline: {provider, score, n} | null }`
- `score_weighted` = Σ(sentiment · confidence · source_weight · novelty) / Σ(confidence · source_weight · novelty)
- Bei unbekanntem Ticker: `isError`. Ohne Artikel im Fenster: `n_articles: 0`, Scores `null`.

## 7. Auth / Rechte

| Fläche | Wer | Durchsetzung |
|---|---|---|
| REST `/api/v1/*` | anonym | nur lesend, Rate-Limit |
| MCP `/mcp` | nur mcp-hub | Bearer-Token, Port nur im internen Docker-Netz bzw. per Firewall |
| DB, Review-Queue | Betreiber | SQL/CLI (`cmd/admin` später) |

## 8. Rechtliche Leitplanken als technische Invarianten

- **§ 76f UrhG (Leistungsschutzrecht):** Öffentliche DTOs haben keinen Text und kein Snippet. Ein Test sichert das ab.
- **§ 42h UrhG (Text and Data Mining):** Text wird nur bei `tdm_reserved = false` geholt und
  nach `purge_after` gelöscht. Sonst wird nur der Titel klassifiziert.
- **MAR:** Sentiment und Event-Typ gibt es nur über MCP. Die Seite trägt einen Disclaimer
  („keine Anlageberatung“).
- Die MVP-Quellen sind überwiegend **Emittenten-Mitteilungen** (Ad-hoc, OTS). Das ist
  rechtlich günstiger als Verlagsinhalte. Trotzdem gehören Nutzungsbedingungen pro Quelle
  in `sources.terms_url`/`notes`, und vor dem Launch braucht es eine juristische Prüfung.

## 9. Key Invariants

1. `classifications` und `article_entities` werden nie per UPDATE verändert, nur ergänzt.
2. `ingested_at` wird beim Insert vom Server gesetzt und nie geändert.
3. Ein `rejected`-Link wird von der Engine nie wieder auf `linked` oder `pending` gesetzt.
4. Kein öffentlicher Endpoint liefert Text, Sentiment oder Event-Typ.
5. Ein Ausfall einer Quelle stoppt keine andere Quelle.
6. Ohne API-Keys bootet alles, und die betroffenen Quellen laufen als `skipped`.
7. Tests rufen nie Live-APIs auf.

## 10. Tradeoffs

| Entscheidung | Gewählt | Verworfen, weil |
|---|---|---|
| Frontend | Next.js mit SSR | Eine Vite-SPA ist für SEO schlechter, und SEO ist ein Ziel der öffentlichen Seite |
| DB-Zugriff | pgx + sqlc + goose | GORM bringt Magie, schwer kontrollierbares SQL und erschwert point-in-time-Queries |
| API und MCP | ein Binary, zwei Ports | Getrennte Dienste bedeuten mehr Betrieb ohne Nutzen. Durch die Ports bleibt MCP vom Internet trennbar |
| Scheduler | In-Process im worker (Ticker/cron-lib) | Eine externe Queue (NATS/Redis) ist für dieses Volumen unnötig |
| Klassifikation | LLM über OpenRouter hinter Interface, nur Cluster-Repräsentanten | FinBERT bräuchte in Go eine ONNX-Runtime bzw. einen Python-Sidecar, und es gibt kein gutes deutsches Finanzmodell. Ist später austauschbar |
| Historie | append-only und `as_of` | Mit UPDATE-in-place wäre ein Backtest ohne Look-ahead-Bias unmöglich |
| Dedupe | SimHash im Zeitfenster | Embeddings bringen Kosten und Infrastruktur, ihr Mehrwert ist bei Pressemitteilungen gering |
| Stammdaten | Seed-CSV im Repo und `cmd/seed` | Eine Live-Stammdaten-API ist zusätzliche Abhängigkeit und kostet |

## 11. Teststrategie

**Must-have**
- Parser pro Quelle mit echten Fixtures in `testdata/`, inklusive kaputter und leerer Feeds.
- Matching als Table-Tests: Issuer, ISIN, Cashtag, Ticker-Stoppliste (z. B. „AG“, „CEO“, „EU“), Rechtsform-Varianten, Umlaute (ß/ss, ö/oe), pending vs. linked, rejected bleibt rejected.
- Dedupe: URL-Normalisierung, dieselbe Ad-hoc über EQS und OTS ergibt einen Cluster, Repräsentanten-Wahl.
- **Point-in-time:** Eine Klassifikation, die nach `as_of` erstellt wurde, ist unsichtbar. Ein Link, der nach `as_of` auf rejected gesetzt wurde, zählt zu `as_of` noch.
- `ticker_sentiment`-Aggregation mit festen Daten (Gewichte, leeres Fenster, unbekannter Ticker).
- **Rechts-Invariante:** Ein JSON-Schema-Snapshot der öffentlichen Responses enthält kein Text-, Sentiment- oder Event-Feld.
- MCP: 401 ohne Token. `tools/list` und `tools/call` laufen über HTTP mit JSON-Antwort, so wie `McpHubClient` sie sendet.
- Classifier-Fake: Der Pipeline-Test läuft ohne LLM. Ein Test prüft das Parsen des LLM-Outputs, inklusive kaputtem JSON.

**Integration** (Postgres im Container, `TEST_DATABASE_URL`)
- Store-Queries, Migrationen up/down/up, Drift-Check zwischen sqlc-Schema und Migrationen.
- Compose-Smoke: `docker compose up`, `/healthz`, ein Fixture-Ingest, die Liste ist nicht leer.

**Frontend:** Vitest + RTL für Filter und Liste, `lint` und `build` als CI-Gate.

**Nicht unit-testbar:** echtes Feed-Verhalten (Formatwechsel, Rate-Limits), LLM-Qualität.
Dafür gibt es ein manuelles Eval-Set (ca. 100 gelabelte DE/EN-Meldungen) als eigenen Schritt.

## 12. Entscheidungen (2026-10-28)

| # | Thema | Stand |
|---|---|---|
| 1 | LLM-Provider | **OpenRouter** (OpenAI-kompatible API). Das Modell ist konfigurierbar (`CLASSIFY_MODEL`). Ein Budget gibt es noch nicht; der Kostendeckel kommt als Config-Wert (`CLASSIFY_MONTHLY_BUDGET_USD`), und der Worker stoppt die Klassifikation beim Erreichen. |
| 2 | US-Universum | **offen**, Ziel „so viel wie möglich“. Das Design skaliert über Seed-Daten; ein Kostenhebel ist Klassifikation nur für verlinkte Cluster-Repräsentanten. |
| 3 | mcp-hub-Anbindung | **Streamable HTTP** (Annahme bestätigt), Präfix noch offen. |
| 4 | Quellenzugang | **recherchiert**, siehe `docs/research-sources.md`. MVP mit 0 $/Monat für Daten: FinanzNachrichten, OTS-RSS, SEC EDGAR, BW/PRN-RSS, Emittenten-IR. Wiener Börse ausgeschlossen. Die Klassifikation läuft überwiegend nur auf Titeln (OTS-TDM-Vorbehalt). Lizenzanfragen an APA-Comm, OeKB und Alpha Vantage gehen parallel. |
| 5 | Stammdaten | **bestätigt:** Seed aus öffentlichen Listen (Wiener Börse, Deutsche Börse, SEC `company_tickers.json`) und OpenFIGI für ISIN↔Ticker. |
