from sqlalchemy import case, func, or_, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import Company, Stock, StockAlias
from app.services.matching.normalize import normalize_text


def _like_escape(value: str) -> str:
    """Escape LIKE/ILIKE wildcards so user input can't match everything."""
    return (
        value.replace("\\", "\\\\").replace("%", "\\%").replace("_", "\\_")
    )


async def search_stocks(
    session: AsyncSession, q: str, limit: int = 20, offset: int = 0
) -> list[dict]:
    """Search stocks by symbol prefix, company name, or alias.

    Symbol/prefix matches rank ahead of looser name/alias matches.
    """
    q = q.strip()
    if not q:
        return []
    qnorm = normalize_text(q)
    # normalize_text already strips LIKE metacharacters from qnorm; q-based
    # patterns must escape them so a query like "%" can't match every row.
    qlike = _like_escape(q)

    rank = case(
        (func.lower(Stock.symbol) == q.lower(), 0),
        (Stock.symbol.ilike(f"{qlike}%", escape="\\"), 1),
        else_=2,
    )
    conditions = [
        Stock.symbol.ilike(f"{qlike}%", escape="\\"),
        Company.name.ilike(f"%{qlike}%", escape="\\"),
    ]
    # Only match aliases when the normalized query has content — an empty qnorm
    # (e.g. q="%") would otherwise turn into ilike("%%") and match every row.
    if qnorm:
        conditions.append(StockAlias.alias_norm.ilike(f"%{qnorm}%"))
    stmt = (
        select(
            Stock.id, Stock.symbol, Stock.exchange, Company.name, func.min(rank)
        )
        .join(Company, Stock.company_id == Company.id)
        .outerjoin(StockAlias, StockAlias.stock_id == Stock.id)
        .where(or_(*conditions))
        .group_by(Stock.id, Stock.symbol, Stock.exchange, Company.name)
        .order_by(func.min(rank), Stock.symbol)
        .limit(limit)
        .offset(offset)
    )
    rows = await session.execute(stmt)
    return [
        {
            "stock_id": r[0],
            "symbol": r[1],
            "exchange": r[2],
            "company_name": r[3],
        }
        for r in rows
    ]
