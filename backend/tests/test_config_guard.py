"""The app must refuse to boot in production with the default JWT secret."""
import pytest

from app.core.config import Settings


def test_production_with_default_secret_raises():
    s = Settings(environment="production", jwt_secret=Settings.DEFAULT_JWT_SECRET)
    with pytest.raises(RuntimeError):
        s.check_production_secrets()


def test_production_with_real_secret_ok():
    s = Settings(environment="production", jwt_secret="a-real-strong-secret")
    s.check_production_secrets()  # must not raise


def test_development_with_default_secret_ok():
    s = Settings(environment="development", jwt_secret=Settings.DEFAULT_JWT_SECRET)
    s.check_production_secrets()  # default is fine outside production


def test_cors_origins_list_parses_csv():
    s = Settings(cors_origins="http://a.com, http://b.com ,http://c.com")
    assert s.cors_origins_list == ["http://a.com", "http://b.com", "http://c.com"]


def test_summarize_configured_requires_key_and_model():
    assert Settings(
        summarize_enabled=True,
        openrouter_api_key="k",
        summarize_model="anthropic/claude-haiku-4.5",
    ).summarize_configured is True


def test_summarize_not_configured_without_key():
    assert Settings(
        summarize_enabled=True, openrouter_api_key="", summarize_model="x/y"
    ).summarize_configured is False


def test_summarize_not_configured_without_model():
    assert Settings(
        summarize_enabled=True, openrouter_api_key="k", summarize_model=""
    ).summarize_configured is False


def test_summarize_not_configured_when_disabled():
    assert Settings(
        summarize_enabled=False, openrouter_api_key="k", summarize_model="x/y"
    ).summarize_configured is False
