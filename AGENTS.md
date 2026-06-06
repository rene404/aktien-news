# AGENTS.md — Aktien News

Authoritative working agreement for agents and contributors. Keep this in sync
when conventions change. The README is the user-facing overview; this file is
the rulebook.

## What this is

Stock news aggregator: search a US stock (NASDAQ + NYSE) by symbol or company
name and get a feed of news auto-ingested from multiple sources and linked to
that company by a confidence-based matching engine.

**Stack:** FastAPI · async SQLAlchemy 2.0 · PostgreSQL 16 · React 19 (Vite + TS)
**Scope:** Phase 1 (foundation, auth, ingestion, matching, public/user/admin APIs, SPA).
Phase 2 (AI summaries via Claude API, price correlation, sentiment) is
schema-ready but deferred — `News.summary` and the `prices` table already exist
but are unused.

## Layout

```
backend/
  app/
    api/        # FastAPI routers — THIN. No business logic here.
    core/       # config, db engine/session, security (jwt + bcrypt)
    models/     # SQLAlchemy ORM models (all.py). Persistence only.
    schemas/    # Pydantic request/response models. Distinct from ORM models.
    services/   # business logic. Routers call these.
      ingest/   # source adapters (rss, finnhub_news, newsapi, alphavantage),
                # dedupe store, runner (testable core + live-fetch wiring)
      matching/ # confidence-based symbol matching + text normalization
    workers/    # APScheduler jobs (in-process)
    scripts/    # one-off ops (create_admin)
  alembic/      # migrations
  tests/        # pytest, fixtures-only (never hits live APIs)
frontend/
  src/api/      # axios client + TS types
  src/auth/     # AuthContext
  src/pages/    # routed pages (incl. admin/)
docker-compose.yml   # postgres:16 on host port 5433
```

## Layering rules (do not violate)

- **Routers are thin.** Parse/validate input, call a service, shape the
  response. No queries or business logic in `app/api/*`.
- **Business logic lives in `app/services/*`.** Services own all DB access for
  their domain.
- **ORM models (`app/models`) never leak into the transport layer.** Endpoints
  return Pydantic schemas (`app/schemas`), not ORM objects, except where a
  `response_model` cleanly serializes a model.
- **Request/response schemas are separate from persistence models.** Don't
  reuse an ORM class as an API contract.
- **Ingest adapters are pure parsers** (`parse_*(payload) -> list[RawArticle]`).
  Live HTTP/RSS fetching is isolated in `finnhub.py` and the `run_*` runner
  helpers so the cores stay unit-testable with fixtures.

## Commands

**Backend** (from `backend/`, uses [uv](https://docs.astral.sh/uv/)):

```bash
docker compose up -d db                 # Postgres on :5433 (run from repo root)
uv sync                                 # install deps
uv run alembic upgrade head             # migrate
uv run uvicorn app.main:app --reload    # serve on :8000
uv run pytest -q                        # tests (needs the test DB, see below)

# Test DB (one-time):
docker compose exec -T db psql -U postgres -c "CREATE DATABASE aktien_news_test;"

# Create/promote an admin:
ADMIN_EMAIL=you@example.com ADMIN_PASSWORD=secret123 uv run python -m app.scripts.create_admin
```

**Frontend** (from `frontend/`, Node 20+):

```bash
npm install
npm run dev      # :5173, proxies API to :8000
npm run build    # tsc -b && vite build
npm run lint     # eslint
```

## Database & migrations

- Async SQLAlchemy 2.0 (`Mapped[...]` / `mapped_column`), asyncpg driver.
- Postgres extensions required: `citext` (case-insensitive email) and `pg_trgm`
  (trigram alias search). Created in the migration and in test setup.
- **Every migration needs both `upgrade` and `downgrade`,** and the downgrade
  must be a safe reverse. Test up/down/up before merging.
- The main suite builds its schema with `Base.metadata.create_all`, but
  `tests/test_migration_drift.py` runs the real migrations against a throwaway
  database and asserts the resulting tables/columns match `Base.metadata` (and
  that the migration round-trips). **When you change a model, regenerate the
  migration** — that test fails if the migration drifts from the models.

## Auth model

- Stateless JWT (HS256). Access token ~30 min, refresh token ~14 days. Both
  carry `sub` (user id) and `role`.
- `get_current_user` / `require_admin` in `app/api/deps.py` gate routes. Use
  these dependencies — don't re-implement auth checks in routers.
- 401 = unauthenticated/invalid token; 403 = authenticated but not admin. Keep
  this distinction.
- Public (no auth): `/search`, `/stocks/{id}/news`, `/news/{id}`, health.
  User: `/auth/*`, `/watchlists/*`. Admin: `/admin/*`.
- `JWT_SECRET` defaults to `change-me-in-production` — must be overridden in any
  non-local environment. Never commit a real secret.

## Matching engine semantics (`app/services/matching`)

Confidence per stock, highest wins:

| Signal | Confidence | Result |
|---|---|---|
| `$TICKER` cashtag | 0.95 | auto-link |
| Uppercase ticker as whole word (≥2 chars) | 0.90 | auto-link |
| Full company name incl. suffix | 0.90 | auto-link |
| Bare core name (≥3 chars) | 0.55 | review queue (`pending`) |

`>= match_high_threshold` (0.85) → `linked`; `>= match_min_threshold` (0.40) →
`pending`; else dropped. `persist_matches` is idempotent and admin-safe: it
never resurrects a `rejected` link and only upgrades `pending → linked`.

## Testing conventions

- pytest + pytest-asyncio (`asyncio_mode = "auto"` — no `@pytest.mark.asyncio`).
- **Tests never call live external APIs.** Use fixtures in `tests/fixtures/`.
- App boots without API keys; key-dependent ingestion jobs are skipped, not
  errors. Keep this true.
- Test behaviour (observable output / status codes), not internals. For bug
  fixes, add a regression test.
- `conftest.py` uses `NullPool` (fresh event loop per test) and truncates tables
  between tests — don't rely on cross-test state.

## Conventions

- Python 3.12+, type hints throughout, `X | None` unions.
- Keep modules small and single-purpose; match the surrounding style.
- Ingestion failures are isolated per-source (one bad feed/symbol must not abort
  the batch) and logged at WARNING — preserve this.
- Frontend: React 19 + react-router 7, axios client in `src/api/client.ts` with
  a request interceptor (bearer token) and a one-shot 401→refresh→replay.

## Workflow

- Don't commit or push unless asked. Branch off `main` for changes.
- Run `uv run pytest -q` (backend) / `npm run lint && npm run build` (frontend)
  before declaring work done.
- There is no CONTEXT.md; this file plus the README are the source of truth.
