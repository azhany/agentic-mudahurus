from mudahurus_migrate import transforms as T


def test_tenant_uuid_deterministic():
    assert T.tenant_uuid(5) == T.tenant_uuid(5)
    assert T.tenant_uuid(5) != T.tenant_uuid(6)


def test_transform_order_normalizes_to_header_item_payment():
    legacy = {
        "id": 10, "user_id": 5, "sku": "ABC", "quantity": 3, "unit_price": "10.00",
        "total_price": "30.00", "status": "payment_accepted", "full_name": "Siti",
        "email": "s@example.com", "contact_no": "012", "mailing_addr": "Jln 1",
        "mailing_addr2": "", "city": "KL", "postcode": "50000", "state": "WP",
        "additional_notes": "leave at door", "payment_image_proof": "proof.jpg",
        "insert_date": "2026-01-01T00:00:00",
    }
    out = T.transform_order(legacy)
    assert out["order"]["tenant_id"] == T.tenant_uuid(5)
    assert out["order"]["shipping_address"]["city"] == "KL"
    assert out["item"]["quantity"] == 3
    assert out["item"]["line_total"] == 30.0
    assert out["payment"] is not None
    # accepted/shipped orders -> payment verified
    assert out["payment"]["status"] == "verified"


def test_transform_order_without_proof_has_no_payment():
    out = T.transform_order({"id": 1, "user_id": 2, "sku": "X", "quantity": 1,
                             "unit_price": "5", "status": "pending"})
    assert out["payment"] is None
    # default 3-day expiry derived when none present
    assert out["order"]["expired_date"]


def test_transform_product_slug_and_price_cleansing():
    out = T.transform_product({"id": 1, "user_id": 2, "product_name": "Kuih Lapis!",
                               "unit_price": "1,250.50", "status": "", "category_id": 9})
    assert out["url_slug"] == "kuih-lapis"
    assert out["unit_price"] == 1250.5
    assert out["status"] == "active"
    assert out["category_id"] == T.row_uuid("category", 9, 2)


def test_profile_rows():
    p = T.profile_rows([{"a": 1, "b": None}, {"a": 2, "b": ""}])
    assert p["count"] == 2
    assert p["columns"]["b"]["null_pct"] == 100.0
