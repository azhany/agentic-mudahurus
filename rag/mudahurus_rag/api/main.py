"""FastAPI: retrieval + read-only assistant (EP-9).

Endpoints
  POST /retrieve       — grounded, tenant-scoped chunks (MH-601)
  POST /assistant/ask  — grounded answer or refusal (MH-602)
  POST /ingest         — trigger ingestion for a tenant (event-driven, MH-506)
  GET  /healthz /readyz /metrics

The tenant_id arrives in the request body from the Go proxy, which resolved it
server-side from the JWT or /store/{username}. This service treats tenant_id as
authoritative and applies it as a mandatory filter; it never reads tenant from
any client-controllable place beyond this trusted server-to-server call.
"""
from __future__ import annotations

import time
from typing import List, Optional

from fastapi import FastAPI
from fastapi.responses import JSONResponse, PlainTextResponse
from pydantic import BaseModel

from ..config import get_settings
from ..engine import get_engine
from ..ingestion.pipeline import ingest_tenant

app = FastAPI(title="MUDAHURUS RAG", version="1.0.0")

# --- minimal Prometheus-style metrics (no external dep) ---
_METRICS = {"retrieve_total": 0, "assistant_total": 0, "assistant_refused": 0, "ingest_total": 0}
_LATENCY_MS: List[float] = []


class RetrieveRequest(BaseModel):
    tenant_id: str
    query: str
    top_k: int = 5
    source_types: Optional[List[str]] = None


class RetrieveHit(BaseModel):
    text: str
    score: float
    source_type: str
    source_id: str


class AssistantRequest(BaseModel):
    tenant_id: str
    question: str
    scope: str = "admin"
    top_k: int = 5


class CitationModel(BaseModel):
    source_type: str
    source_id: str
    score: float


class AssistantResponse(BaseModel):
    answer: str
    grounded: bool
    refused: bool
    citations: List[CitationModel]


class IngestRequest(BaseModel):
    tenant_id: str
    changed_since: Optional[str] = None


@app.get("/healthz")
def healthz():
    return {"status": "ok"}


@app.get("/readyz")
def readyz():
    eng = get_engine()
    return {"status": "ready", "embedding_dim": eng.embedder.dim,
            "vector_store": type(eng.store).__name__}


@app.post("/retrieve", response_model=List[RetrieveHit])
def retrieve(req: RetrieveRequest):
    eng = get_engine()
    start = time.perf_counter()
    chunks = eng.retrieval.retrieve(req.tenant_id, req.query, req.top_k, req.source_types)
    _LATENCY_MS.append((time.perf_counter() - start) * 1000.0)
    _METRICS["retrieve_total"] += 1
    return [RetrieveHit(text=c.text, score=c.score, source_type=c.source_type, source_id=c.source_id)
            for c in chunks]


@app.post("/assistant/ask", response_model=AssistantResponse)
def assistant_ask(req: AssistantRequest):
    eng = get_engine()
    ans = eng.assistant.ask(req.tenant_id, req.question, req.scope, req.top_k)
    _METRICS["assistant_total"] += 1
    if ans.refused:
        _METRICS["assistant_refused"] += 1
    return AssistantResponse(
        answer=ans.answer, grounded=ans.grounded, refused=ans.refused,
        citations=[CitationModel(source_type=c.source_type, source_id=c.source_id, score=c.score)
                   for c in ans.citations],
    )


@app.post("/ingest")
def ingest(req: IngestRequest):
    eng = get_engine()
    s = get_settings()
    report = ingest_tenant(s.rag_database_url, req.tenant_id, eng.embedder, eng.store, req.changed_since)
    _METRICS["ingest_total"] += 1
    return JSONResponse(report.__dict__)


@app.get("/metrics")
def metrics():
    lines = [f"mudahurus_rag_{k} {v}" for k, v in _METRICS.items()]
    if _LATENCY_MS:
        ordered = sorted(_LATENCY_MS)
        p95 = ordered[min(len(ordered) - 1, int(len(ordered) * 0.95))]
        lines.append(f"mudahurus_rag_retrieve_latency_p95_ms {p95:.2f}")
    return PlainTextResponse("\n".join(lines) + "\n")
