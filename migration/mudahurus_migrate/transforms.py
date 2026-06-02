"""Pure transform functions (legacy row dict -> target row dict).

Kept side-effect-free so they are unit-testable without any database. The CLI
(`__main__.py`) wires these between MySQL reads and PostgreSQL writes.
"""
from __future__ import annotations

import re
import uuid
from datetime import datetime, timedelta
from typing import Any, Dict, List, Optional

NAMESPACE = uuid.UUID("00000000-0000-0000-0000-00000000ab12")


def tenant_uuid(legacy_user_id: Any) -> str:
    """Deterministic UUID from legacy user id so re-runs are idempotent."""
    return str(uuid.uuid5(NAMESPACE, f"tenant:{legacy_user_id}"))


def row_uuid(kind: str, legacy_id: Any, user_id: Any) -> str:
    return str(uuid.uuid5(NAMESPACE, f"{kind}:{user_id}:{legacy_id}"))


def slugify(value: str) -> str:
    value = (value or "").strip().lower()
    value = re.sub(r"[^a-z0-9]+", "-", value)
    return value.strip("-")


def clean_str(v: Any) -> str:
    return "" if v is None else str(v).strip()


def clean_price(v: Any) -> float:
    if v is None or v == "":
        return 0.0
    try:
        return float(str(v).replace(",", ""))
    except ValueError:
        return 0.0


def transform_user(row: Dict[str, Any]) -> Dict[str, Any]:
    full_name = " ".join(
        clean_str(row.get(k)) for k in ("first_name", "last_name") if row.get(k)
    ).strip()
    return {
        "id": tenant_uuid(row["id"]),
        "legacy_user_id": row["id"],
        "username": clean_str(row.get("username")).lower(),
        "email": clean_str(row.get("email")).lower(),
        # Legacy hash is preserved but flagged; users re-hash to Argon2id on next login.
        "password_hash": clean_str(row.get("password")) or "!legacy",
        "role": "seller",
        "full_name": full_name,
        "store_name": clean_str(row.get("company")),
        "phone": clean_str(row.get("phone")),
    }


def transform_category(row: Dict[str, Any]) -> Dict[str, Any]:
    uid = row["user_id"]
    return {
        "id": row_uuid("category", row["id"], uid),
        "tenant_id": tenant_uuid(uid),
        "name": clean_str(row.get("category")) or clean_str(row.get("name")),
        "description": clean_str(row.get("description")),
    }


def transform_product(row: Dict[str, Any]) -> Dict[str, Any]:
    uid = row["user_id"]
    name = clean_str(row.get("product_name"))
    cat = row.get("category_id")
    return {
        "id": row_uuid("product", row["id"], uid),
        "tenant_id": tenant_uuid(uid),
        "category_id": row_uuid("category", cat, uid) if cat else None,
        "sku": clean_str(row.get("sku")),
        "product_name": name,
        "description": clean_str(row.get("description")),
        "unit_price": clean_price(row.get("unit_price")),
        "url_slug": slugify(name),
        "image_key": "",  # set during storage backfill
        "_legacy_image": clean_str(row.get("image")),
        "status": "active" if clean_str(row.get("status")) in ("", "active") else "inactive",
    }


def _expiry(row: Dict[str, Any]) -> str:
    exp = row.get("expired_date")
    if exp:
        return str(exp)
    base = row.get("insert_date") or datetime.utcnow()
    if isinstance(base, str):
        try:
            base = datetime.fromisoformat(base)
        except ValueError:
            base = datetime.utcnow()
    return (base + timedelta(days=3)).isoformat()


def transform_order(row: Dict[str, Any]) -> Dict[str, Any]:
    """Normalize a flat legacy order into header + one item + optional payment."""
    uid = row["user_id"]
    oid = row_uuid("order", row["id"], uid)
    shipping = {
        "mailing_addr": clean_str(row.get("mailing_addr")),
        "mailing_addr2": clean_str(row.get("mailing_addr2")),
        "city": clean_str(row.get("city")),
        "postcode": clean_str(row.get("postcode")),
        "state": clean_str(row.get("state")),
    }
    qty = int(row.get("quantity") or 1)
    unit = clean_price(row.get("unit_price"))
    total = clean_price(row.get("total_price")) or (qty * unit)
    header = {
        "id": oid,
        "tenant_id": tenant_uuid(uid),
        "status": clean_str(row.get("status")) or "pending",
        "full_name": clean_str(row.get("full_name")),
        "email": clean_str(row.get("email")),
        "contact_no": clean_str(row.get("contact_no")),
        "shipping_address": shipping,
        "additional_notes": clean_str(row.get("additional_notes")),
        "total_price": total,
        "expired_date": _expiry(row),
    }
    item = {
        "id": row_uuid("order_item", row["id"], uid),
        "order_id": oid,
        "sku": clean_str(row.get("sku")),
        "quantity": qty,
        "unit_price": unit,
        "line_total": qty * unit,
    }
    payment: Optional[Dict[str, Any]] = None
    proof = clean_str(row.get("payment_image_proof"))
    if proof:
        status = "verified" if header["status"] in ("payment_accepted", "shipped") else "submitted"
        payment = {
            "id": row_uuid("payment", row["id"], uid),
            "order_id": oid,
            "proof_key": "",         # set during storage backfill
            "_legacy_proof": proof,
            "amount": total,
            "status": status,
        }
    return {"order": header, "item": item, "payment": payment}


def transform_customer(row: Dict[str, Any]) -> Dict[str, Any]:
    uid = row["user_id"]
    return {
        "id": row_uuid("customer", row["id"], uid),
        "tenant_id": tenant_uuid(uid),
        "full_name": clean_str(row.get("full_name")),
        "ic_no": clean_str(row.get("ic_no")),
        "dob": row.get("dob") or None,
        "email": clean_str(row.get("email")),
        "contact_no": clean_str(row.get("contact_no")),
        "mailing_addr": clean_str(row.get("mailing_addr")),
        "city": clean_str(row.get("city")),
        "postcode": clean_str(row.get("postcode")),
        "state": clean_str(row.get("state")),
        "customer_loyalty_code": clean_str(row.get("customer_loyalty_code")),
        "type": clean_str(row.get("type")) or "regular",
    }


def transform_coupon(row: Dict[str, Any]) -> Dict[str, Any]:
    uid = row["user_id"]
    pid = row.get("product_id")
    return {
        "id": row_uuid("coupon", row["id"], uid),
        "tenant_id": tenant_uuid(uid),
        "product_id": row_uuid("product", pid, uid) if pid else None,
        "campaign": clean_str(row.get("campaign")),
        "description": clean_str(row.get("description")),
        "expired_date": row.get("expired_date") or None,
    }


def profile_rows(rows: List[Dict[str, Any]]) -> Dict[str, Any]:
    """Lightweight data-quality profile of a legacy table (MH-406 profiling)."""
    if not rows:
        return {"count": 0, "columns": {}}
    cols = {}
    for col in rows[0].keys():
        values = [r.get(col) for r in rows]
        nulls = sum(1 for v in values if v is None or v == "")
        cols[col] = {"null_pct": round(100 * nulls / len(values), 1)}
    return {"count": len(rows), "columns": cols}
