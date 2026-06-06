import uuid
from datetime import datetime

from pydantic import BaseModel


class NewsItem(BaseModel):
    id: uuid.UUID
    title: str
    url: str
    source_type: str
    published_at: datetime | None


class NewsList(BaseModel):
    items: list[NewsItem]
    total: int
    limit: int
    offset: int


class NewsStockRef(BaseModel):
    symbol: str
    company_name: str


class NewsDetail(BaseModel):
    id: uuid.UUID
    title: str
    url: str
    description: str | None
    source_type: str
    published_at: datetime | None
    stocks: list[NewsStockRef]
