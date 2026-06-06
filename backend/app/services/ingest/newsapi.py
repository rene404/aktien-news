from datetime import datetime

from app.services.ingest.types import RawArticle


def parse_newsapi(payload: dict) -> list[RawArticle]:
    """Map a NewsAPI /everything payload to normalized articles."""
    articles: list[RawArticle] = []
    for item in payload.get("articles", []):
        url = (item.get("url") or "").strip()
        title = (item.get("title") or "").strip()
        if not url or not title:
            continue
        published = None
        raw_dt = item.get("publishedAt")
        if raw_dt:
            try:
                published = datetime.fromisoformat(raw_dt.replace("Z", "+00:00"))
            except ValueError:
                published = None
        articles.append(
            RawArticle(
                url=url,
                title=title,
                description=(item.get("description") or None),
                source_type="newsapi",
                published_at=published,
            )
        )
    return articles
