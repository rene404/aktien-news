import uuid

from fastapi import APIRouter, Depends, HTTPException, Query, status
from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.api.deps import require_admin
from app.core.db import get_session
from app.models import Company, News, NewsStock, Stock, User
from app.schemas.admin import ReviewDecision, ReviewList

router = APIRouter(prefix="/admin/review", tags=["admin"])


@router.get("", response_model=ReviewList)
async def list_review_queue(
    limit: int = Query(50, ge=1, le=200),
    offset: int = Query(0, ge=0),
    _: User = Depends(require_admin),
    session: AsyncSession = Depends(get_session),
):
    total = await session.scalar(
        select(func.count())
        .select_from(NewsStock)
        .where(NewsStock.status == "pending")
    )
    rows = await session.execute(
        select(
            NewsStock.id,
            NewsStock.confidence,
            NewsStock.matched_alias,
            News.id,
            News.title,
            News.url,
            Stock.symbol,
            Company.name,
        )
        .join(News, NewsStock.news_id == News.id)
        .join(Stock, NewsStock.stock_id == Stock.id)
        .join(Company, Stock.company_id == Company.id)
        .where(NewsStock.status == "pending")
        .order_by(NewsStock.created_at.desc())
        .limit(limit)
        .offset(offset)
    )
    items = [
        {
            "news_stock_id": r[0],
            "confidence": float(r[1]),
            "matched_alias": r[2],
            "news": {"id": r[3], "title": r[4], "url": r[5]},
            "stock": {"symbol": r[6], "company_name": r[7]},
        }
        for r in rows
    ]
    return {"items": items, "total": int(total or 0), "limit": limit, "offset": offset}


@router.post("/{news_stock_id}", status_code=status.HTTP_204_NO_CONTENT)
async def decide(
    news_stock_id: uuid.UUID,
    body: ReviewDecision,
    _: User = Depends(require_admin),
    session: AsyncSession = Depends(get_session),
):
    link = await session.get(NewsStock, news_stock_id)
    if link is None:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "Review item not found")
    link.status = "linked" if body.decision == "approve" else "rejected"
    await session.commit()
