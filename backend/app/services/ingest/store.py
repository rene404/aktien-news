from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import News
from app.services.ingest.types import RawArticle


async def upsert_article(
    session: AsyncSession, raw: RawArticle
) -> tuple[News, bool]:
    """Insert an article keyed by URL. Returns (news, created).

    Dedupe is by URL: re-ingesting the same URL returns the existing row and
    does not create a duplicate.
    """
    if not raw.url or not raw.title:
        raise ValueError("article requires url and title")

    existing = await session.scalar(select(News).where(News.url == raw.url))
    if existing is not None:
        return existing, False

    news = News(
        url=raw.url,
        title=raw.title,
        description=raw.description,
        source_type=raw.source_type,
        source_id=raw.source_id,
        published_at=raw.published_at,
    )
    session.add(news)
    await session.flush()
    return news, True
