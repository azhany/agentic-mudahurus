-- Orders normalized into orders + order_items + payments (ARCHITECTURE §6.1).
-- Legacy embedded a single line item + shipping fields in one row; a parity
-- view (orders_legacy_flat) reproduces that flat shape for parity testing.

CREATE TABLE orders (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    customer_id      UUID,  -- optional link to customers; populated on checkout
    status           TEXT NOT NULL DEFAULT 'pending'
                       CHECK (status IN ('pending','payment_received','payment_accepted','shipped','expired','cancelled','rejected')),
    full_name        TEXT NOT NULL DEFAULT '',
    email            TEXT NOT NULL DEFAULT '',
    contact_no       TEXT NOT NULL DEFAULT '',
    shipping_address JSONB NOT NULL DEFAULT '{}'::jsonb,  -- {mailing_addr, mailing_addr2, city, postcode, state}
    additional_notes TEXT NOT NULL DEFAULT '',
    total_price      NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (total_price >= 0),
    expired_date     TIMESTAMPTZ NOT NULL,               -- now + 3 days on creation (FR-3.2)
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_orders_tenant ON orders(tenant_id);
CREATE INDEX idx_orders_tenant_status ON orders(tenant_id, status);
CREATE INDEX idx_orders_expired ON orders(expired_date) WHERE status = 'pending';

CREATE TABLE order_items (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id    UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id  UUID REFERENCES products(id) ON DELETE SET NULL,
    sku         TEXT NOT NULL,
    product_name TEXT NOT NULL DEFAULT '',
    quantity    INTEGER NOT NULL DEFAULT 1 CHECK (quantity > 0),
    unit_price  NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (unit_price >= 0),
    line_total  NUMERIC(12,2) NOT NULL DEFAULT 0
);
CREATE INDEX idx_order_items_order ON order_items(order_id);

CREATE TABLE payments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id    UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    proof_key   TEXT NOT NULL DEFAULT '',           -- object-storage key for payment proof
    amount      NUMERIC(12,2) NOT NULL DEFAULT 0,
    status      TEXT NOT NULL DEFAULT 'submitted'
                  CHECK (status IN ('submitted','verified','rejected')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_payments_order ON payments(order_id);

-- Parity view: legacy flat order shape (one row per order, first line item).
CREATE VIEW orders_legacy_flat AS
SELECT
    o.id,
    o.tenant_id AS user_id,
    oi.sku,
    oi.quantity,
    oi.unit_price,
    o.total_price,
    o.status,
    o.full_name,
    o.shipping_address->>'mailing_addr'  AS mailing_addr,
    o.shipping_address->>'mailing_addr2' AS mailing_addr2,
    o.shipping_address->>'city'          AS city,
    o.shipping_address->>'postcode'      AS postcode,
    o.shipping_address->>'state'         AS state,
    o.email,
    o.contact_no,
    o.additional_notes,
    o.expired_date,
    o.created_at AS insert_date,
    (SELECT p.proof_key FROM payments p WHERE p.order_id = o.id ORDER BY p.created_at DESC LIMIT 1) AS payment_image_proof
FROM orders o
LEFT JOIN LATERAL (
    SELECT * FROM order_items WHERE order_id = o.id ORDER BY id LIMIT 1
) oi ON TRUE;
