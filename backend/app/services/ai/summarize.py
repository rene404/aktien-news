"""Background summarization core.

`summarize_pending` is the testable unit: it takes an injected `summarizer`
callable so tests never touch the live API. Per-article failures are isolated
and recorded as 'failed' so one bad article cannot abort the batch.
"""
import logging
from collections.abc import Awaitable, Callable

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import News

logger = logging.getLogger(__name__)

Summarizer = Callable[[str, str | None, str | None], Awaitable[str]]


async def summarize_pending(
    session: AsyncSession, summarizer: Summarizer, limit: int
) -> dict[str, int]:
    """Summarize up to `limit` articles with summary_status == 'pending'.

    Idempotent: 'done' rows are never reprocessed. Each article that raises is
    marked 'failed' (logged) and the batch continues. Returns counts
    {processed, done, failed}.
    """
    rows = (
        await session.scalars(
            select(News)
            .where(News.summary_status == "pending")
            .order_by(News.fetched_at)
            .limit(limit)
        )
    ).all()

    done = 0
    failed = 0
    for news in rows:
        try:
            news.summary = await summarizer(news.title, news.description, news.url)
            news.summary_status = "done"
            done += 1
        except Exception as exc:  # noqa: BLE001 — one bad article must not abort the batch
            logger.warning("summarize failed for news %s: %s", news.id, exc)
            news.summary_status = "failed"
            failed += 1
    await session.commit()
    return {"processed": len(rows), "done": done, "failed": failed}
