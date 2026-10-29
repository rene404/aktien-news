# Plan Phase 3: MCP-Server für boerseonline

Entwurf vom 2026-10-29. Entschieden am 2026-10-29: Phase 3 baut zuerst den MCP-Server.
Die LLM-Klassifikation (Modell, Budget, SEC-Volltexte) wird zurückgestellt (siehe unten).
Grundlage: `docs/design.md` (§6 MCP, §7, §11) und der Live-Stand vom 2026-10-29:

- Das Schema ist vorhanden: `classifications` (append-only), `article_entities` mit `decided_at`,
  `articles.ingested_at`. Solange es keine Klassifikationen gibt, liefern die Tools Meldungen,
  Cluster und Ticker, aber Event-Typ, Sentiment und Scores bleiben `null`.
- boerseonline spricht JSON-RPC 2.0 per HTTP-POST mit der mcp-hub-Proxy-Instanz
  (`mcp-hub/clients/boerseonline/proxy.yaml`). Erwartet wird JSON im Body (kein SSE), 401/403 als Auth-Fehler,
  Tool-Fehler als `isError`. Neue Child-Server in `proxy.yaml` brauchen Maintainer-Sign-off.

Jede Aufgabe läuft auf einem eigenen `feature/phase3-*`-Branch per TDD. Für alle gilt der Gate
`docker compose --profile test run --rm backend-test sh -c "gofmt -l .; go vet ./... && go test ./..."`.

- [x] **1. Point-in-time-Service: `SearchNews` und `TickerSentiment`**
  Files: `backend/internal/service/insight.go`, `backend/internal/service/insight_test.go`, `backend/internal/store/queries/insight.sql`
  What: `SearchNews(ticker?, query?, since, until?, eventType?, minConfidence=0.85, asOf=jetzt, limit 1–50)` liefert Artikel mit `id`, `cluster_id`, Titel, URL, Quelle, `source_weight`, `published_at`, `ingested_at`, Tickern und, falls vorhanden, Event-Typ, Sentiment, Confidence, Novelty und `classified_at`. `minConfidence` filtert die Link-Confidence. `eventType` setzt eine Klassifikation voraus. `TickerSentiment(ticker, window 1d|7d|30d, asOf)` liefert `n_articles`, `n_clusters`, `score_weighted` nach design §6, `score_mean`, `by_event_type` und `baseline: null`. Point-in-time bedeutet: Es zählen nur `ingested_at`, `decided_at` (jüngste Zeile in `article_entities`) und `classified_at` ≤ `as_of`. Bei mehreren Klassifikationen gilt die jüngste ≤ `as_of`.
  Command: `docker compose --profile test run --rm backend-test go test ./internal/service/`
  Done: Ein Artikel mit `ingested_at` nach `as_of` ist unsichtbar. Ein Link, der nach `as_of` rejected wurde, zählt zu `as_of` noch und danach nicht mehr. Pending zählt nie. Eine Klassifikation nach `as_of` ist unsichtbar, eine neuere ersetzt eine ältere. `score_weighted` stimmt mit festen Daten exakt. Ohne Klassifikationen gibt es Zählungen, aber Scores `null`. Ein unbekannter Ticker ergibt `ErrNotFound`, ungültige Argumente einen `ValidationError`. `query` sucht im Titel ohne Groß-/Kleinschreibung.

- [x] **2. MCP-Server**
  Files: `backend/internal/mcpserver/server.go`, `backend/internal/mcpserver/server_test.go`, `backend/cmd/api/main.go`, `backend/internal/config/config.go`, `backend/internal/config/config_test.go`, `backend/go.mod`, `docker-compose.yml`, `.env.example`
  What: `mcpserver` registriert die Tools `search_news` und `ticker_sentiment` mit dem offiziellen Go-SDK `github.com/modelcontextprotocol/go-sdk` (Version und JSON-Response-Option vorher in der Doku prüfen). Die Tools sind dünn: Argumente lesen, `service` aufrufen, Ergebnis formen. Der Transport ist Streamable HTTP unter `/mcp` auf eigenem Port (`MCP_ADDR`, Default `:8081`) mit JSON-Antworten, zustandslos. Auth läuft über eine Bearer-Middleware mit zeitkonstantem Vergleich (`MCP_AUTH_TOKEN`). Ohne Token startet MCP nicht, und die API bootet trotzdem. In Compose ist der Port nicht auf den Host gemappt, sondern nur im Docker-Netz erreichbar.
  Command: `docker compose --profile test run --rm backend-test go test ./internal/mcpserver/ ./internal/httpapi/ ./internal/config/`
  Done: Ohne oder mit falschem Token kommt 401. `initialize`, `tools/list` und `tools/call` per einfachem HTTP-POST (wie `McpHubClient`) liefern JSON. `tools/call` mit unbekanntem Ticker liefert `isError: true` mit Text. `as_of` wird durchgereicht. Der DTO-Rechtstest der öffentlichen API bleibt unverändert grün, und `/api/v1/*` liefert weiterhin kein Sentiment. Live antwortet `tools/list` aus einem Container im Compose-Netz.

- [ ] **3. Anbindung an mcp-hub (boerseonline)**
  Files: `docker-compose.yml`, `../mcp-hub/clients/boerseonline/proxy.yaml`, `../mcp-hub/clients/boerseonline/docker-compose.client.yml`, `AGENTS.md`, `README.md`
  What: aktien-news hängt sich an ein externes Docker-Netz, das die Proxy-Instanz erreicht. In `proxy.yaml` kommt der Server `news` dazu (`http://<api-host>:8081/mcp`) mit Bearer-Header. Die Änderung im mcp-hub-Repo erfordert laut Datei einen Maintainer-Sign-off, sie läuft deshalb dort auf eigenem Branch und wird erst nach Freigabe gemergt.
  Command: `docker compose up -d --build api` · `tools/list` gegen die Proxy-Instanz
  Done: Die Proxy-Instanz listet `news_search_news` und `news_ticker_sentiment`. `call_tool("news_search_news", {"ticker": "VOE", "since": …})` aus dem boerseonline-Container liefert Meldungen.

Umsetzungsnotizen (Aufgaben 1–2): `search_news` liefert einen Eintrag pro Artikel **und** verlinkter Firma
(Klassifikationen hängen am Paar), mit `company` und deren `tickers`. Beide Tools antworten mit einem Objekt
(`{items: [...]}` bzw. dem Sentiment-Objekt) als `structuredContent` plus JSON-Text in `content`.
Go-SDK `v1.8.0`, `Stateless` + `JSONResponse`; `tools/list` geht ohne `initialize`, wie `McpHubClient` es sendet.

## Zurückgestellt: Klassifikation

Erst nach einer Entscheidung zu Modell und Budget:

- Classifier-Kern (Prompt, Ausgabe-Schema, Parser, Fake) und OpenRouter-Client mit Kostentabelle und Monatsdeckel.
- Classify-Stufe im Worker: nur Cluster-Repräsentant je fest verlinkter Firma, Mitglieder erben mit `novelty = 1 / Rang`.
- SEC-Volltexte (8-K/6-K-Hauptdokument) über das vorhandene `StoreText`.
- Eval-Set (ca. 100 gelabelte Titel) und Live-Abnahme.

## Bewusst nicht in Phase 3

- **Alpha Vantage Baseline:** Die kommerzielle Nutzung ist ungeklärt (Recherche). `baseline` bleibt `null`.
- Review-UI oder `cmd/admin` für pending-Links: Dafür reicht weiter SQL.
- Öffentliche API und Website: keine Änderung. Sentiment bleibt intern (MAR).
