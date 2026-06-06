from functools import lru_cache
from typing import ClassVar

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env", env_file_encoding="utf-8", extra="ignore"
    )

    # Deployment environment: "development" | "production"
    environment: str = "development"

    # CORS — comma-separated list of allowed browser origins
    cors_origins: str = "http://localhost:5173"

    # Database
    database_url: str = "postgresql+asyncpg://postgres:postgres@localhost:5433/aktien_news"
    test_database_url: str = "postgresql+asyncpg://postgres:postgres@localhost:5433/aktien_news_test"

    # Auth / JWT
    jwt_secret: str = "change-me-in-production"
    jwt_algorithm: str = "HS256"
    access_token_expire_minutes: int = 30
    refresh_token_expire_days: int = 14

    # Admin bootstrap
    admin_email: str = ""
    admin_password: str = ""

    # External data sources (CI/tests must not rely on these)
    finnhub_api_key: str = ""
    newsapi_api_key: str = ""
    alphavantage_api_key: str = ""

    # Symbol universe scope
    exchanges: str = "US"  # Finnhub US covers NASDAQ + NYSE

    # Background ingestion scheduler
    enable_scheduler: bool = True

    # Matching thresholds
    match_high_threshold: float = 0.85
    match_min_threshold: float = 0.40

    DEFAULT_JWT_SECRET: ClassVar[str] = "change-me-in-production"

    @property
    def cors_origins_list(self) -> list[str]:
        return [o.strip() for o in self.cors_origins.split(",") if o.strip()]

    def check_production_secrets(self) -> None:
        """Fail fast if booting in production with insecure defaults."""
        if self.environment == "production" and self.jwt_secret == self.DEFAULT_JWT_SECRET:
            raise RuntimeError(
                "JWT_SECRET is still the default value in production. "
                "Set a strong JWT_SECRET before deploying."
            )


@lru_cache
def get_settings() -> Settings:
    return Settings()


settings = get_settings()
