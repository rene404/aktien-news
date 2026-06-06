"""Thin Finnhub HTTP client. Network calls live here so loaders stay testable."""
import httpx

from app.core.config import settings

BASE_URL = "https://finnhub.io/api/v1"


async def fetch_symbols(exchange: str | None = None) -> list[dict]:
    """Fetch the symbol list for an exchange (default from settings, US = NASDAQ+NYSE)."""
    exchange = exchange or settings.exchanges
    params = {"exchange": exchange, "token": settings.finnhub_api_key}
    async with httpx.AsyncClient(timeout=30) as http:
        resp = await http.get(f"{BASE_URL}/stock/symbol", params=params)
        resp.raise_for_status()
        return resp.json()


async def fetch_company_news(symbol: str, frm: str, to: str) -> list[dict]:
    params = {
        "symbol": symbol, "from": frm, "to": to,
        "token": settings.finnhub_api_key,
    }
    async with httpx.AsyncClient(timeout=30) as http:
        resp = await http.get(f"{BASE_URL}/company-news", params=params)
        resp.raise_for_status()
        return resp.json()
