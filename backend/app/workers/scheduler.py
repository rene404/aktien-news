"""Scheduled ingestion jobs (APScheduler, in-process).

build_scheduler() registers all jobs but does NOT start them or make any
network call — start() is called from the app lifespan. Each job opens its own
session and isolates failures so one bad source never aborts the others.
"""
import logging
from datetime import date, timedelta

from apscheduler.schedulers.asyncio import AsyncIOScheduler

from app.core.config import settings
from app.core.db import SessionLocal
from app.services.ai.client import summarize_article
from app.services.ai.summarize import summarize_pending
from app.services.ingest.runner import run_finnhub, run_rss_feeds
from app.services.symbol_loader import refresh_symbols

logger = logging.getLogger(__name__)


async def job_refresh_symbols() -> None:
    if not settings.finnhub_api_key:
        logger.info("skip symbol refresh: no FINNHUB_API_KEY")
        return
    async with SessionLocal() as session:
        n = await refresh_symbols(session, settings.exchanges)
        logger.info("symbol refresh: %s stocks", n)


async def job_rss() -> None:
    async with SessionLocal() as session:
        result = await run_rss_feeds(session)
        logger.info("rss ingest: %s", result)


async def job_finnhub_news() -> None:
    if not settings.finnhub_api_key:
        logger.info("skip finnhub news: no FINNHUB_API_KEY")
        return
    to = date.today()
    frm = to - timedelta(days=1)
    async with SessionLocal() as session:
        from sqlalchemy import select

        from app.models import Stock

        symbols = (await session.scalars(select(Stock.symbol).limit(50))).all()
        for symbol in symbols:
            try:
                await run_finnhub(session, symbol, frm.isoformat(), to.isoformat())
            except Exception as exc:  # noqa: BLE001
                logger.warning("finnhub news failed for %s: %s", symbol, exc)


async def job_summarize() -> None:
    if not settings.summarize_configured:
        logger.info("skip summarize: OPENROUTER_API_KEY/SUMMARIZE_MODEL not set")
        return
    async with SessionLocal() as session:
        result = await summarize_pending(
            session, summarize_article, settings.summarize_batch_size
        )
        logger.info("summarize: %s", result)


def build_scheduler() -> AsyncIOScheduler:
    scheduler = AsyncIOScheduler(timezone="UTC")
    scheduler.add_job(job_refresh_symbols, "interval", hours=24, id="symbol_refresh")
    scheduler.add_job(job_rss, "interval", minutes=15, id="rss_ingest")
    scheduler.add_job(job_finnhub_news, "interval", minutes=30, id="finnhub_news")
    scheduler.add_job(job_summarize, "interval", minutes=10, id="summarize")
    return scheduler
