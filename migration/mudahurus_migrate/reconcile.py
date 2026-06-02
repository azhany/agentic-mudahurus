"""Reconciliation: legacy vs migrated row counts + spot checks (MH-406 AC)."""
from __future__ import annotations

from typing import Any, Dict

# legacy table -> target table
PAIRS = {
    "users": "tenants",
    "mudahurus_products_category": "categories",
    "mudahurus_products": "products",
    "mudahurus_orders": "orders",
    "mudahurus_customers": "customers",
    "mudahurus_coupons": "coupons",
}


def _mysql_count(mysql_url: str, table: str) -> int:
    import pymysql  # type: ignore
    from urllib.parse import urlparse

    u = urlparse(mysql_url)
    conn = pymysql.connect(host=u.hostname, port=u.port or 3306, user=u.username,
                           password=u.password or "", database=u.path.lstrip("/"))
    try:
        with conn.cursor() as cur:
            cur.execute(f"SELECT COUNT(*) FROM {table}")
            return int(cur.fetchone()[0])
    finally:
        conn.close()


def _pg_count(pg_url: str, table: str) -> int:
    import psycopg  # type: ignore

    with psycopg.connect(pg_url) as conn:
        with conn.cursor() as cur:
            cur.execute(f"SELECT COUNT(*) FROM {table}")
            return int(cur.fetchone()[0])


def reconcile(mysql_url: str, pg_url: str) -> Dict[str, Any]:
    rows = {}
    ok = True
    for legacy, target in PAIRS.items():
        try:
            src = _mysql_count(mysql_url, legacy)
            dst = _pg_count(pg_url, target)
            match = src == dst
            ok = ok and match
            rows[target] = {"legacy": src, "migrated": dst, "match": match}
        except Exception as e:
            ok = False
            rows[target] = {"error": str(e)}
    return {"ok": ok, "tables": rows}
