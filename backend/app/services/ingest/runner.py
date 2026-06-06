"""Orchestrates ingestion: normalize -> dedupe-upsert -> match new articles.

`ingest_articles` is the testable core (works on a list of RawArticle). The
`run_*` helpers wire in the live HTTP/RSS fetches and are driven by the
scheduler; they are not exercised in CI.
"""
import logging
from datetime import datetime, timezone

import httpx
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import Feed
from app.services import finnhub
from app.services.ingest.finnhub_news import parse_finnhub
from app.services.ingest.rss import parse_rss
from app.services.ingest.store import upsert_article
from app.services.ingest.types import RawArticle
from app.services.matching.engine import build_index, match_text, persist_matches

logger = logging.getLogger(__name__)


async def ingest_articles(
    session: AsyncSession, articles: list[RawArticle]
) -> dict[str, int]:
    """Upsert articles (dedupe by URL) and match only the newly-created ones.

    Returns counts: received, new, links (news_stocks rows touched).
    """
    index = await build_index(session)  # built once per batch
    new_count = 0
    link_count = 0
    for raw in articles:
        news, created = await upsert_article(session, raw)
        if not created:
            continue
        new_count += 1
        matches = match_text(news.title, news.description, news.url, index)
        link_count += await persist_matches(session, news.id, matches)
    await session.commit()
    return {"received": len(articles), "new": new_count, "links": link_count}


async def run_rss_feeds(session: AsyncSession) -> dict[str, int]:
    """Fetch every active RSS feed, ingest, and stamp last_fetched_at."""
    feeds = (await session.scalars(select(Feed).where(Feed.active.is_(True)))).all()
    all_articles: list[RawArticle] = []
    async with httpx.AsyncClient(timeout=30, follow_redirects=True) as http:
        for feed in feeds:
            try:
                resp = await http.get(feed.url)
                resp.raise_for_status()
                all_articles.extend(parse_rss(resp.text, source_id=feed.id))
                feed.last_fetched_at = datetime.now(timezone.utc)
            except Exception as exc:  # noqa: BLE001 — one bad feed must not abort others
                logger.warning("RSS fetch failed for %s: %s", feed.url, exc)
    return await ingest_articles(session, all_articles)


async def run_finnhub(session: AsyncSession, symbol: str, frm: str, to: str) -> dict:
    payload = await finnhub.fetch_company_news(symbol, frm, to)
    return await ingest_articles(session, parse_finnhub(payload))
