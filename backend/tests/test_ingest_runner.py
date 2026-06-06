import json
from pathlib import Path

from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import News, NewsStock, Stock
from app.services.ingest.alphavantage import parse_alphavantage
from app.services.ingest.finnhub_news import parse_finnhub
from app.services.ingest.newsapi import parse_newsapi
from app.services.ingest.rss import parse_rss
from app.services.ingest.runner import ingest_articles
from app.services.symbol_loader import load_symbols

FIX = Path(__file__).parent / "fixtures"
SYMBOLS = json.loads((FIX / "finnhub_symbols.json").read_text())


def _all_articles():
    arts = []
    arts += parse_finnhub(json.loads((FIX / "finnhub_news.json").read_text()))
    arts += parse_newsapi(json.loads((FIX / "newsapi.json").read_text()))
    arts += parse_alphavantage(json.loads((FIX / "alphavantage.json").read_text()))
    arts += parse_rss((FIX / "sample.rss").read_text())
    return arts


async def test_end_to_end_ingest_and_match(session: AsyncSession):
    await load_symbols(session, SYMBOLS)
    articles = _all_articles()  # 1 + 1 + 1 + 2 = 5 unique

    result = await ingest_articles(session, articles)
    assert result["received"] == 5
    assert result["new"] == 5

    assert await session.scalar(select(func.count()).select_from(News)) == 5

    # the Finnhub "Apple Inc." article auto-links to AAPL
    aapl = await session.scalar(select(Stock).where(Stock.symbol == "AAPL"))
    aapl_links = await session.scalar(
        select(func.count()).select_from(NewsStock).where(
            NewsStock.stock_id == aapl.id, NewsStock.status == "linked"
        )
    )
    assert aapl_links >= 1


async def test_reingest_is_idempotent(session: AsyncSession):
    await load_symbols(session, SYMBOLS)
    articles = _all_articles()
    await ingest_articles(session, articles)
    result2 = await ingest_articles(session, articles)  # same batch again

    assert result2["new"] == 0  # nothing new
    assert await session.scalar(select(func.count()).select_from(News)) == 5
