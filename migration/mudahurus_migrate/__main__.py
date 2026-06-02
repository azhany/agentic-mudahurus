"""Migration CLI: profile | migrate | reconcile (MH-406).

Drivers (pymysql, psycopg, boto3) are optional; absent any of them the relevant
step degrades to a clear message instead of crashing.
"""
from __future__ import annotations

import argparse
import json
import sys
from typing import Any, Dict, List

from . import transforms as T

LEGACY_TABLES = {
    "users": "SELECT * FROM users",
    "categories": "SELECT * FROM mudahurus_products_category",
    "products": "SELECT * FROM mudahurus_products",
    "orders": "SELECT * FROM mudahurus_orders",
    "customers": "SELECT * FROM mudahurus_customers",
    "coupons": "SELECT * FROM mudahurus_coupons",
}


def _mysql_rows(mysql_url: str, query: str) -> List[Dict[str, Any]]:
    import pymysql  # type: ignore
    from urllib.parse import urlparse

    u = urlparse(mysql_url)
    conn = pymysql.connect(
        host=u.hostname, port=u.port or 3306, user=u.username,
        password=u.password or "", database=u.path.lstrip("/"),
        cursorclass=pymysql.cursors.DictCursor,
    )
    try:
        with conn.cursor() as cur:
            cur.execute(query)
            return list(cur.fetchall())
    finally:
        conn.close()


def cmd_profile(args) -> int:
    report: Dict[str, Any] = {}
    for name, query in LEGACY_TABLES.items():
        try:
            rows = _mysql_rows(args.mysql_url, query)
            report[name] = T.profile_rows(rows)
        except Exception as e:  # driver missing or table absent
            report[name] = {"error": str(e)}
    print(json.dumps(report, indent=2, default=str))
    return 0


