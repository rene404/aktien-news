"""Create or promote an admin user from ADMIN_EMAIL / ADMIN_PASSWORD env vars.

Usage:
    ADMIN_EMAIL=a@b.com ADMIN_PASSWORD=secret123 python -m app.scripts.create_admin
"""
import asyncio
import sys

from app.core.config import settings
from app.core.db import SessionLocal
from app.core.security import hash_password
from app.services import users


async def main() -> None:
    email = settings.admin_email
    password = settings.admin_password
    if not email or not password:
        print("ADMIN_EMAIL and ADMIN_PASSWORD must be set", file=sys.stderr)
        raise SystemExit(1)

    async with SessionLocal() as session:
        user = await users.get_by_email(session, email)
        if user is None:
            user = await users.create_user(session, email, password, role="admin")
            print(f"admin created: {email}")
        else:
            user.role = "admin"
            user.password_hash = hash_password(password)
            print(f"admin updated: {email}")
        await session.commit()


if __name__ == "__main__":
    asyncio.run(main())
