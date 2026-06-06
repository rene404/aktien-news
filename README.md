# Aktien News — Stock News Aggregator

Search a stock symbol or company name and get a feed of news automatically
ingested from multiple sources and linked to that company.

**Stack:** FastAPI · async SQLAlchemy 2.0 · PostgreSQL · React (Vite + TS)
**Scope (Phase 1):** US exchanges (NASDAQ + NYSE)

## Architecture

```
React SPA (Vite)  ──REST──▶  FastAPI  ──▶  PostgreSQL
                                │
                                ├─ ingestion workers (APScheduler)
                                │    Finnhub · NewsAPI · Alpha Vantage · RSS
                                └─ symbol matching (auto-link / review queue)
```

News articles are deduped by URL and matched to stocks by a confidence-based
engine: `$TICKER`, an uppercase ticker, or a full company name auto-link;
ambiguous bare names (e.g. "apple") go to an admin **review queue**.

## Prerequisites

- Docker (for PostgreSQL) · Python 3.12+ with [uv](https://docs.astral.sh/uv/) · Node 20+

## Backend

```bash
# 1. Start PostgreSQL (host port 5433)
docker compose up -d db
docker compose exec -T db psql -U postgres -c "CREATE DATABASE aktien_news_test;"

# 2. Configure + install
cd backend
cp .env.example .env        # fill in FINNHUB_API_KEY etc. for live ingestion
uv sync

# 3. Migrate + run
uv run alembic upgrade head
uv run uvicorn app.main:app --reload     # http://localhost:8000

# Create an admin
ADMIN_EMAIL=you@example.com ADMIN_PASSWORD=secret123 uv run python -m app.scripts.create_admin

# Tests
uv run pytest -q
```

API keys are optional for development — the app boots without them; ingestion
jobs that need a key are skipped. Tests never call live APIs (fixtures only).

## Frontend

```bash
cd frontend
npm install
npm run dev                  # http://localhost:5173 (proxies to :8000)
```

## API surface (Phase 1)

| Area | Endpoints |
|---|---|
| Auth | `POST /auth/register` · `/auth/login` · `/auth/refresh` · `GET /auth/me` |
| Search | `GET /search?q=` |
| News | `GET /stocks/{id}/news` · `GET /news/{id}` |
| Watchlists | `GET /watchlists` · `POST/DELETE /watchlists/{id}/stocks` |
| Admin | `GET/POST/PATCH/DELETE /admin/feeds` · `GET /admin/review` · `POST /admin/review/{id}` |

## Roadmap (Phase 2 — schema-ready, deferred)

AI summaries (Claude API) · price correlation · sentiment.
