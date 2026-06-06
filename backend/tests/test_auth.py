from httpx import AsyncClient


async def _register(client: AsyncClient, email="a@b.com", password="secret123"):
    return await client.post(
        "/auth/register", json={"email": email, "password": password}
    )


async def test_register_returns_user(client: AsyncClient):
    resp = await _register(client)
    assert resp.status_code == 201
    body = resp.json()
    assert body["email"] == "a@b.com"
    assert body["role"] == "user"


async def test_duplicate_email_conflicts(client: AsyncClient):
    await _register(client)
    resp = await _register(client)
    assert resp.status_code == 409


async def test_login_and_refresh_flow(client: AsyncClient):
    await _register(client)
    login = await client.post(
        "/auth/login", json={"email": "a@b.com", "password": "secret123"}
    )
    assert login.status_code == 200
    tokens = login.json()
    assert tokens["token_type"] == "bearer"

    refresh = await client.post(
        "/auth/refresh", json={"refresh_token": tokens["refresh_token"]}
    )
    assert refresh.status_code == 200
    assert "access_token" in refresh.json()


async def test_login_bad_credentials(client: AsyncClient):
    await _register(client)
    resp = await client.post(
        "/auth/login", json={"email": "a@b.com", "password": "wrong"}
    )
    assert resp.status_code == 401


async def test_me_requires_token(client: AsyncClient):
    assert (await client.get("/auth/me")).status_code == 401


async def test_me_returns_current_user(client: AsyncClient):
    await _register(client)
    login = await client.post(
        "/auth/login", json={"email": "a@b.com", "password": "secret123"}
    )
    token = login.json()["access_token"]
    resp = await client.get("/auth/me", headers={"Authorization": f"Bearer {token}"})
    assert resp.status_code == 200
    assert resp.json()["email"] == "a@b.com"
