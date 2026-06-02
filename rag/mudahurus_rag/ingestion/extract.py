"""Structured extractor (MH-503, FR-9.1).

Pulls per-tenant products / categories / orders / customers from Postgres and
normalizes each record into a text document for embedding.

PII policy (PRD §6 Privacy): customer IC number, DOB and contact number are
EXCLUDED from embedded text by default. Only non-PII, retrieval-useful fields
are embedded. Supports incremental extraction via a `changed_since` cutoff.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional


@dataclass
class Document:
    tenant_id: str
    source_type: str          # product | category | order | customer | payment_doc
    source_id: str
    text: str
    payload: Dict[str, Any] = field(default_factory=dict)


def _connect(database_url: str):
    import psycopg  # type: ignore

    return psycopg.connect(database_url)


def extract_for_tenant(database_url: str, tenant_id: str,
                       changed_since: Optional[str] = None) -> List[Document]:
    """Extract & normalize a tenant's records. Returns [] if psycopg is absent
    (dev/test without a DB) — callers handle the empty case gracefully."""
    try:
        conn = _connect(database_url)
    except Exception:
        return []

    docs: List[Document] = []
    try:
        with conn.cursor() as cur:
            docs += _extract_products(cur, tenant_id, changed_since)
            docs += _extract_categories(cur, tenant_id, changed_since)
            docs += _extract_orders(cur, tenant_id, changed_since)
            docs += _extract_customers(cur, tenant_id, changed_since)
    finally:
        conn.close()
    return docs


def _since_clause(changed_since: Optional[str], col: str = "updated_at") -> str:
    return f" AND {col} > %(since)s" if changed_since else ""


def _extract_products(cur, tenant_id: str, since: Optional[str]) -> List[Document]:
    cur.execute(
        "SELECT id, sku, product_name, description, unit_price, status "
        "FROM products WHERE tenant_id=%(tid)s" + _since_clause(since),
        {"tid": tenant_id, "since": since},
    )
    out = []
    for row in cur.fetchall():
        pid, sku, name, desc, price, status = row
        text = (
            f"Product: {name}\nSKU: {sku}\nPrice: RM{price}\n"
            f"Status: {status}\nDescription: {desc or ''}"
        )
        out.append(Document(tenant_id, "product", str(pid), text,
                            {"name": name, "sku": sku, "price": str(price), "status": status}))
    return out


def _extract_categories(cur, tenant_id: str, since: Optional[str]) -> List[Document]:
    cur.execute(
        "SELECT id, name, description FROM categories WHERE tenant_id=%(tid)s" + _since_clause(since),
        {"tid": tenant_id, "since": since},
    )
    return [
        Document(tenant_id, "category", str(cid), f"Category: {name}\n{desc or ''}", {"name": name})
        for cid, name, desc in cur.fetchall()
    ]


def _extract_orders(cur, tenant_id: str, since: Optional[str]) -> List[Document]:
    cur.execute(
        "SELECT id, status, full_name, total_price, created_at FROM orders "
        "WHERE tenant_id=%(tid)s" + _since_clause(since),
        {"tid": tenant_id, "since": since},
    )
    out = []
    for oid, status, full_name, total, created in cur.fetchall():
        # Note: full_name is shown to the seller's own assistant only (tenant-scoped).
        text = f"Order {oid}\nStatus: {status}\nCustomer: {full_name}\nTotal: RM{total}\nDate: {created}"
        out.append(Document(tenant_id, "order", str(oid), text,
                            {"status": status, "total": str(total)}))
    return out


def _extract_customers(cur, tenant_id: str, since: Optional[str]) -> List[Document]:
    # PII (ic_no, dob, contact_no) intentionally NOT selected for embedding.
    cur.execute(
        "SELECT id, full_name, customer_loyalty_code, type, city, state "
        "FROM customers WHERE tenant_id=%(tid)s" + _since_clause(since),
        {"tid": tenant_id, "since": since},
    )
    out = []
    for cid, name, loyalty, ctype, city, state in cur.fetchall():
        text = f"Customer: {name}\nLoyalty: {loyalty}\nType: {ctype}\nLocation: {city}, {state}"
        out.append(Document(tenant_id, "customer", str(cid), text,
                            {"loyalty_code": loyalty, "type": ctype}))
    return out


def list_tenant_ids(database_url: str) -> List[str]:
    try:
        conn = _connect(database_url)
    except Exception:
        return []
    try:
        with conn.cursor() as cur:
            cur.execute("SELECT id FROM tenants")
            return [str(r[0]) for r in cur.fetchall()]
    finally:
        conn.close()
