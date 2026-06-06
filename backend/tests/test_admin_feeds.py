from httpx import AsyncClient
from sqlalchemy.ext.asyncio import AsyncSession

from app.services import users


async def _admin_token(client: AsyncClient, session: AsyncSession, email="admin@b.com"):
    await client.post("/auth/register", json={"email": email, "password": "secret123"})
    user = await users.get_by_email(session, email)
    user.role = "admin"
    await session.commit()
    login = await client.post(
        "/auth/login", json={"email": email, "password": "secret123"}
    )
    return login.json()["access_token"]


async def _user_token(client: AsyncClient, email="user@b.com"):
    await client.post("/auth/register", json={"email": email, "password": "secret123"})
    login = await client.post(
        "/auth/login", json={"email": email, "password": "secret123"}
    )
    return login.json()["access_token"]


def _h(t):
    return {"Authorization": f"Bearer {t}"}


async def test_non_admin_forbidden(client: AsyncClient):
    token = await _user_token(client)
    assert (await client.get("/admin/feeds", headers=_h(token))).status_code == 403
    resp = await client.post(
        "/admin/feeds",
        json={"url": "http://x/feed", "name": "X"},
        headers=_h(token),
    )
    assert resp.status_code == 403


async def test_admin_feed_crud(client: AsyncClient, session: AsyncSession):
    token = await _admin_token(client, session)
    created = await client.post(
        "/admin/feeds",
        json={"url": "http://x/feed.rss", "name": "Market Wire"},
        headers=_h(token),
    )
    assert created.status_code == 201
    feed_id = created.json()["id"]

    listing = await client.get("/admin/feeds", headers=_h(token))
    assert any(f["id"] == feed_id for f in listing.json())

    patched = await client.patch(
        f"/admin/feeds/{feed_id}", json={"active": False}, headers=_h(token)
    )
    assert patched.status_code == 200
    assert patched.json()["active"] is False

    deleted = await client.delete(f"/admin/feeds/{feed_id}", headers=_h(token))
    assert deleted.status_code == 204


async def test_duplicate_feed_url_conflict(client: AsyncClient, session: AsyncSession):
    token = await _admin_token(client, session)
    body = {"url": "http://x/dup.rss", "name": "Dup"}
    assert (await client.post("/admin/feeds", json=body, headers=_h(token))).status_code == 201
    second = await client.post("/admin/feeds", json=body, headers=_h(token))
    assert second.status_code == 409
