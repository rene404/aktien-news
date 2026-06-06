from datetime import datetime, timezone

from app.services.ingest.types import RawArticle


def parse_finnhub(payload: list[dict]) -> list[RawArticle]:
    """Map a Finnhub company-news payload to normalized articles."""
    articles: list[RawArticle] = []
    for item in payload:
        url = (item.get("url") or "").strip()
        title = (item.get("headline") or "").strip()
        if not url or not title:
            continue
        ts = item.get("datetime")
        published = (
            datetime.fromtimestamp(ts, tz=timezone.utc)
            if isinstance(ts, (int, float)) and ts > 0
            else None
        )
        articles.append(
            RawArticle(
                url=url,
                title=title,
                description=(item.get("summary") or None),
                source_type="finnhub",
                published_at=published,
            )
        )
    return articles
