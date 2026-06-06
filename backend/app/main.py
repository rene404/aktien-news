from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.api import (
    admin_feeds,
    admin_review,
    auth,
    health,
    news,
    search,
    watchlists,
)
from app.core.config import settings


@asynccontextmanager
async def lifespan(app: FastAPI):
    settings.check_production_secrets()
    scheduler = None
    if settings.enable_scheduler:
        from app.workers.scheduler import build_scheduler

        scheduler = build_scheduler()
        scheduler.start()
    try:
        yield
    finally:
        if scheduler is not None:
            scheduler.shutdown(wait=False)


app = FastAPI(title="Aktien News API", lifespan=lifespan)

app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.cors_origins_list,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(health.router)
app.include_router(auth.router)
app.include_router(search.router)
app.include_router(news.router)
app.include_router(watchlists.router)
app.include_router(admin_feeds.router)
app.include_router(admin_review.router)
