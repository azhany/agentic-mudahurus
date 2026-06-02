"""Vector store abstraction (MH-501).

Qdrant in production; a pure-Python in-memory cosine store as the dev/test
fallback. Both enforce a mandatory tenant_id payload filter on every search —
the filter is built server-side and can never be omitted (ARCHITECTURE §5, §8).
"""
from __future__ import annotations

import math
import uuid
from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional


@dataclass
class Point:
    id: str
    vector: List[float]
    payload: Dict[str, Any]


@dataclass
class SearchHit:
    id: str
    score: float
    payload: Dict[str, Any]


def deterministic_point_id(tenant_id: str, source_type: str, source_id: str, chunk_no: int) -> str:
    """Idempotent point id keyed on (tenant, source_type, source_id, chunk_no)
    (ARCHITECTURE §7). Re-runs upsert in place; no duplicates (MH-505)."""
    name = f"{tenant_id}:{source_type}:{source_id}:{chunk_no}"
    return str(uuid.uuid5(uuid.NAMESPACE_URL, name))


class VectorStore:
    def ensure_collection(self, dim: int) -> None: ...
    def upsert(self, points: List[Point]) -> int: ...
    def search(self, tenant_id: str, vector: List[float], top_k: int,
               source_types: Optional[List[str]] = None) -> List[SearchHit]: ...
    def count(self, tenant_id: Optional[str] = None) -> int: ...


@dataclass
class InMemoryVectorStore(VectorStore):
    """Dev/test backend. Mirrors the Qdrant tenant-filter semantics exactly."""
    _points: Dict[str, Point] = field(default_factory=dict)

    def ensure_collection(self, dim: int) -> None:
        return None

    def upsert(self, points: List[Point]) -> int:
        for p in points:
            self._points[p.id] = p
        return len(points)

    def search(self, tenant_id: str, vector: List[float], top_k: int,
               source_types: Optional[List[str]] = None) -> List[SearchHit]:
        if not tenant_id:
            raise ValueError("tenant_id filter is mandatory")
        hits: List[SearchHit] = []
        for p in self._points.values():
            # MANDATORY tenant isolation
            if p.payload.get("tenant_id") != tenant_id:
                continue
            if source_types and p.payload.get("source_type") not in source_types:
                continue
            hits.append(SearchHit(id=p.id, score=_cosine(vector, p.vector), payload=p.payload))
        hits.sort(key=lambda h: h.score, reverse=True)
        return hits[:top_k]

    def count(self, tenant_id: Optional[str] = None) -> int:
        if tenant_id is None:
            return len(self._points)
        return sum(1 for p in self._points.values() if p.payload.get("tenant_id") == tenant_id)


class QdrantVectorStore(VectorStore):
    """Production backend backed by qdrant-client."""

    def __init__(self, url: str, collection: str) -> None:
        from qdrant_client import QdrantClient  # type: ignore

        self._client = QdrantClient(url=url)
        self._collection = collection

    def ensure_collection(self, dim: int) -> None:
        from qdrant_client.models import Distance, VectorParams  # type: ignore

        existing = {c.name for c in self._client.get_collections().collections}
        if self._collection not in existing:
            self._client.create_collection(
                collection_name=self._collection,
                vectors_config=VectorParams(size=dim, distance=Distance.COSINE),
            )

    def upsert(self, points: List[Point]) -> int:
        from qdrant_client.models import PointStruct  # type: ignore

        self._client.upsert(
            collection_name=self._collection,
            points=[PointStruct(id=p.id, vector=p.vector, payload=p.payload) for p in points],
        )
        return len(points)

    def search(self, tenant_id: str, vector: List[float], top_k: int,
               source_types: Optional[List[str]] = None) -> List[SearchHit]:
        if not tenant_id:
            raise ValueError("tenant_id filter is mandatory")
        from qdrant_client.models import FieldCondition, Filter, MatchAny, MatchValue  # type: ignore

        must = [FieldCondition(key="tenant_id", match=MatchValue(value=tenant_id))]
        if source_types:
            must.append(FieldCondition(key="source_type", match=MatchAny(any=source_types)))
        res = self._client.search(
            collection_name=self._collection,
            query_vector=vector,
            query_filter=Filter(must=must),
            limit=top_k,
        )
        return [SearchHit(id=str(r.id), score=float(r.score), payload=dict(r.payload or {})) for r in res]

    def count(self, tenant_id: Optional[str] = None) -> int:
        from qdrant_client.models import FieldCondition, Filter, MatchValue  # type: ignore

        flt = None
        if tenant_id:
            flt = Filter(must=[FieldCondition(key="tenant_id", match=MatchValue(value=tenant_id))])
        return self._client.count(collection_name=self._collection, count_filter=flt).count


def build_vector_store(url: str, collection: str) -> VectorStore:
    try:
        return QdrantVectorStore(url, collection)
    except Exception:
        return InMemoryVectorStore()


def _cosine(a: List[float], b: List[float]) -> float:
    if len(a) != len(b):
        n = min(len(a), len(b))
        a, b = a[:n], b[:n]
    dot = sum(x * y for x, y in zip(a, b))
    na = math.sqrt(sum(x * x for x in a))
    nb = math.sqrt(sum(y * y for y in b))
    if na == 0 or nb == 0:
        return 0.0
    return dot / (na * nb)
