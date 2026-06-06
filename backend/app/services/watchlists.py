import uuid

from fastapi import HTTPException, status
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import Company, Stock, Watchlist, WatchlistStock


async def _owned_watchlist(
    session: AsyncSession, watchlist_id: uuid.UUID, user_id: uuid.UUID
) -> Watchlist:
    wl = await session.get(Watchlist, watchlist_id)
    if wl is None:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "Watchlist not found")
    if wl.user_id != user_id:
        raise HTTPException(status.HTTP_403_FORBIDDEN, "Not your watchlist")
    return wl


async def get_or_create_default(
    session: AsyncSession, user_id: uuid.UUID
) -> Watchlist:
    wl = await session.scalar(
        select(Watchlist).where(Watchlist.user_id == user_id).limit(1)
    )
    if wl is None:
        wl = Watchlist(user_id=user_id, name="My Watchlist")
        session.add(wl)
        await session.flush()
    return wl


async def list_with_stocks(session: AsyncSession, user_id: uuid.UUID) -> list[dict]:
    # Read-only: the default watchlist is created at registration, so a GET has
    # no side effects.
    watchlists = (
        await session.scalars(
            select(Watchlist).where(Watchlist.user_id == user_id).order_by(
                Watchlist.name
            )
        )
    ).all()
    out = []
    for wl in watchlists:
        rows = await session.execute(
            select(Stock.id, Stock.symbol, Company.name)
            .join(WatchlistStock, WatchlistStock.stock_id == Stock.id)
            .join(Company, Stock.company_id == Company.id)
            .where(WatchlistStock.watchlist_id == wl.id)
            .order_by(Stock.symbol)
        )
        out.append(
            {
                "id": wl.id,
                "name": wl.name,
                "stocks": [
                    {"stock_id": sid, "symbol": sym, "company_name": name}
                    for sid, sym, name in rows
                ],
            }
        )
    return out


async def add_stock(
    session: AsyncSession,
    user_id: uuid.UUID,
    watchlist_id: uuid.UUID,
    stock_id: uuid.UUID,
) -> None:
    await _owned_watchlist(session, watchlist_id, user_id)
    if await session.get(Stock, stock_id) is None:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "Stock not found")
    exists = await session.get(WatchlistStock, (watchlist_id, stock_id))
    if exists is None:
        session.add(
            WatchlistStock(watchlist_id=watchlist_id, stock_id=stock_id)
        )
    await session.commit()


async def remove_stock(
    session: AsyncSession,
    user_id: uuid.UUID,
    watchlist_id: uuid.UUID,
    stock_id: uuid.UUID,
) -> None:
    await _owned_watchlist(session, watchlist_id, user_id)
    await session.execute(
        WatchlistStock.__table__.delete().where(
            WatchlistStock.watchlist_id == watchlist_id,
            WatchlistStock.stock_id == stock_id,
        )
    )
    await session.commit()
