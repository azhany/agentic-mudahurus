# RAG Plane (Python / FastAPI)

Decoupled ingestion + read-only grounded assistant (ARCHITECTURE §7–8, ADR-004).

```bash
pip install -e '.[dev]'                 # core + test deps (offline fallbacks)
pip install -e '.[embeddings,vector,ocr,db]'   # production backends
pytest -q                               # tests (run with pure-Python fallbacks)
uvicorn mudahurus_rag.api.main:app --reload     # :8000
```

- `vectorstore.py` Qdrant (+ in-memory fallback) with a MANDATORY tenant filter.
- `embeddings.py` bge-m3 via sentence-transformers (+ hashing fallback).
- `ingestion/` extract (PII-excluded) → ocr → chunk → embed → upsert (idempotent).
- `assistant/` grounding + refusal; `retrieval/` tenant-scoped top-k.
- `airflow/dags/mudahurus_ingest.py` per-tenant scheduled + event-driven DAG.
- `enhancements/` post-v1 agent orchestration scaffold (EH-1).

Pure-Python fallbacks let the whole pipeline run and be tested offline; install
the extras for production-grade embeddings, Qdrant, OCR and Postgres.
