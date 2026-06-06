import uuid

from fastapi import APIRouter, Depends, status
from sqlalchemy.ext.asyncio import AsyncSession

from app.api.deps import get_current_user
from app.core.db import get_session
from app.models import User
from app.schemas.watchlists import AddStockRequest, WatchlistOut
from app.services import watchlists as wl_svc

router = APIRouter(prefix="/watchlists", tags=["watchlists"])


@router.get("", response_model=list[WatchlistOut])
async def list_watchlists(
    current: User = Depends(get_current_user),
    session: AsyncSession = Depends(get_session),
):
    return await wl_svc.list_with_stocks(session, current.id)


@router.post("/{watchlist_id}/stocks", status_code=status.HTTP_204_NO_CONTENT)
async def add_stock(
    watchlist_id: uuid.UUID,
    body: AddStockRequest,
    current: User = Depends(get_current_user),
    session: AsyncSession = Depends(get_session),
):
    await wl_svc.add_stock(session, current.id, watchlist_id, body.stock_id)


@router.delete(
    "/{watchlist_id}/stocks/{stock_id}", status_code=status.HTTP_204_NO_CONTENT
)
async def remove_stock(
    watchlist_id: uuid.UUID,
    stock_id: uuid.UUID,
    current: User = Depends(get_current_user),
    session: AsyncSession = Depends(get_session),
):
    await wl_svc.remove_stock(session, current.id, watchlist_id, stock_id)
