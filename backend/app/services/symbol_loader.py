"""Load the symbol universe (companies, stocks, aliases) into the DB.

`load_symbols` is pure DB work over an in-memory list, so it can be tested with
a fixture without hitting Finnhub. `refresh_symbols` wires in the HTTP fetch.
"""
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import Company, Stock, StockAlias
from app.services import finnhub
from app.services.matching.normalize import company_name_forms, normalize_text

# Finnhub `type` values we keep (drop ETFs, bonds, etc. for phase 1).
EQUITY_TYPES = {"Common Stock", "ADR", ""}


def _alias_set(symbol: str, name: str) -> list[tuple[str, str]]:
    """Return (alias, alias_norm) pairs for a stock: symbol, full name, core name."""
    full_norm, core_norm = company_name_forms(name)
    out: dict[str, str] = {}
    out[normalize_text(symbol)] = symbol
    if full_norm:
        out.setdefault(full_norm, name)
    if core_norm:
        out.setdefault(core_norm, name)
    return [(alias, norm) for norm, alias in out.items() if norm]


async def load_symbols(
    session: AsyncSession, symbols: list[dict], exchange: str = "US"
) -> int:
    """Upsert companies/stocks/aliases from a Finnhub symbol list. Idempotent.

    Returns the number of stocks created or updated.
    """
    count = 0
    for entry in symbols:
        symbol = (entry.get("symbol") or "").strip()
        name = (entry.get("description") or "").strip()
        if not symbol or not name:
            continue
        if entry.get("type") not in EQUITY_TYPES:
            continue

        existing = await session.scalar(
            select(Stock).where(Stock.symbol == symbol, Stock.exchange == exchange)
        )
        if existing is None:
            company = Company(name=name)
            session.add(company)
            await session.flush()
            stock = Stock(symbol=symbol, exchange=exchange, company_id=company.id)
            session.add(stock)
            await session.flush()
        else:
            stock = existing
            # refresh company name if it changed
            company = await session.get(Company, stock.company_id)
            if company and company.name != name:
                company.name = name

        # rebuild aliases for this stock (idempotent)
        await session.execute(
            StockAlias.__table__.delete().where(StockAlias.stock_id == stock.id)
        )
        for alias, alias_norm in _alias_set(symbol, name):
            session.add(
                StockAlias(stock_id=stock.id, alias=alias, alias_norm=alias_norm)
            )
        count += 1

    await session.commit()
    return count


async def refresh_symbols(session: AsyncSession, exchange: str = "US") -> int:
    symbols = await finnhub.fetch_symbols(exchange)
    return await load_symbols(session, symbols, exchange)
