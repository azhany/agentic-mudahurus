"""Shared singletons wiring the RAG components together."""
from __future__ import annotations

from functools import lru_cache

from .assistant.llm import build_llm
from .assistant.service import AssistantService
from .config import get_settings
from .embeddings import Embedder, build_embedder
from .retrieval.service import RetrievalService
from .vectorstore import VectorStore, build_vector_store


class Engine:
    def __init__(self) -> None:
        s = get_settings()
        self.settings = s
        self.embedder: Embedder = build_embedder(s.embedding_model, s.embedding_dim)
        self.store: VectorStore = build_vector_store(s.qdrant_url, s.qdrant_collection)
        self.store.ensure_collection(self.embedder.dim)
        self.retrieval = RetrievalService(self.embedder, self.store)
        self.assistant = AssistantService(
            self.retrieval, build_llm(s.llm_provider, s.llm_api_key, s.llm_model), s.min_score
        )


@lru_cache(maxsize=1)
def get_engine() -> Engine:
    return Engine()
