"""Admins are created outside /auth/register, so the script must itself
guarantee the default watchlist (regression for the Phase 1 fix that moved
default-watchlist creation into the register endpoint)."""
from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import Watchlist
from app.scripts.create_admin import ensure_admin
from app.services import users


async def _watchlist_count(session: AsyncSession, user_id) -> int:
    return await session.scalar(
        select(func.count()).select_from(Watchlist).where(Watchlist.user_id == user_id)
    )


async def test_ensure_admin_creates_user_and_default_watchlist(session: AsyncSession):
    user, created = await ensure_admin(session, "admin@b.com", "secret123")
    await session.commit()

    assert created is True
    assert user.role == "admin"
    assert await _watchlist_count(session, user.id) == 1


async def test_ensure_admin_is_idempotent(session: AsyncSession):
    await ensure_admin(session, "admin@b.com", "secret123")
    user, created = await ensure_admin(session, "admin@b.com", "newsecret456")
    await session.commit()

    assert created is False  # promoted/updated, not created
    assert user.role == "admin"
    assert await _watchlist_count(session, user.id) == 1  # no duplicate


async def test_ensure_admin_backfills_watchlist_when_promoting(session: AsyncSession):
    # A plain user created outside register has no default watchlist...
    existing = await users.create_user(session, "promote@b.com", "secret123")
    await session.commit()
    assert await _watchlist_count(session, existing.id) == 0

    # ...promoting them to admin must provision one.
    user, created = await ensure_admin(session, "promote@b.com", "secret123")
    await session.commit()

    assert created is False
    assert user.role == "admin"
    assert await _watchlist_count(session, user.id) == 1
