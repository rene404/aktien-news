"""Create or promote an admin user from ADMIN_EMAIL / ADMIN_PASSWORD env vars.

Usage:
    ADMIN_EMAIL=a@b.com ADMIN_PASSWORD=secret123 python -m app.scripts.create_admin
"""
import asyncio
import sys

from sqlalchemy.ext.asyncio import AsyncSession

from app.core.config import settings
from app.core.db import SessionLocal
from app.core.security import hash_password
from app.models import User
from app.services import users, watchlists


async def ensure_admin(
    session: AsyncSession, email: str, password: str
) -> tuple[User, bool]:
    """Create the admin (or promote an existing user) and guarantee a default
    watchlist. Returns (user, created). Caller commits.

    The default watchlist is provisioned here too because users created outside
    the /auth/register endpoint would otherwise have none — and there is no
    endpoint to create one.
    """
    user = await users.get_by_email(session, email)
    created = user is None
    if user is None:
        user = await users.create_user(session, email, password, role="admin")
    else:
        user.role = "admin"
        user.password_hash = hash_password(password)
    await watchlists.get_or_create_default(session, user.id)
    return user, created


async def main() -> None:
    email = settings.admin_email
    password = settings.admin_password
    if not email or not password:
        print("ADMIN_EMAIL and ADMIN_PASSWORD must be set", file=sys.stderr)
        raise SystemExit(1)

    async with SessionLocal() as session:
        _, created = await ensure_admin(session, email, password)
        await session.commit()
        print(f"admin {'created' if created else 'updated'}: {email}")


if __name__ == "__main__":
    asyncio.run(main())
