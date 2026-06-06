"""OpenRouter chat-completions adapter.

Provider-neutral by design: the model is `settings.summarize_model`, so the
summarizer can be swapped across providers without code changes. Network calls
live here (like `app/services/finnhub.py`) so the summarize core stays testable.
"""
import httpx

from app.core.config import settings

_SYSTEM_PROMPT = (
    "You are a financial-news summarizer. Summarize the article in 1-2 concise, "
    "factual sentences useful to an investor. Respond with the summary only — no "
    "preamble, no markdown, no headings."
)


async def summarize_article(
    title: str, description: str | None, url: str | None
) -> str:
    """Return a short summary for one article via OpenRouter. Raises on HTTP error."""
    user_content = (
        f"Title: {title}\n\n"
        f"Description: {description or '(none)'}\n\n"
        f"URL: {url or '(none)'}"
    )
    payload = {
        "model": settings.summarize_model,
        "messages": [
            {"role": "system", "content": _SYSTEM_PROMPT},
            {"role": "user", "content": user_content},
        ],
        "max_tokens": 256,
    }
    headers = {
        "Authorization": f"Bearer {settings.openrouter_api_key}",
        "X-Title": "aktien-news",
    }
    async with httpx.AsyncClient(timeout=30) as http:
        resp = await http.post(
            f"{settings.openrouter_base_url}/chat/completions",
            json=payload,
            headers=headers,
        )
        resp.raise_for_status()
        data = resp.json()
    return data["choices"][0]["message"]["content"].strip()
