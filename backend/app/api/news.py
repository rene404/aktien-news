import uuid

from fastapi import APIRouter, Depends, HTTPException, Query, status
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.db import get_session
from app.schemas.news import NewsDetail, NewsList
from app.services import news as news_svc

router = APIRouter(tags=["news"])


@router.get("/stocks/{stock_id}/news", response_model=NewsList)
async def stock_news(
    stock_id: uuid.UUID,
    limit: int = Query(20, ge=1, le=100),
    offset: int = Query(0, ge=0),
    session: AsyncSession = Depends(get_session),
):
    result = await news_svc.get_stock_news(session, stock_id, limit, offset)
    if result is None:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "Stock not found")
    items, total = result
    return {"items": items, "total": total, "limit": limit, "offset": offset}


@router.get("/news/{news_id}", response_model=NewsDetail)
async def news_detail(
    news_id: uuid.UUID, session: AsyncSession = Depends(get_session)
):
    result = await news_svc.get_news_detail(session, news_id)
    if result is None:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "News not found")
    news, stocks = result
    return {
        "id": news.id,
        "title": news.title,
        "url": news.url,
        "description": news.description,
        "source_type": news.source_type,
        "published_at": news.published_at,
        "summary": news.summary,
        "stocks": stocks,
    }
