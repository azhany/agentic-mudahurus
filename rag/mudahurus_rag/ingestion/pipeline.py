"""Ingestion pipeline orchestration (MH-505, DAG contract ARCHITECTURE §7):
extract → (OCR) → chunk → embed → upsert(Qdrant) → index health check.

Idempotent: deterministic point IDs mean re-runs upsert in place (no dupes).
"""
from __future__ import annotations

from dataclasses import dataclass
from typing import List, Optional

from ..chunking import chunk_text
from ..embeddings import Embedder
from ..vectorstore import Point, VectorStore, deterministic_point_id
from .extract import Document, extract_for_tenant


@dataclass
class IngestReport:
    tenant_id: str
    documents: int
    chunks: int
    upserted: int
    index_count: int
    ok: bool


def documents_to_points(documents: List[Document], embedder: Embedder) -> List[Point]:
    texts: List[str] = []
    metas: List[tuple] = []
    for doc in documents:
        for chunk in chunk_text(doc.text):
            texts.append(chunk.text)
            metas.append((doc, chunk.chunk_no))
    if not texts:
        return []
    vectors = embedder.embed(texts)
    points: List[Point] = []
    for (doc, chunk_no), vec, text in zip(metas, vectors, texts):
        pid = deterministic_point_id(doc.tenant_id, doc.source_type, doc.source_id, chunk_no)
        payload = {
            "tenant_id": doc.tenant_id,
            "source_type": doc.source_type,
            "source_id": doc.source_id,
            "chunk_no": chunk_no,
            "text": text,
            **doc.payload,
        }
        points.append(Point(id=pid, vector=vec, payload=payload))
    return points


def ingest_tenant(
    database_url: str,
    tenant_id: str,
    embedder: Embedder,
    store: VectorStore,
    changed_since: Optional[str] = None,
    extra_documents: Optional[List[Document]] = None,
) -> IngestReport:
    store.ensure_collection(embedder.dim)
    documents = extract_for_tenant(database_url, tenant_id, changed_since)
    if extra_documents:
        documents += extra_documents
    points = documents_to_points(documents, embedder)
    upserted = store.upsert(points) if points else 0
    index_count = store.count(tenant_id)
    return IngestReport(
        tenant_id=tenant_id,
        documents=len(documents),
        chunks=len(points),
        upserted=upserted,
        index_count=index_count,
        ok=True,
    )