def cmd_migrate(args) -> int:
    try:
        import psycopg  # type: ignore
    except Exception:
        print("psycopg not installed; install extras: pip install -e '.[full]'", file=sys.stderr)
        return 2

    counts = {k: 0 for k in ("tenants", "categories", "products", "orders", "order_items", "payments", "customers", "coupons")}
    conn = psycopg.connect(args.postgres_url)
    conn.autocommit = False
    try:
        with conn.cursor() as cur:
            for row in _safe_rows(args.mysql_url, LEGACY_TABLES["users"]):
                t = T.transform_user(row)
                cur.execute(
                    """INSERT INTO tenants (id, legacy_user_id, username, email, password_hash, role, full_name, store_name, phone)
                       VALUES (%(id)s,%(legacy_user_id)s,%(username)s,%(email)s,%(password_hash)s,%(role)s,%(full_name)s,%(store_name)s,%(phone)s)
                       ON CONFLICT (id) DO UPDATE SET username=EXCLUDED.username, email=EXCLUDED.email""",
                    t)
                counts["tenants"] += 1

            for row in _safe_rows(args.mysql_url, LEGACY_TABLES["categories"]):
                c = T.transform_category(row)
                cur.execute(
                    """INSERT INTO categories (id, tenant_id, name, description)
                       VALUES (%(id)s,%(tenant_id)s,%(name)s,%(description)s)
                       ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name""", c)
                counts["categories"] += 1

            for row in _safe_rows(args.mysql_url, LEGACY_TABLES["products"]):
                p = T.transform_product(row)
                p.pop("_legacy_image", None)
                cur.execute(
                    """INSERT INTO products (id, tenant_id, category_id, sku, product_name, description, unit_price, url_slug, image_key, status)
                       VALUES (%(id)s,%(tenant_id)s,%(category_id)s,%(sku)s,%(product_name)s,%(description)s,%(unit_price)s,%(url_slug)s,%(image_key)s,%(status)s)
                       ON CONFLICT (id) DO UPDATE SET product_name=EXCLUDED.product_name, unit_price=EXCLUDED.unit_price""", p)
                counts["products"] += 1

            for row in _safe_rows(args.mysql_url, LEGACY_TABLES["orders"]):
                norm = T.transform_order(row)
                o, item, payment = norm["order"], norm["item"], norm["payment"]
                cur.execute(
                    """INSERT INTO orders (id, tenant_id, status, full_name, email, contact_no, shipping_address, additional_notes, total_price, expired_date)
                       VALUES (%(id)s,%(tenant_id)s,%(status)s,%(full_name)s,%(email)s,%(contact_no)s,%(shipping_address)s,%(additional_notes)s,%(total_price)s,%(expired_date)s)
                       ON CONFLICT (id) DO NOTHING""",
                    {**o, "shipping_address": json.dumps(o["shipping_address"])})
                counts["orders"] += 1
                cur.execute(
                    """INSERT INTO order_items (id, order_id, sku, quantity, unit_price, line_total)
                       VALUES (%(id)s,%(order_id)s,%(sku)s,%(quantity)s,%(unit_price)s,%(line_total)s)
                       ON CONFLICT (id) DO NOTHING""", item)
                counts["order_items"] += 1
                if payment:
                    payment.pop("_legacy_proof", None)
                    cur.execute(
                        """INSERT INTO payments (id, order_id, proof_key, amount, status)
                           VALUES (%(id)s,%(order_id)s,%(proof_key)s,%(amount)s,%(status)s)
                           ON CONFLICT (id) DO NOTHING""", payment)
                    counts["payments"] += 1

            for row in _safe_rows(args.mysql_url, LEGACY_TABLES["customers"]):
                c = T.transform_customer(row)
                cur.execute(
                    """INSERT INTO customers (id, tenant_id, full_name, ic_no, dob, email, contact_no, mailing_addr, city, postcode, state, customer_loyalty_code, type)
                       VALUES (%(id)s,%(tenant_id)s,%(full_name)s,%(ic_no)s,%(dob)s,%(email)s,%(contact_no)s,%(mailing_addr)s,%(city)s,%(postcode)s,%(state)s,%(customer_loyalty_code)s,%(type)s)
                       ON CONFLICT (id) DO NOTHING""", c)
                counts["customers"] += 1

            for row in _safe_rows(args.mysql_url, LEGACY_TABLES["coupons"]):
                c = T.transform_coupon(row)
                cur.execute(
                    """INSERT INTO coupons (id, tenant_id, product_id, campaign, description, expired_date)
                       VALUES (%(id)s,%(tenant_id)s,%(product_id)s,%(campaign)s,%(description)s,%(expired_date)s)
                       ON CONFLICT (id) DO NOTHING""", c)
                counts["coupons"] += 1
        conn.commit()
    except Exception as e:
        conn.rollback()
        print(f"migration failed, rolled back: {e}", file=sys.stderr)
        return 1
    finally:
        conn.close()
    print(json.dumps({"loaded": counts}, indent=2))
    return 0


def cmd_reconcile(args) -> int:
    from .reconcile import reconcile
    report = reconcile(args.mysql_url, args.postgres_url)
    print(json.dumps(report, indent=2, default=str))
    return 0 if report.get("ok") else 1


def _safe_rows(mysql_url: str, query: str) -> List[Dict[str, Any]]:
    try:
        return _mysql_rows(mysql_url, query)
    except Exception as e:
        print(f"warning: could not read ({query}): {e}", file=sys.stderr)
        return []


def main(argv=None) -> int:
    parser = argparse.ArgumentParser(prog="mudahurus_migrate")
    sub = parser.add_subparsers(dest="cmd", required=True)
    for name in ("profile", "migrate", "reconcile"):
        sp = sub.add_parser(name)
        sp.add_argument("--mysql-url", default="")
        sp.add_argument("--postgres-url", default="")
    args = parser.parse_args(argv)
    return {"profile": cmd_profile, "migrate": cmd_migrate, "reconcile": cmd_reconcile}[args.cmd](args)


if __name__ == "__main__":
    raise SystemExit(main())
