import uuid

from pydantic import BaseModel


class StockResult(BaseModel):
    stock_id: uuid.UUID
    symbol: str
    exchange: str
    company_name: str


class SearchResponse(BaseModel):
    results: list[StockResult]
