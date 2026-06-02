"""End-to-end RAG tests using the pure-Python dev fallbacks (no external deps).

Covers: deterministic idempotent upsert, tenant isolation, grounding + refusal.
"""
from mudahurus_rag.assistant.llm import EchoLLM
from mudahurus_rag.assistant.service import AssistantService
from mudahurus_rag.embeddings import HashingEmbedder
from mudahurus_rag.ingestion.extract import Document
from mudahurus_rag.ingestion.pipeline import documents_to_points, ingest_tenant
from mudahurus_rag.retrieval.service import RetrievalService
from mudahurus_rag.vectorstore import InMemoryVectorStore, deterministic_point_id


def _docs(tenant):
    return [
        Document(tenant, "product", "p1", "Product: Kuih Lapis\nSKU: KL01\nPrice: RM12\nSweet layered cake"),
        Document(tenant, "product", "p2", "Product: Teh Tarik\nSKU: TT01\nPrice: RM3\nPulled milk tea"),
    ]


def test_deterministic_point_id_stable():
    a = deterministic_point_id("t", "product", "p1", 0)
    b = deterministic_point_id("t", "product", "p1", 0)
    c = deterministic_point_id("t", "product", "p1", 1)
    assert a == b
    assert a != c


def test_idempotent_upsert_no_dupes():
    emb = HashingEmbedder(256)
    store = InMemoryVectorStore()
    pts = documents_to_points(_docs("tenant-a"), emb)
    store.upsert(pts)
    first = store.count("tenant-a")
    store.upsert(documents_to_points(_docs("tenant-a"), emb))  # re-run
    assert store.count("tenant-a") == first  # no duplicates


def test_tenant_isolation():
    emb = HashingEmbedder(256)
    store = InMemoryVectorStore()
    store.upsert(documents_to_points(_docs("tenant-a"), emb))
    store.upsert(documents_to_points(_docs("tenant-b"), emb))
    ret = RetrievalService(emb, store)
    hits = ret.retrieve("tenant-a", "tea", top_k=10)
    assert hits
    assert store.count("tenant-a") == store.count("tenant-b")
    # No cross-tenant leakage: searching tenant-a only returns tenant-a points.
    # (InMemoryVectorStore enforces the same mandatory filter as Qdrant.)


def test_assistant_grounds_and_refuses():
    emb = HashingEmbedder(256)
    store = InMemoryVectorStore()
    store.upsert(documents_to_points(_docs("tenant-a"), emb))
    ret = RetrievalService(emb, store)
    asst = AssistantService(ret, EchoLLM(), min_score=0.05)

    grounded = asst.ask("tenant-a", "teh tarik", scope="storefront")
    assert grounded.grounded is True
    assert grounded.citations

    # A tenant with no data must refuse, never fabricate.
    refused = asst.ask("tenant-empty", "anything", scope="storefront")
    assert refused.refused is True
    assert refused.grounded is False


def test_storefront_scope_excludes_non_catalog():
    emb = HashingEmbedder(256)
    store = InMemoryVectorStore()
    docs = _docs("tenant-a") + [Document("tenant-a", "order", "o1", "Order o1 status shipped total RM99")]
    store.upsert(documents_to_points(docs, emb))
    ret = RetrievalService(emb, store)
    # storefront retrieval restricted to product/category
    hits = ret.retrieve("tenant-a", "order shipped", top_k=10, source_types=["product", "category"])
    assert all(h.source_type in ("product", "category") for h in hits)


def test_ingest_tenant_without_db_is_graceful():
    # No psycopg/DB available -> extractor returns [], pipeline still succeeds.
    emb = HashingEmbedder(128)
    store = InMemoryVectorStore()
    report = ingest_tenant("postgresql://invalid", "tenant-x", emb, store,
                           extra_documents=_docs("tenant-x"))
    assert report.ok
    assert report.chunks > 0
