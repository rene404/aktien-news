import json
import uuid
from pathlib import Path

from httpx import AsyncClient
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import News, NewsStock, Stock
from app.services.symbol_loader import load_symbols

SYMBOLS = json.loads(
    (Path(__file__).parent / "fixtures" / "finnhub_symbols.json").read_text()
)


async def _seed_with_news(session: AsyncSession):
    await load_symbols(session, SYMBOLS)
    aapl = await session.scalar(select(Stock).where(Stock.symbol == "AAPL"))

    linked = News(
        title="Apple linked story", url="http://t/linked", source_type="rss"
    )
    pending = News(
        title="Apple pending story", url="http://t/pending", source_type="rss"
    )
    session.add_all([linked, pending])
    await session.flush()
    session.add_all([
        NewsStock(news_id=linked.id, stock_id=aapl.id, confidence=0.9, status="linked"),
        NewsStock(news_id=pending.id, stock_id=aapl.id, confidence=0.5, status="pending"),
    ])
    await session.commit()
    return aapl, linked, pending


async def test_stock_news_returns_only_linked(
    client: AsyncClient, session: AsyncSession
):
    aapl, linked, pending = await _seed_with_news(session)
    resp = await client.get(f"/stocks/{aapl.id}/news")
    assert resp.status_code == 200
    body = resp.json()
    ids = {item["id"] for item in body["items"]}
    assert str(linked.id) in ids
    assert str(pending.id) not in ids  # pending excluded
    assert body["total"] == 1


async def test_stock_news_unknown_404(client: AsyncClient, session: AsyncSession):
    resp = await client.get(f"/stocks/{uuid.uuid4()}/news")
    assert resp.status_code == 404


async def test_news_detail(client: AsyncClient, session: AsyncSession):
    aapl, linked, _ = await _seed_with_news(session)
    resp = await client.get(f"/news/{linked.id}")
    assert resp.status_code == 200
    body = resp.json()
    assert body["title"] == "Apple linked story"
    assert any(s["symbol"] == "AAPL" for s in body["stocks"])
    assert body["summary"] is None  # not yet summarized


async def test_news_detail_includes_summary(client: AsyncClient, session: AsyncSession):
    aapl, linked, _ = await _seed_with_news(session)
    linked.summary = "A concise AI summary."
    linked.summary_status = "done"
    await session.commit()

    resp = await client.get(f"/news/{linked.id}")
    assert resp.status_code == 200
    assert resp.json()["summary"] == "A concise AI summary."


async def test_news_detail_unknown_404(client: AsyncClient, session: AsyncSession):
    resp = await client.get(f"/news/{uuid.uuid4()}")
    assert resp.status_code == 404
