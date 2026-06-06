"""Role gating via deps, exercised through a temporary protected route."""
from fastapi import Depends
from httpx import AsyncClient
from sqlalchemy.ext.asyncio import AsyncSession

from app.api.deps import require_admin
from app.main import app
from app.services import users


@app.get("/_test/admin-only")
async def _admin_only(admin=Depends(require_admin)):
    return {"ok": True}


async def _make_token(
    client: AsyncClient, session: AsyncSession, email: str, admin: bool = False
) -> str:
    await client.post("/auth/register", json={"email": email, "password": "secret123"})
    if admin:
        # mutate via the SAME session the client uses, so the cached ORM
        # object reflects the new role on the subsequent login read.
        user = await users.get_by_email(session, email)
        user.role = "admin"
        await session.commit()
    login = await client.post(
        "/auth/login", json={"email": email, "password": "secret123"}
    )
    return login.json()["access_token"]


async def test_missing_token_401(client: AsyncClient):
    assert (await client.get("/_test/admin-only")).status_code == 401


async def test_invalid_token_401(client: AsyncClient):
    resp = await client.get(
        "/_test/admin-only", headers={"Authorization": "Bearer garbage"}
    )
    assert resp.status_code == 401


async def test_non_admin_403(client: AsyncClient, session: AsyncSession):
    token = await _make_token(client, session, "user@b.com", admin=False)
    resp = await client.get(
        "/_test/admin-only", headers={"Authorization": f"Bearer {token}"}
    )
    assert resp.status_code == 403


async def test_admin_allowed(client: AsyncClient, session: AsyncSession):
    token = await _make_token(client, session, "admin@b.com", admin=True)
    resp = await client.get(
        "/_test/admin-only", headers={"Authorization": f"Bearer {token}"}
    )
    assert resp.status_code == 200
