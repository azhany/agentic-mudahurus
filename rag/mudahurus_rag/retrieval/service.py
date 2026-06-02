"""Retrieval service (MH-601, FR-9.4).

Returns grounded, tenant-scoped chunks. The tenant_id filter is applied by the
vector store on every search and is NEVER taken from client input here — the
caller (FastAPI) receives it from the Go proxy which resolved it server-side.
"""
from __future__ import annotations

from dataclasses import dataclass
from typing import List, Optional

from ..embeddings import Embedder
from ..vectorstore import SearchHit, VectorStore


@dataclass
class RetrievedChunk:
    text: str
    score: float
    source_type: str
    source_id: str


class RetrievalService:
    def __init__(self, embedder: Embedder, store: VectorStore) -> None:
        self._embedder = embedder
        self._store = store

    def retrieve(self, tenant_id: str, query: str, top_k: int = 5,
                 source_types: Optional[List[str]] = None) -> List[RetrievedChunk]:
        if not tenant_id:
            raise ValueError("tenant_id is required")
        if not query.strip():
            return []
        vector = self._embedder.embed([query])[0]
        hits: List[SearchHit] = self._store.search(tenant_id, vector, top_k, source_types)
        return [
            RetrievedChunk(
                text=h.payload.get("text", ""),
                score=h.score,
                source_type=h.payload.get("source_type", ""),
                source_id=h.payload.get("source_id", ""),
            )
            for h in hits
        ]
