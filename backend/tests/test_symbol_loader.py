import json
from pathlib import Path

from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import Company, Stock, StockAlias
from app.services.symbol_loader import load_symbols

FIXTURE = json.loads(
    (Path(__file__).parent / "fixtures" / "finnhub_symbols.json").read_text()
)


async def _count(session: AsyncSession, model) -> int:
    return await session.scalar(select(func.count()).select_from(model))


async def test_load_populates_equities_only(session: AsyncSession):
    n = await load_symbols(session, FIXTURE)
    # AAPL, MSFT, TSLA kept; SPY (ETP) and the empty-symbol row dropped
    assert n == 3
    assert await _count(session, Stock) == 3
    assert await _count(session, Company) == 3


async def test_load_creates_expected_aliases(session: AsyncSession):
    await load_symbols(session, FIXTURE)
    aapl = await session.scalar(select(Stock).where(Stock.symbol == "AAPL"))
    aliases = await session.scalars(
        select(StockAlias.alias_norm).where(StockAlias.stock_id == aapl.id)
    )
    norms = set(aliases)
    assert {"aapl", "apple inc", "apple"} <= norms


async def test_load_is_idempotent(session: AsyncSession):
    await load_symbols(session, FIXTURE)
    await load_symbols(session, FIXTURE)
    assert await _count(session, Stock) == 3
    # aliases not duplicated on re-run
    aapl = await session.scalar(select(Stock).where(Stock.symbol == "AAPL"))
    n_alias = await session.scalar(
        select(func.count()).select_from(StockAlias).where(
            StockAlias.stock_id == aapl.id
        )
    )
    assert n_alias == 3
