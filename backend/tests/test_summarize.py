"""summarize_pending core + job_summarize gating. No live API calls."""
from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.config import settings
from app.models import News
from app.services.ai.summarize import summarize_pending


def _news(title: str, status: str = "pending") -> News:
    return News(
        title=title,
        url=f"http://news.test/{title.replace(' ', '_')}",
        source_type="rss",
        summary_status=status,
    )


async def _fake_summarizer(title, description, url) -> str:
    return f"summary of {title}"


async def _count_pending(session: AsyncSession) -> int:
    return await session.scalar(
        select(func.count()).select_from(News).where(News.summary_status == "pending")
    )


async def test_summarizes_pending_and_stores_result(session: AsyncSession):
    n = _news("Apple earnings")
    session.add(n)
    await session.commit()

    result = await summarize_pending(session, _fake_summarizer, limit=10)

    assert result == {"processed": 1, "done": 1, "failed": 0}
    assert n.summary == "summary of Apple earnings"
    assert n.summary_status == "done"


async def test_only_pending_is_processed(session: AsyncSession):
    pending = _news("pending one", status="pending")
    already = _news("already done", status="done")
    already.summary = "preexisting"
    session.add_all([pending, already])
    await session.commit()

    result = await summarize_pending(session, _fake_summarizer, limit=10)

    assert result["processed"] == 1
    assert pending.summary_status == "done"
    assert already.summary == "preexisting"  # untouched


async def test_is_idempotent(session: AsyncSession):
    session.add(_news("once"))
    await session.commit()

    first = await summarize_pending(session, _fake_summarizer, limit=10)
    second = await summarize_pending(session, _fake_summarizer, limit=10)

    assert first["done"] == 1
    assert second == {"processed": 0, "done": 0, "failed": 0}  # nothing left pending


async def test_failure_is_isolated_and_marked(session: AsyncSession):
    good = _news("good article")
    bad = _news("BOOM article")
    session.add_all([good, bad])
    await session.commit()

    async def flaky(title, description, url):
        if "BOOM" in title:
            raise RuntimeError("model error")
        return f"summary of {title}"

    result = await summarize_pending(session, flaky, limit=10)

    assert result == {"processed": 2, "done": 1, "failed": 1}
    assert good.summary_status == "done"
    assert bad.summary_status == "failed"
    assert bad.summary is None


async def test_limit_is_respected(session: AsyncSession):
    session.add_all([_news("a"), _news("b"), _news("c")])
    await session.commit()

    result = await summarize_pending(session, _fake_summarizer, limit=2)

    assert result["processed"] == 2
    assert await _count_pending(session) == 1  # one left for the next cycle


async def test_job_summarize_skips_when_not_configured(monkeypatch):
    """job_summarize must be a no-op (no summarizer call) when key/model absent."""
    from app.workers import scheduler

    monkeypatch.setattr(settings, "openrouter_api_key", "")
    monkeypatch.setattr(settings, "summarize_model", "")

    called = False

    async def _boom(*args, **kwargs):
        nonlocal called
        called = True
        raise AssertionError("summarizer must not be called when unconfigured")

    monkeypatch.setattr(scheduler, "summarize_pending", _boom, raising=False)

    await scheduler.job_summarize()  # should return cleanly
    assert called is False
