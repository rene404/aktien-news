"""Confidence-based symbol matching.

Scans an article's URL + title + description against the symbol universe and
classifies each candidate link as 'linked' (auto) or 'pending' (review queue).

Confidence rules (highest wins per stock):
  $TICKER cashtag                         -> 0.95  (linked)
  TICKER as an uppercase whole word       -> 0.90  (linked)
  full company name incl. suffix phrase   -> 0.90  (linked)
  core company name phrase (>=3 chars)    -> 0.55  (pending -> review queue)

The bare core name is deliberately low-confidence so generic words ("apple")
land in the review queue rather than auto-linking.
"""
import re
from dataclasses import dataclass

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.config import settings
from app.models import Company, News, NewsStock, Stock
from app.services.matching.normalize import (
    company_name_forms,
    contains_phrase,
    normalize_text,
)

CASHTAG = 0.95
SYMBOL_TOKEN = 0.90
FULL_NAME = 0.90
CORE_NAME = 0.55


@dataclass(frozen=True)
class Candidate:
    stock_id: object
    symbol: str
    full_norm: str
    core_norm: str


@dataclass(frozen=True)
class Match:
    stock_id: object
    confidence: float
    status: str  # 'linked' | 'pending'
    matched_alias: str


async def build_index(session: AsyncSession) -> list[Candidate]:
    rows = await session.execute(
        select(Stock.id, Stock.symbol, Company.name).join(
            Company, Stock.company_id == Company.id
        )
    )
    index: list[Candidate] = []
    for stock_id, symbol, name in rows:
        full_norm, core_norm = company_name_forms(name)
        index.append(Candidate(stock_id, symbol, full_norm, core_norm))
    return index


def _classify(confidence: float) -> str | None:
    if confidence >= settings.match_high_threshold:
        return "linked"
    if confidence >= settings.match_min_threshold:
        return "pending"
    return None


def match_text(
    title: str, description: str | None, url: str | None, index: list[Candidate]
) -> list[Match]:
    raw = " ".join(filter(None, [title, description, url]))
    norm = normalize_text(raw)
    matches: list[Match] = []
    for c in index:
        best = 0.0
        alias = ""
        # cashtag — case-insensitive against raw text
        if re.search(rf"\${re.escape(c.symbol)}\b", raw, re.IGNORECASE):
            best, alias = CASHTAG, f"${c.symbol}"
        # uppercase ticker as a standalone word (case-sensitive)
        if len(c.symbol) >= 2 and re.search(rf"\b{re.escape(c.symbol)}\b", raw):
            if SYMBOL_TOKEN > best:
                best, alias = SYMBOL_TOKEN, c.symbol
        # full company name (only meaningful when a suffix exists)
        if c.full_norm != c.core_norm and contains_phrase(norm, c.full_norm):
            if FULL_NAME > best:
                best, alias = FULL_NAME, c.full_norm
        # bare core name — low confidence
        if len(c.core_norm) >= 3 and contains_phrase(norm, c.core_norm):
            if CORE_NAME > best:
                best, alias = CORE_NAME, c.core_norm

        status = _classify(best)
        if status:
            matches.append(Match(c.stock_id, round(best, 3), status, alias))
    return matches


async def match_article(session: AsyncSession, news: News) -> list[Match]:
    index = await build_index(session)
    return match_text(news.title, news.description, news.url, index)


async def persist_matches(
    session: AsyncSession, news_id, matches: list[Match]
) -> int:
    """Upsert matches into news_stocks. Idempotent and admin-decision-safe:
    never resurrects a 'rejected' link, never downgrades a 'linked' one.
    Returns the number of rows inserted or updated.
    """
    changed = 0
    for m in matches:
        existing = await session.scalar(
            select(NewsStock).where(
                NewsStock.news_id == news_id, NewsStock.stock_id == m.stock_id
            )
        )
        if existing is None:
            session.add(
                NewsStock(
                    news_id=news_id,
                    stock_id=m.stock_id,
                    confidence=m.confidence,
                    status=m.status,
                    matched_alias=m.matched_alias,
                )
            )
            changed += 1
        elif existing.status == "rejected":
            continue  # respect admin rejection
        else:
            existing.confidence = m.confidence
            existing.matched_alias = m.matched_alias
            if existing.status == "pending" and m.status == "linked":
                existing.status = "linked"  # upgrade only
            changed += 1
    await session.flush()
    return changed


async def match_and_store(session: AsyncSession, news: News) -> list[Match]:
    matches = await match_article(session, news)
    await persist_matches(session, news.id, matches)
    return matches
