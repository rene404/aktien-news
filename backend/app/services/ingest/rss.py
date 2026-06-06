import uuid
from datetime import datetime, timezone
from time import mktime

import feedparser

from app.services.ingest.types import RawArticle


def parse_rss(feed_content: str, source_id: uuid.UUID | None = None) -> list[RawArticle]:
    """Parse RSS/Atom feed content into normalized articles.

    Accepts raw feed text (so it can be tested with a fixture file); the live
    fetch path passes the HTTP response body.
    """
    parsed = feedparser.parse(feed_content)
    articles: list[RawArticle] = []
    for entry in parsed.entries:
        url = (entry.get("link") or "").strip()
        title = (entry.get("title") or "").strip()
        if not url or not title:
            continue
        published = None
        tm = entry.get("published_parsed") or entry.get("updated_parsed")
        if tm:
            published = datetime.fromtimestamp(mktime(tm), tz=timezone.utc)
        articles.append(
            RawArticle(
                url=url,
                title=title,
                description=(entry.get("summary") or None),
                source_type="rss",
                source_id=source_id,
                published_at=published,
            )
        )
    return articles
