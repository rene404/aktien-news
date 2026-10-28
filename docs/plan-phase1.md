# Plan Phase 1: Fundament + erste vertikale Scheibe

Freigegeben am 2026-10-28. Grundlage: `docs/design.md`, `docs/research-sources.md`.
Go und Node sind lokal nicht installiert; die ganze Toolchain läuft in Docker.

- [x] **1. Compose-Gerüst, Go-Modul, `/healthz`, CI, AGENTS.md**
  Files: `docker-compose.yml`, `backend/Dockerfile`, `backend/go.mod`, `backend/cmd/api/main.go`, `backend/internal/config/config.go`, `backend/internal/httpapi/router.go`, `backend/internal/httpapi/health.go`, `backend/internal/httpapi/health_test.go`, `.github/workflows/ci.yml`, `.gitignore`, `.env.example`, `AGENTS.md`, `CLAUDE.md`, `README.md`
  Command: `docker compose --profile test run --rm backend-test go test ./internal/httpapi/` · `docker compose up -d --build api && curl -s localhost:8090/healthz`
  Done: Test grün, curl liefert `{"status":"ok"}`.

- [x] **2. Schema-Migration + Store + Roundtrip-Test**
  Files: `backend/internal/store/migrations/00001_init.sql`, `backend/internal/store/migrate.go`, `backend/internal/store/sqlc.yaml`, `backend/internal/store/queries/articles.sql`, `backend/internal/store/queries/catalog.sql`, `backend/internal/store/db/` (sqlc), `backend/internal/store/migrate_test.go`, `backend/internal/store/testdb/testdb.go`
  Command: `docker compose --profile test run --rm backend-test go test ./internal/store/`
  Done: up → down → up grün, Insert/Select-Smoke-Test grün, doppeltes `(source_id, external_id)` wird abgelehnt.

- [x] **3. Stammdaten-Seed**
  Files: `backend/seed/companies.csv`, `backend/seed/listings.csv`, `backend/seed/index_memberships.csv`, `backend/seed/sources.csv`, `backend/internal/catalog/seed.go`, `backend/internal/catalog/seed_test.go`, `backend/cmd/seed/main.go`
  Command: `docker compose --profile test run --rm backend-test go test ./internal/catalog/`
  Done: Seed ist idempotent; Lookups per ISIN und per `(mic, ticker)` sind korrekt.

- [ ] **4. FinanzNachrichten-Parser + exaktes Dedupe**
  Files: `backend/internal/ingest/raw.go`, `backend/internal/ingest/finanznachrichten/parse.go`, `backend/internal/ingest/finanznachrichten/parse_test.go`, `backend/internal/ingest/finanznachrichten/testdata/adhoc.xml`, `backend/internal/ingest/finanznachrichten/testdata/empty.xml`, `backend/internal/ingest/finanznachrichten/testdata/broken.xml`, `backend/internal/dedupe/normalize.go`, `backend/internal/dedupe/normalize_test.go`
  Command: `docker compose --profile test run --rm backend-test go test ./internal/ingest/... ./internal/dedupe/`
  Done: ISIN wird gelesen; ein leerer Feed ergibt 0 Artikel ohne Fehler, ein kaputter Feed einen Fehler; `RawArticle` hat kein Feld für einen Textauszug; `utm_*`-Varianten werden gleich normalisiert.

- [ ] **5. Matching-Engine + append-only Entity-Links**
  Files: `backend/internal/matching/match.go`, `backend/internal/matching/normalize.go`, `backend/internal/matching/match_test.go`, `backend/internal/matching/persist.go`, `backend/internal/matching/persist_test.go`, `backend/internal/store/queries/entities.sql`
  Command: `docker compose --profile test run --rm backend-test go test ./internal/matching/`
  Done: Table-Tests grün (Stoppliste, Umlaute, Rechtsformen); keine Duplikate; `rejected` bleibt `rejected`.

- [ ] **6. Worker: Fetch-Runner mit Isolation pro Quelle + `ingest_runs`**
  Files: `backend/internal/ingest/fetch/http.go`, `backend/internal/ingest/runner.go`, `backend/internal/ingest/runner_test.go`, `backend/cmd/worker/main.go`, `docker-compose.yml`
  Command: `docker compose --profile test run --rm backend-test go test ./internal/ingest/` · `docker compose up -d --build worker && docker compose logs worker --tail 20`
  Done: Ein Lauf `ok` und ein Lauf `failed` sind protokolliert; ein zweiter Lauf erzeugt 0 neue Artikel; ein Live-Lauf ist im Log sichtbar.

- [ ] **7. Öffentliche REST-API**
  Files: `backend/internal/service/news.go`, `backend/internal/service/news_test.go`, `backend/internal/httpapi/news.go`, `backend/internal/httpapi/news_test.go`, `backend/internal/httpapi/dto.go`, `backend/internal/httpapi/dto_test.go`, `backend/internal/store/queries/news.sql`
  Command: `docker compose --profile test run --rm backend-test go test ./internal/service/ ./internal/httpapi/`
  Done: Filter, Pagination und 400 sind getestet; pending-Links sind unsichtbar; der Rechts-Invarianten-Test (keine Felder für Text, Sentiment oder Event-Typ) ist grün.

- [ ] **8. Next.js-Webseite**
  Files: `frontend/package.json`, `frontend/Dockerfile`, `frontend/next.config.ts`, `frontend/app/layout.tsx`, `frontend/app/page.tsx`, `frontend/app/aktie/[ticker]/page.tsx`, `frontend/components/NewsList.tsx`, `frontend/components/NewsList.test.tsx`, `frontend/components/Filters.tsx`, `frontend/components/Filters.test.tsx`, `frontend/components/Disclaimer.tsx`, `frontend/lib/api.ts`, `frontend/vitest.config.ts`, `docker-compose.yml`, `.github/workflows/ci.yml`
  Command: `docker compose --profile test run --rm web-test sh -c "npm ci && npm run lint && npm run test && npm run build"` · `docker compose up -d --build && curl -s localhost:3000 | grep -c "<a "`
  Done: lint, test und build grün; das SSR-HTML enthält Links mit `target="_blank"` und keinen Textauszug.

## Ausblick

- **Phase 2:** OTS, SEC EDGAR 8-K, BW/PRN, IR-Feeds; US-Stammdaten-Import (SEC/Nasdaq/OpenFIGI); SimHash-Clustering; TDM-Check, `text_policy` und Retention-Purge.
- **Phase 3:** OpenRouter-Classifier mit Budgetdeckel; Novelty; Alpha-Vantage-Baseline; MCP-Server (`search_news`, `ticker_sentiment`, `as_of`); Caddy, TLS, 301 von boersennews.at; Compose-Smoke-Test.
