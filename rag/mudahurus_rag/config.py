"""Runtime configuration (12-factor, from env). See .env.example."""
from __future__ import annotations

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", extra="ignore")

    http_addr: str = "0.0.0.0:8000"
    qdrant_url: str = "http://localhost:6333"
    qdrant_collection: str = "mudahurus"
    embedding_model: str = "BAAI/bge-m3"
    embedding_dim: int = 1024
    llm_provider: str = "echo"            # echo | openai | anthropic (pluggable)
    llm_api_key: str = ""
    llm_model: str = ""
    rag_database_url: str = "postgresql://mudahurus:mudahurus@localhost:5432/mudahurus"

    # Retrieval defaults
    top_k: int = 5
    min_score: float = 0.25               # below this, the assistant refuses


_settings: Settings | None = None


def get_settings() -> Settings:
    global _settings
    if _settings is None:
        _settings = Settings()
    return _settings
