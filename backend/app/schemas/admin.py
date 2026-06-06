import uuid
from datetime import datetime
from typing import Literal

from pydantic import BaseModel, Field


# ---- Feeds ----
class FeedCreate(BaseModel):
    url: str = Field(min_length=1)
    name: str = Field(min_length=1)


class FeedUpdate(BaseModel):
    name: str | None = None
    active: bool | None = None


class FeedOut(BaseModel):
    id: uuid.UUID
    url: str
    name: str
    active: bool
    last_fetched_at: datetime | None


# ---- Review queue ----
class ReviewNewsRef(BaseModel):
    id: uuid.UUID
    title: str
    url: str


class ReviewStockRef(BaseModel):
    symbol: str
    company_name: str


class ReviewItem(BaseModel):
    news_stock_id: uuid.UUID
    confidence: float
    matched_alias: str | None
    news: ReviewNewsRef
    stock: ReviewStockRef


class ReviewList(BaseModel):
    items: list[ReviewItem]
    total: int
    limit: int
    offset: int


class ReviewDecision(BaseModel):
    decision: Literal["approve", "reject"]
