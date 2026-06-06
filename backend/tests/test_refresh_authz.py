"""Refresh must reflect the current DB state, not stale token claims."""
from httpx import AsyncClient
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.security import create_refresh_token, decode_token
from app.services import users


async def _register(client: AsyncClient, email="a@b.com", password="secret123"):
    return await client.post(
        "/auth/register", json={"email": email, "password": password}
    )


async def test_refresh_for_deleted_user_is_rejected(
    client: AsyncClient, session: AsyncSession
):
    await _register(client)
    login = await client.post(
        "/auth/login", json={"email": "a@b.com", "password": "secret123"}
    )
    refresh_token = login.json()["refresh_token"]

    user = await users.get_by_email(session, "a@b.com")
    await session.delete(user)
    await session.commit()

    resp = await client.post("/auth/refresh", json={"refresh_token": refresh_token})
    assert resp.status_code == 401


async def test_refresh_uses_current_role_not_token_role(
    client: AsyncClient, session: AsyncSession
):
    """A refresh token minted while admin must not keep granting admin after
    the role is revoked — the new access token reflects the DB role."""
    await _register(client)
    user = await users.get_by_email(session, "a@b.com")
    user.role = "admin"
    await session.commit()
    # a refresh token still carrying role=admin
    stale = create_refresh_token(str(user.id), "admin")

    # now demote
    user.role = "user"
    await session.commit()

    resp = await client.post("/auth/refresh", json={"refresh_token": stale})
    assert resp.status_code == 200
    new_access = resp.json()["access_token"]
    assert decode_token(new_access)["role"] == "user"  # not admin


async def test_refresh_rejects_access_token(client: AsyncClient):
    await _register(client)
    login = await client.post(
        "/auth/login", json={"email": "a@b.com", "password": "secret123"}
    )
    access = login.json()["access_token"]
    resp = await client.post("/auth/refresh", json={"refresh_token": access})
    assert resp.status_code == 401  # access token is not a refresh token
