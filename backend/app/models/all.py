import uuid
from datetime import date, datetime

from sqlalchemy import (
    BigInteger,
    Boolean,
    CheckConstraint,
    DateTime,
    Date,
    ForeignKey,
    Index,
    Numeric,
    String,
    Text,
    UniqueConstraint,
    func,
)
from sqlalchemy.dialects.postgresql import CITEXT, UUID
from sqlalchemy.orm import Mapped, mapped_column, relationship

from app.core.db import Base


def _uuid_pk() -> Mapped[uuid.UUID]:
    return mapped_column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)


class User(Base):
    __tablename__ = "users"

    id: Mapped[uuid.UUID] = _uuid_pk()
    email: Mapped[str] = mapped_column(CITEXT, unique=True, nullable=False)
    password_hash: Mapped[str] = mapped_column(Text, nullable=False)
    role: Mapped[str] = mapped_column(String(16), nullable=False, default="user")
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now()
    )

    __table_args__ = (
        CheckConstraint("role in ('user','admin')", name="ck_users_role"),
    )


class Company(Base):
    __tablename__ = "companies"

    id: Mapped[uuid.UUID] = _uuid_pk()
    name: Mapped[str] = mapped_column(Text, nullable=False)
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now()
    )

    stocks: Mapped[list["Stock"]] = relationship(back_populates="company")


class Stock(Base):
    __tablename__ = "stocks"

    id: Mapped[uuid.UUID] = _uuid_pk()
    symbol: Mapped[str] = mapped_column(Text, nullable=False)
    exchange: Mapped[str] = mapped_column(Text, nullable=False)
    company_id: Mapped[uuid.UUID] = mapped_column(
        ForeignKey("companies.id"), nullable=False
    )

    company: Mapped["Company"] = relationship(back_populates="stocks")
    aliases: Mapped[list["StockAlias"]] = relationship(
        back_populates="stock", cascade="all, delete-orphan"
    )

    __table_args__ = (
        UniqueConstraint("symbol", "exchange", name="uq_stocks_symbol_exchange"),
    )


class StockAlias(Base):
    __tablename__ = "stock_aliases"

    id: Mapped[uuid.UUID] = _uuid_pk()
    stock_id: Mapped[uuid.UUID] = mapped_column(
        ForeignKey("stocks.id", ondelete="CASCADE"), nullable=False
    )
    alias: Mapped[str] = mapped_column(Text, nullable=False)
    alias_norm: Mapped[str] = mapped_column(Text, nullable=False)

    stock: Mapped["Stock"] = relationship(back_populates="aliases")

    __table_args__ = (
        Index("ix_stock_aliases_alias_norm", "alias_norm"),
        Index(
            "ix_stock_aliases_alias_norm_trgm",
            "alias_norm",
            postgresql_using="gin",
            postgresql_ops={"alias_norm": "gin_trgm_ops"},
        ),
    )


class News(Base):
    __tablename__ = "news"

    id: Mapped[uuid.UUID] = _uuid_pk()
    url: Mapped[str] = mapped_column(Text, unique=True, nullable=False)
    title: Mapped[str] = mapped_column(Text, nullable=False)
    description: Mapped[str | None] = mapped_column(Text, nullable=True)
    source_type: Mapped[str] = mapped_column(String(32), nullable=False)
    source_id: Mapped[uuid.UUID | None] = mapped_column(
        UUID(as_uuid=True), nullable=True
    )
    published_at: Mapped[datetime | None] = mapped_column(
        DateTime(timezone=True), nullable=True
    )
    fetched_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now()
    )
    summary: Mapped[str | None] = mapped_column(Text, nullable=True)  # phase 2

    stock_links: Mapped[list["NewsStock"]] = relationship(
        back_populates="news", cascade="all, delete-orphan"
    )

    __table_args__ = (
        CheckConstraint(
            "source_type in ('finnhub','newsapi','alphavantage','rss')",
            name="ck_news_source_type",
        ),
        Index("ix_news_published_at", "published_at"),
    )


class NewsStock(Base):
    __tablename__ = "news_stocks"

    id: Mapped[uuid.UUID] = _uuid_pk()
    news_id: Mapped[uuid.UUID] = mapped_column(
        ForeignKey("news.id", ondelete="CASCADE"), nullable=False
    )
    stock_id: Mapped[uuid.UUID] = mapped_column(
        ForeignKey("stocks.id", ondelete="CASCADE"), nullable=False
    )
    confidence: Mapped[float] = mapped_column(Numeric(4, 3), nullable=False)
    status: Mapped[str] = mapped_column(String(16), nullable=False)
    matched_alias: Mapped[str | None] = mapped_column(Text, nullable=True)
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now()
    )

    news: Mapped["News"] = relationship(back_populates="stock_links")
    stock: Mapped["Stock"] = relationship()

    __table_args__ = (
        UniqueConstraint("news_id", "stock_id", name="uq_news_stocks_pair"),
        CheckConstraint(
            "status in ('linked','pending','rejected')", name="ck_news_stocks_status"
        ),
        Index("ix_news_stocks_status", "status"),
    )


class Feed(Base):
    __tablename__ = "feeds"

    id: Mapped[uuid.UUID] = _uuid_pk()
    url: Mapped[str] = mapped_column(Text, unique=True, nullable=False)
    name: Mapped[str] = mapped_column(Text, nullable=False)
    active: Mapped[bool] = mapped_column(Boolean, nullable=False, default=True)
    created_by: Mapped[uuid.UUID | None] = mapped_column(
        ForeignKey("users.id"), nullable=True
    )
    last_fetched_at: Mapped[datetime | None] = mapped_column(
        DateTime(timezone=True), nullable=True
    )
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now()
    )


class Watchlist(Base):
    __tablename__ = "watchlists"

    id: Mapped[uuid.UUID] = _uuid_pk()
    user_id: Mapped[uuid.UUID] = mapped_column(
        ForeignKey("users.id", ondelete="CASCADE"), nullable=False
    )
    name: Mapped[str] = mapped_column(Text, nullable=False, default="My Watchlist")

    stocks: Mapped[list["WatchlistStock"]] = relationship(
        back_populates="watchlist", cascade="all, delete-orphan"
    )


class WatchlistStock(Base):
    __tablename__ = "watchlist_stocks"

    watchlist_id: Mapped[uuid.UUID] = mapped_column(
        ForeignKey("watchlists.id", ondelete="CASCADE"), primary_key=True
    )
    stock_id: Mapped[uuid.UUID] = mapped_column(
        ForeignKey("stocks.id", ondelete="CASCADE"), primary_key=True
    )

    watchlist: Mapped["Watchlist"] = relationship(back_populates="stocks")


class Price(Base):  # phase 2, unused in phase 1
    __tablename__ = "prices"

    stock_id: Mapped[uuid.UUID] = mapped_column(
        ForeignKey("stocks.id", ondelete="CASCADE"), primary_key=True
    )
    date: Mapped[date] = mapped_column(Date, primary_key=True)
    open: Mapped[float | None] = mapped_column(Numeric, nullable=True)
    high: Mapped[float | None] = mapped_column(Numeric, nullable=True)
    low: Mapped[float | None] = mapped_column(Numeric, nullable=True)
    close: Mapped[float | None] = mapped_column(Numeric, nullable=True)
    volume: Mapped[int | None] = mapped_column(BigInteger, nullable=True)
