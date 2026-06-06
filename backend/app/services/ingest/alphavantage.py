from datetime import datetime, timezone

from app.services.ingest.types import RawArticle


def _parse_av_time(value: str | None) -> datetime | None:
    # Alpha Vantage uses e.g. "20220410T013000"
    if not value:
        return None
    try:
        return datetime.strptime(value, "%Y%m%dT%H%M%S").replace(tzinfo=timezone.utc)
    except ValueError:
        return None


def parse_alphavantage(payload: dict) -> list[RawArticle]:
    """Map an Alpha Vantage NEWS_SENTIMENT payload to normalized articles."""
    articles: list[RawArticle] = []
    for item in payload.get("feed", []):
        url = (item.get("url") or "").strip()
        title = (item.get("title") or "").strip()
        if not url or not title:
            continue
        articles.append(
            RawArticle(
                url=url,
                title=title,
                description=(item.get("summary") or None),
                source_type="alphavantage",
                published_at=_parse_av_time(item.get("time_published")),
            )
        )
    return articles
