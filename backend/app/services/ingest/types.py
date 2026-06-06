import uuid
from dataclasses import dataclass
from datetime import datetime


@dataclass(frozen=True)
class RawArticle:
    """Source-agnostic normalized article produced by every adapter."""

    url: str
    title: str
    source_type: str  # 'finnhub' | 'newsapi' | 'alphavantage' | 'rss'
    description: str | None = None
    published_at: datetime | None = None
    source_id: uuid.UUID | None = None  # feeds.id for RSS, else None
