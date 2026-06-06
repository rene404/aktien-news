import json
from pathlib import Path

from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import News, NewsStock, Stock
from app.services.matching.engine import match_and_store
from app.services.symbol_loader import load_symbols

FIXTURE = json.loads(
    (Path(__file__).parent / "fixtures" / "finnhub_symbols.json").read_text()
)


async def _seed_symbols(session: AsyncSession):
    await load_symbols(session, FIXTURE)


async def _make_news(session: AsyncSession, title, description=None, url=None):
    news = News(
        title=title,
        description=description,
        url=url or f"http://news.test/{abs(hash(title))}",
        source_type="rss",
    )
    session.add(news)
    await session.flush()
    return news


async def _link_status(session: AsyncSession, news_id, symbol) -> str | None:
    stock = await session.scalar(select(Stock).where(Stock.symbol == symbol))
    return await session.scalar(
        select(NewsStock.status).where(
            NewsStock.news_id == news_id, NewsStock.stock_id == stock.id
        )
    )


async def test_full_company_name_auto_links(session: AsyncSession):
    await _seed_symbols(session)
    news = await _make_news(session, "Apple Inc. announces a new iPhone")
    await match_and_store(session, news)
    assert await _link_status(session, news.id, "AAPL") == "linked"


async def test_cashtag_auto_links(session: AsyncSession):
    await _seed_symbols(session)
    news = await _make_news(session, "$AAPL surges 5% in premarket trading")
    await match_and_store(session, news)
    assert await _link_status(session, news.id, "AAPL") == "linked"


async def test_uppercase_ticker_auto_links(session: AsyncSession):
    await _seed_symbols(session)
    news = await _make_news(session, "Analysts upgrade MSFT to buy")
    await match_and_store(session, news)
    assert await _link_status(session, news.id, "MSFT") == "linked"


async def test_bare_common_word_goes_to_pending_not_linked(session: AsyncSession):
    """The fruit 'apple' must NOT auto-link AAPL — it lands in the review queue."""
    await _seed_symbols(session)
    news = await _make_news(
        session, "I ate an apple today and it was delicious"
    )
    await match_and_store(session, news)
    status = await _link_status(session, news.id, "AAPL")
    assert status == "pending"  # not 'linked'


async def test_unrelated_text_no_match(session: AsyncSession):
    await _seed_symbols(session)
    news = await _make_news(session, "Local bakery wins award for sourdough bread")
    await match_and_store(session, news)
    assert await _link_status(session, news.id, "AAPL") is None
    assert await _link_status(session, news.id, "MSFT") is None


async def test_rematch_is_idempotent(session: AsyncSession):
    await _seed_symbols(session)
    news = await _make_news(session, "Apple Inc. announces a new iPhone")
    await match_and_store(session, news)
    await match_and_store(session, news)  # re-run
    n = await session.scalar(
        select(func.count()).select_from(NewsStock).where(
            NewsStock.news_id == news.id
        )
    )
    assert n == 1
    assert await _link_status(session, news.id, "AAPL") == "linked"
