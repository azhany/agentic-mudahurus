"""MUDAHURUS 2.0 RAG plane (Python).

Decoupled data/AI plane (ARCHITECTURE §2, ADR-004): ingestion (extract → OCR →
chunk → embed → upsert to Qdrant) and a read-only, grounded assistant served by
FastAPI. Every vector search is scoped by a mandatory, server-injected tenant_id
filter (ARCHITECTURE §5, §8).
"""

__version__ = "1.0.0"
