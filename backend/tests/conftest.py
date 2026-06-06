import asyncio

import pytest
import pytest_asyncio
from httpx import ASGITransport, AsyncClient
from sqlalchemy import text
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine
from sqlalchemy.pool import NullPool

import app.models  # noqa: F401  (register tables)
from app.core.config import settings
from app.core.db import Base, get_session
from app.main import app

# NullPool: never reuse a connection across event loops (pytest-asyncio uses a
# fresh loop per test), which avoids asyncpg "attached to a different loop" errors.
test_engine = create_async_engine(settings.test_database_url, poolclass=NullPool)
TestSession = async_sessionmaker(
    test_engine, class_=AsyncSession, expire_on_commit=False
)


@pytest.fixture(scope="session", autouse=True)
def _setup_schema():
    async def setup() -> None:
        async with test_engine.begin() as conn:
            await conn.execute(text("CREATE EXTENSION IF NOT EXISTS citext"))
            await conn.execute(text("CREATE EXTENSION IF NOT EXISTS pg_trgm"))
            await conn.run_sync(Base.metadata.drop_all)
            await conn.run_sync(Base.metadata.create_all)

    asyncio.run(setup())
    yield


@pytest_asyncio.fixture(autouse=True)
async def _clean_tables():
    yield
    async with test_engine.begin() as conn:
        for table in reversed(Base.metadata.sorted_tables):
            await conn.execute(text(f'TRUNCATE TABLE "{table.name}" CASCADE'))


@pytest_asyncio.fixture
async def session() -> AsyncSession:
    async with TestSession() as s:
        yield s


@pytest_asyncio.fixture
async def client(session: AsyncSession) -> AsyncClient:
    async def _override():
        yield session

    app.dependency_overrides[get_session] = _override
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as c:
        yield c
    app.dependency_overrides.clear()
