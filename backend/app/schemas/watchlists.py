import uuid

from pydantic import BaseModel


class WatchlistStockItem(BaseModel):
    stock_id: uuid.UUID
    symbol: str
    company_name: str


class WatchlistOut(BaseModel):
    id: uuid.UUID
    name: str
    stocks: list[WatchlistStockItem]


class AddStockRequest(BaseModel):
    stock_id: uuid.UUID
