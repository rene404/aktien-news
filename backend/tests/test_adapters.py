import json
from pathlib import Path

from app.services.ingest.alphavantage import parse_alphavantage
from app.services.ingest.finnhub_news import parse_finnhub
from app.services.ingest.newsapi import parse_newsapi
from app.services.ingest.rss import parse_rss

FIX = Path(__file__).parent / "fixtures"


def test_finnhub_adapter():
    articles = parse_finnhub(json.loads((FIX / "finnhub_news.json").read_text()))
    assert len(articles) == 1  # empty-headline row skipped
    a = articles[0]
    assert a.source_type == "finnhub"
    assert a.url == "https://example.com/finnhub/apple-iphone"
    assert a.title == "Apple Inc. unveils new iPhone lineup"
    assert a.published_at is not None


def test_newsapi_adapter():
    articles = parse_newsapi(json.loads((FIX / "newsapi.json").read_text()))
    assert len(articles) == 1  # null-title row skipped
    a = articles[0]
    assert a.source_type == "newsapi"
    assert a.title.startswith("Microsoft Corporation")
    assert a.published_at is not None


def test_alphavantage_adapter():
    articles = parse_alphavantage(
        json.loads((FIX / "alphavantage.json").read_text())
    )
    assert len(articles) == 1  # empty-title row skipped
    a = articles[0]
    assert a.source_type == "alphavantage"
    assert a.title.startswith("Tesla Inc")
    assert a.published_at is not None


def test_rss_adapter():
    import uuid

    fid = uuid.uuid4()
    articles = parse_rss((FIX / "sample.rss").read_text(), source_id=fid)
    assert len(articles) == 2
    assert {a.url for a in articles} == {
        "https://example.com/rss/apple-supplier",
        "https://example.com/rss/nvda-high",
    }
    assert all(a.source_type == "rss" for a in articles)
    assert all(a.source_id == fid for a in articles)
    assert all(a.published_at is not None for a in articles)
