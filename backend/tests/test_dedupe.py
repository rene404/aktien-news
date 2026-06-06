from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import News
from app.services.ingest.store import upsert_article
from app.services.ingest.types import RawArticle


def _article(url="http://news.test/a1"):
    return RawArticle(url=url, title="Headline", source_type="rss")


async def test_first_insert_creates(session: AsyncSession):
    news, created = await upsert_article(session, _article())
    assert created is True
    assert news.id is not None


async def test_same_url_does_not_duplicate(session: AsyncSession):
    await upsert_article(session, _article())
    news2, created = await upsert_article(session, _article())
    assert created is False
    total = await session.scalar(select(func.count()).select_from(News))
    assert total == 1


async def test_missing_fields_rejected(session: AsyncSession):
    import pytest

    with pytest.raises(ValueError):
        await upsert_article(session, RawArticle(url="", title="x", source_type="rss"))
