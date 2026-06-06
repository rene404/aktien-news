import json
from pathlib import Path

from httpx import AsyncClient
from sqlalchemy.ext.asyncio import AsyncSession

from app.services.symbol_loader import load_symbols

SYMBOLS = json.loads(
    (Path(__file__).parent / "fixtures" / "finnhub_symbols.json").read_text()
)


async def _seed(session: AsyncSession):
    await load_symbols(session, SYMBOLS)


async def test_search_by_symbol(client: AsyncClient, session: AsyncSession):
    await _seed(session)
    resp = await client.get("/search", params={"q": "AAPL"})
    assert resp.status_code == 200
    symbols = [r["symbol"] for r in resp.json()["results"]]
    assert "AAPL" in symbols
    assert symbols[0] == "AAPL"  # exact symbol ranks first


async def test_search_by_company_name(client: AsyncClient, session: AsyncSession):
    await _seed(session)
    resp = await client.get("/search", params={"q": "Apple"})
    symbols = [r["symbol"] for r in resp.json()["results"]]
    assert "AAPL" in symbols


async def test_search_by_alias_prefix(client: AsyncClient, session: AsyncSession):
    await _seed(session)
    resp = await client.get("/search", params={"q": "micro"})
    symbols = [r["symbol"] for r in resp.json()["results"]]
    assert "MSFT" in symbols


async def test_search_no_match_empty(client: AsyncClient, session: AsyncSession):
    await _seed(session)
    resp = await client.get("/search", params={"q": "zzzznotreal"})
    assert resp.json()["results"] == []


async def test_search_wildcard_is_escaped(client: AsyncClient, session: AsyncSession):
    """A bare '%' must not match every stock (LIKE wildcard injection)."""
    await _seed(session)
    resp = await client.get("/search", params={"q": "%"})
    assert resp.status_code == 200
    assert resp.json()["results"] == []
