"""Guard against model<->migration drift.

The rest of the suite builds its schema with ``Base.metadata.create_all``, so a
migration that falls out of sync with the models would go unnoticed. This test
runs the real Alembic migrations against a throwaway database and checks that:

  1. ``upgrade head`` produces the same tables/columns as ``Base.metadata``;
  2. the migration round-trips (upgrade -> downgrade -> upgrade) cleanly.

It is a plain sync test on purpose: Alembic's env.py calls ``asyncio.run``,
which would explode inside pytest-asyncio's running event loop.
"""
import asyncio
from pathlib import Path

from alembic import command
from alembic.config import Config
from sqlalchemy import MetaData, text
from sqlalchemy.engine import make_url
from sqlalchemy.ext.asyncio import create_async_engine

import app.core.config as config_module
import app.models  # noqa: F401  (register tables on Base.metadata)
from app.core.config import settings
from app.core.db import Base

THROWAWAY_DB = "aktien_news_migration_check"
ALEMBIC_INI = str(Path(__file__).resolve().parent.parent / "alembic.ini")


def _url_with_db(db_name: str) -> str:
    return make_url(settings.test_database_url).set(database=db_name).render_as_string(
        hide_password=False
    )


async def _recreate_database() -> None:
    engine = create_async_engine(_url_with_db("postgres"), isolation_level="AUTOCOMMIT")
    async with engine.connect() as conn:
        await conn.execute(text(f'DROP DATABASE IF EXISTS "{THROWAWAY_DB}" WITH (FORCE)'))
        await conn.execute(text(f'CREATE DATABASE "{THROWAWAY_DB}"'))
    await engine.dispose()


async def _drop_database() -> None:
    engine = create_async_engine(_url_with_db("postgres"), isolation_level="AUTOCOMMIT")
    async with engine.connect() as conn:
        await conn.execute(text(f'DROP DATABASE IF EXISTS "{THROWAWAY_DB}" WITH (FORCE)'))
    await engine.dispose()


async def _reflect_schema() -> dict[str, set[str]]:
    """Return {table_name: {column_names}} for the throwaway DB, minus Alembic's
    own bookkeeping table."""
    engine = create_async_engine(_url_with_db(THROWAWAY_DB))

    def _do(sync_conn) -> dict[str, set[str]]:
        md = MetaData()
        md.reflect(bind=sync_conn)
        return {
            name: {c.name for c in table.columns}
            for name, table in md.tables.items()
            if name != "alembic_version"
        }

    async with engine.connect() as conn:
        schema = await conn.run_sync(_do)
    await engine.dispose()
    return schema


def _expected_schema() -> dict[str, set[str]]:
    return {
        table.name: {c.name for c in table.columns}
        for table in Base.metadata.sorted_tables
    }


def _alembic_cfg() -> Config:
    return Config(ALEMBIC_INI)


def test_migration_matches_models_and_round_trips():
    original_url = settings.database_url
    config_module.settings.database_url = _url_with_db(THROWAWAY_DB)
    try:
        asyncio.run(_recreate_database())

        # 1. upgrade head -> schema must match the models exactly
        command.upgrade(_alembic_cfg(), "head")
        assert asyncio.run(_reflect_schema()) == _expected_schema(), (
            "migration is out of sync with the ORM models — regenerate it"
        )

        # 2. downgrade base -> all application tables removed
        command.downgrade(_alembic_cfg(), "base")
        assert asyncio.run(_reflect_schema()) == {}

        # 3. upgrade head again -> migration round-trips cleanly
        command.upgrade(_alembic_cfg(), "head")
        assert asyncio.run(_reflect_schema()) == _expected_schema()
    finally:
        config_module.settings.database_url = original_url
        asyncio.run(_drop_database())
