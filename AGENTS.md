# AGENTS.md — Aktien News

Authoritative working agreement for agents and contributors. The README is the
user-facing overview; this file is the rulebook. Design rationale lives in
`docs/design.md`, source/licensing research in `docs/research-sources.md`, the
current task plan in `docs/plan-phase1.md`.

## What this is

aktien-news.at: a news hub for ATX/Wiener Börse, DAX/Germany and US stocks.
It ingests news, deduplicates and clusters it, links it to companies, and (Phase 3)
classifies event type + sentiment. It serves:
- a **public** site + REST API: title, source, link, time, tickers — nothing else;
- an **internal** MCP server (child of mcp-hub) for boerseonline: news incl.
  sentiment, point-in-time via `as_of`.

**Stack:** Go 1.26 (stdlib `net/http`, pgx, sqlc, goose) · PostgreSQL 16 ·
Next.js (TS) · Docker Compose. **Go and Node are not installed on the host —
all tooling runs in containers.**

## Layout

```
backend/
  cmd/api, cmd/worker, cmd/seed   # entrypoints, wiring only
  internal/
    config/       # env settings, all with local defaults
    httpapi/      # public REST handlers — THIN
    mcpserver/    # MCP tools — THIN (Phase 3)
    service/      # query logic shared by REST + MCP
    store/        # migrations (goose), sqlc queries, generated code
    ingest/<src>/ # pure parsers: Parse([]byte) -> []RawArticle
    ingest/fetch/ # live HTTP fetching, robots/TDM checks
    dedupe/ matching/ classify/ catalog/
  seed/           # CSV master data
frontend/         # Next.js App Router (SSR)
docs/             # design, research, plans
```

## Layering rules

- Architecture style and patterns: `docs/design.md` §3.1–3.3 (modular monolith,
  functional core / imperative shell, append-only temporal data).
- Handlers and MCP tools are thin: parse → call `service` → shape response.
- DB access only through `internal/store`. No SQL in handlers.
- Public DTOs (`httpapi/dto.go`) are separate from store types.
- Ingest parsers are pure and fixture-tested; networking lives in `ingest/fetch`.

## Legal invariants (do not weaken without sign-off)

1. Public responses never contain body text, snippets, feed `description`,
   sentiment or event type. A test enforces this on the DTOs.
2. Only titles go to the LLM unless `sources.text_policy = full_text`; a
   detected TDM reservation forces `none`. Stored text is purged after
   `purge_after`.
3. Respect each source's terms: FinanzNachrichten ≥ 10 min polling, links open
   in a new window, no frames. Wiener Börse is not scraped (ToS).
4. Sentiment/event type are internal (MCP only). The site shows a
   "keine Anlageberatung" disclaimer.

## Data invariants

- `classifications` and `article_entities` are append-only (no UPDATE).
- `articles.ingested_at` is set by the DB on insert and never changes.
- The matcher never resurrects a `rejected` link; it only upgrades `pending → linked`.
- A feed-supplied ISIN is only issuer-grade for `adhoc`/`filing` sources. In news
  feeds (e.g. FN index feeds) it is a loose tag: linked only when the title
  also names the company, otherwise pending (design §5).
- One failing source never aborts another; failures log at WARNING and land in `ingest_runs`.
- The stack boots without API keys; key-dependent sources are `skipped`.

## Commands (repo root)

```bash
docker compose up -d --build                      # db + api (:8090) + worker [+ web later]
docker compose logs -f worker                     # ingest runs (one line per source)
docker compose --profile test run --rm backend-test                       # go test ./...
docker compose --profile test run --rm backend-test go test ./internal/httpapi/
docker compose --profile test run --rm backend-test sh -c "go vet ./... && go test ./..."
docker compose --profile tools run --rm --build seed   # migrate + load backend/seed/*.csv (idempotent)
docker compose --profile tools run --rm sqlc      # regenerate internal/store/db
docker compose --profile tools run --rm sqlc diff # CI: generated code must be current
docker compose --profile test down -v             # clean up incl. volumes
```

- Schema changes: add a new goose file in `internal/store/migrations/`
  (with `-- +goose Down`), then regenerate sqlc. Never edit an applied migration.
- `go mod tidy` fails behind the corporate firewall on goose's test-only
  dependency (`modernc.org/sqlite`); use `go mod tidy -e` locally.

Host ports: Postgres 5433, API `${API_PORT:-8090}` (8080 is taken on the dev box).

## Testing

- Go `testing` package, table tests; integration tests use `TEST_DATABASE_URL`
  (a fresh database per test, see `internal/store/testdb`).
- Tests never hit live external services — fixtures in `testdata/`.
- Test observable behaviour; every bug fix gets a regression test.
- Migrations need `up` and `down`; the store tests run up → down → up.

## Workflow

- **Gitflow.** `develop` is the main working branch; `main` only receives
  releases (`release/*`) and `hotfix/*`. Never commit directly to `main` or `develop`.
  - Branch off `develop`: `feature/<topic>` (e.g. `feature/phase1-seed`),
    `fix/<topic>`, `chore/<topic>`.
  - Small commits: one TDD cycle (red → green → refactor) per commit.
  - Merge back with `git merge --no-ff` into `develop`, one feature branch per
    plan task; delete the branch after merging.
- Don't push unless asked.
- Before declaring backend work done: `go vet` + `go test` via `backend-test`.
- Before declaring frontend work done: lint + test + build via `web-test`.
- CI (`.github/workflows/ci.yml`) runs the same container commands — keep it green.
- LLM calls go through OpenRouter (Phase 3); verify model IDs/pricing against
  current docs, not memory.
