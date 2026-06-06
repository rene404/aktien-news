import json
from pathlib import Path

from httpx import AsyncClient
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import News, NewsStock, Stock
from app.services import users
from app.services.symbol_loader import load_symbols

SYMBOLS = json.loads(
    (Path(__file__).parent / "fixtures" / "finnhub_symbols.json").read_text()
)


def _h(t):
    return {"Authorization": f"Bearer {t}"}


async def _admin_token(client: AsyncClient, session: AsyncSession, email="admin@b.com"):
    await client.post("/auth/register", json={"email": email, "password": "secret123"})
    user = await users.get_by_email(session, email)
    user.role = "admin"
    await session.commit()
    login = await client.post(
        "/auth/login", json={"email": email, "password": "secret123"}
    )
    return login.json()["access_token"]


async def _pending_link(session: AsyncSession, title, url):
    await load_symbols(session, SYMBOLS)
    aapl = await session.scalar(select(Stock).where(Stock.symbol == "AAPL"))
    news = News(title=title, url=url, source_type="rss")
    session.add(news)
    await session.flush()
    link = NewsStock(
        news_id=news.id, stock_id=aapl.id, confidence=0.55, status="pending"
    )
    session.add(link)
    await session.commit()
    return aapl, news, link


async def test_non_admin_forbidden(client: AsyncClient):
    await client.post("/auth/register", json={"email": "u@b.com", "password": "secret123"})
    login = await client.post(
        "/auth/login", json={"email": "u@b.com", "password": "secret123"}
    )
    token = login.json()["access_token"]
    assert (await client.get("/admin/review", headers=_h(token))).status_code == 403


async def test_queue_lists_pending(client: AsyncClient, session: AsyncSession):
    aapl, news, link = await _pending_link(session, "Maybe Apple", "http://t/p1")
    token = await _admin_token(client, session)
    resp = await client.get("/admin/review", headers=_h(token))
    assert resp.status_code == 200
    body = resp.json()
    assert body["total"] == 1
    assert body["items"][0]["stock"]["symbol"] == "AAPL"


async def test_approve_makes_news_visible(client: AsyncClient, session: AsyncSession):
    aapl, news, link = await _pending_link(session, "Maybe Apple", "http://t/p2")
    token = await _admin_token(client, session)

    # not visible while pending
    before = await client.get(f"/stocks/{aapl.id}/news")
    assert before.json()["total"] == 0

    decide = await client.post(
        f"/admin/review/{link.id}", json={"decision": "approve"}, headers=_h(token)
    )
    assert decide.status_code == 204

    after = await client.get(f"/stocks/{aapl.id}/news")
    assert after.json()["total"] == 1


async def test_reject_keeps_news_hidden(client: AsyncClient, session: AsyncSession):
    aapl, news, link = await _pending_link(session, "Maybe Apple", "http://t/p3")
    token = await _admin_token(client, session)

    decide = await client.post(
        f"/admin/review/{link.id}", json={"decision": "reject"}, headers=_h(token)
    )
    assert decide.status_code == 204

    after = await client.get(f"/stocks/{aapl.id}/news")
    assert after.json()["total"] == 0
    # and it's gone from the pending queue
    queue = await client.get("/admin/review", headers=_h(token))
    assert queue.json()["total"] == 0
