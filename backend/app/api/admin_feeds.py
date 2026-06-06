import uuid

from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy import select
from sqlalchemy.exc import IntegrityError
from sqlalchemy.ext.asyncio import AsyncSession

from app.api.deps import require_admin
from app.core.db import get_session
from app.models import Feed, User
from app.schemas.admin import FeedCreate, FeedOut, FeedUpdate

router = APIRouter(prefix="/admin/feeds", tags=["admin"])


@router.get("", response_model=list[FeedOut])
async def list_feeds(
    _: User = Depends(require_admin),
    session: AsyncSession = Depends(get_session),
):
    feeds = await session.scalars(select(Feed).order_by(Feed.created_at.desc()))
    return list(feeds)


@router.post("", response_model=FeedOut, status_code=status.HTTP_201_CREATED)
async def create_feed(
    body: FeedCreate,
    admin: User = Depends(require_admin),
    session: AsyncSession = Depends(get_session),
):
    feed = Feed(url=body.url, name=body.name, created_by=admin.id)
    session.add(feed)
    try:
        await session.commit()
    except IntegrityError:
        await session.rollback()
        raise HTTPException(status.HTTP_409_CONFLICT, "Feed URL already exists")
    await session.refresh(feed)
    return feed


@router.patch("/{feed_id}", response_model=FeedOut)
async def update_feed(
    feed_id: uuid.UUID,
    body: FeedUpdate,
    _: User = Depends(require_admin),
    session: AsyncSession = Depends(get_session),
):
    feed = await session.get(Feed, feed_id)
    if feed is None:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "Feed not found")
    if body.name is not None:
        feed.name = body.name
    if body.active is not None:
        feed.active = body.active
    await session.commit()
    await session.refresh(feed)
    return feed


@router.delete("/{feed_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_feed(
    feed_id: uuid.UUID,
    _: User = Depends(require_admin),
    session: AsyncSession = Depends(get_session),
):
    feed = await session.get(Feed, feed_id)
    if feed is None:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "Feed not found")
    await session.delete(feed)
    await session.commit()
