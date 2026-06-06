import json
import uuid
from pathlib import Path

from httpx import AsyncClient
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import Stock
from app.services.symbol_loader import load_symbols

SYMBOLS = json.loads(
    (Path(__file__).parent / "fixtures" / "finnhub_symbols.json").read_text()
)


async def _token(client: AsyncClient, email: str) -> str:
    await client.post("/auth/register", json={"email": email, "password": "secret123"})
    login = await client.post(
        "/auth/login", json={"email": email, "password": "secret123"}
    )
    return login.json()["access_token"]


def _auth(token: str) -> dict:
    return {"Authorization": f"Bearer {token}"}


async def test_requires_auth(client: AsyncClient):
    assert (await client.get("/watchlists")).status_code == 401


async def test_default_watchlist_autocreated(client: AsyncClient, session: AsyncSession):
    token = await _token(client, "a@b.com")
    resp = await client.get("/watchlists", headers=_auth(token))
    assert resp.status_code == 200
    body = resp.json()
    assert len(body) == 1
    assert body[0]["stocks"] == []


async def test_add_and_remove_stock(client: AsyncClient, session: AsyncSession):
    await load_symbols(session, SYMBOLS)
    aapl = await session.scalar(select(Stock).where(Stock.symbol == "AAPL"))
    token = await _token(client, "a@b.com")
    wl_id = (await client.get("/watchlists", headers=_auth(token))).json()[0]["id"]

    add = await client.post(
        f"/watchlists/{wl_id}/stocks",
        json={"stock_id": str(aapl.id)},
        headers=_auth(token),
    )
    assert add.status_code == 204
    body = (await client.get("/watchlists", headers=_auth(token))).json()
    assert any(s["symbol"] == "AAPL" for s in body[0]["stocks"])

    rm = await client.delete(
        f"/watchlists/{wl_id}/stocks/{aapl.id}", headers=_auth(token)
    )
    assert rm.status_code == 204
    body = (await client.get("/watchlists", headers=_auth(token))).json()
    assert body[0]["stocks"] == []


async def test_cannot_modify_others_watchlist(
    client: AsyncClient, session: AsyncSession
):
    await load_symbols(session, SYMBOLS)
    aapl = await session.scalar(select(Stock).where(Stock.symbol == "AAPL"))
    owner = await _token(client, "owner@b.com")
    wl_id = (await client.get("/watchlists", headers=_auth(owner))).json()[0]["id"]

    attacker = await _token(client, "attacker@b.com")
    resp = await client.post(
        f"/watchlists/{wl_id}/stocks",
        json={"stock_id": str(aapl.id)},
        headers=_auth(attacker),
    )
    assert resp.status_code == 403


async def test_add_to_unknown_watchlist_404(client: AsyncClient):
    token = await _token(client, "a@b.com")
    resp = await client.post(
        f"/watchlists/{uuid.uuid4()}/stocks",
        json={"stock_id": str(uuid.uuid4())},
        headers=_auth(token),
    )
    assert resp.status_code == 404
