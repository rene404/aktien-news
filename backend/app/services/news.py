import uuid

from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import Company, News, NewsStock, Stock


async def get_stock_news(
    session: AsyncSession, stock_id: uuid.UUID, limit: int, offset: int
) -> tuple[list[News], int] | None:
    """Linked news for a stock, newest first. Returns None if stock unknown."""
    stock = await session.get(Stock, stock_id)
    if stock is None:
        return None

    base = (
        select(News)
        .join(NewsStock, NewsStock.news_id == News.id)
        .where(NewsStock.stock_id == stock_id, NewsStock.status == "linked")
    )
    total = await session.scalar(
        select(func.count()).select_from(base.subquery())
    )
    rows = await session.scalars(
        base.order_by(News.published_at.desc().nullslast())
        .limit(limit)
        .offset(offset)
    )
    return list(rows), int(total or 0)


async def get_news_detail(session: AsyncSession, news_id: uuid.UUID):
    news = await session.get(News, news_id)
    if news is None:
        return None
    refs = await session.execute(
        select(Stock.symbol, Company.name)
        .join(NewsStock, NewsStock.stock_id == Stock.id)
        .join(Company, Stock.company_id == Company.id)
        .where(NewsStock.news_id == news_id, NewsStock.status == "linked")
    )
    stocks = [{"symbol": s, "company_name": n} for s, n in refs]
    return news, stocks
