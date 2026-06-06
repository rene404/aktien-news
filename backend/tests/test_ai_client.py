"""The OpenRouter adapter must build the right request and parse the response.
Never hits the network — httpx.AsyncClient.post is monkeypatched."""
import httpx

from app.core.config import settings
from app.services.ai import client as ai_client


class _FakeResponse:
    def __init__(self, payload: dict):
        self._payload = payload

    def raise_for_status(self) -> None:
        pass

    def json(self) -> dict:
        return self._payload


async def test_summarize_article_builds_request_and_parses_response(monkeypatch):
    monkeypatch.setattr(settings, "openrouter_api_key", "test-key")
    monkeypatch.setattr(settings, "summarize_model", "anthropic/claude-haiku-4.5")
    monkeypatch.setattr(settings, "openrouter_base_url", "https://openrouter.ai/api/v1")

    captured: dict = {}

    async def fake_post(self, url, json=None, headers=None):
        captured["url"] = url
        captured["json"] = json
        captured["headers"] = headers
        return _FakeResponse(
            {"choices": [{"message": {"content": "  Apple beat estimates.  "}}]}
        )

    monkeypatch.setattr(httpx.AsyncClient, "post", fake_post)

    result = await ai_client.summarize_article(
        "Apple beats earnings", "Q4 revenue up 8%", "http://news.test/a"
    )

    # response parsed and stripped
    assert result == "Apple beat estimates."

    # request shape
    assert captured["url"] == "https://openrouter.ai/api/v1/chat/completions"
    assert captured["headers"]["Authorization"] == "Bearer test-key"
    body = captured["json"]
    assert body["model"] == "anthropic/claude-haiku-4.5"
    assert body["max_tokens"] == 256
    assert body["messages"][0]["role"] == "system"
    assert body["messages"][-1]["role"] == "user"
    assert "Apple beats earnings" in body["messages"][-1]["content"]


async def test_summarize_article_propagates_http_errors(monkeypatch):
    monkeypatch.setattr(settings, "openrouter_api_key", "test-key")
    monkeypatch.setattr(settings, "summarize_model", "x/y")

    async def fake_post(self, url, json=None, headers=None):
        request = httpx.Request("POST", url)
        response = httpx.Response(500, request=request)
        raise httpx.HTTPStatusError("boom", request=request, response=response)

    monkeypatch.setattr(httpx.AsyncClient, "post", fake_post)

    try:
        await ai_client.summarize_article("t", "d", "u")
        assert False, "expected HTTPStatusError"
    except httpx.HTTPStatusError:
        pass
