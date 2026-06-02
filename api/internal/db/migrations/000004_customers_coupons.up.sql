-- Customers (FR-4.x). PII fields (ic_no, dob, contact) flagged; excluded from
-- embeddings by the RAG extractor by default (PRD §6 Privacy).
CREATE TABLE customers (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    full_name              TEXT NOT NULL,
    ic_no                  TEXT NOT NULL DEFAULT '',   -- PII
    dob                    DATE,                        -- PII
    email                  TEXT NOT NULL DEFAULT '',
    contact_no             TEXT NOT NULL DEFAULT '',    -- PII
    mailing_addr           TEXT NOT NULL DEFAULT '',
    city                   TEXT NOT NULL DEFAULT '',
    postcode               TEXT NOT NULL DEFAULT '',
    state                  TEXT NOT NULL DEFAULT '',
    customer_loyalty_code  TEXT NOT NULL DEFAULT '',
    type                   TEXT NOT NULL DEFAULT 'regular',
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, customer_loyalty_code)
);
CREATE INDEX idx_customers_tenant ON customers(tenant_id);

CREATE TABLE coupons (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id   UUID REFERENCES products(id) ON DELETE SET NULL,
    campaign     TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    expired_date TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_coupons_tenant ON coupons(tenant_id);

-- now that customers exist, link orders.customer_id (kept nullable for guest checkout)
ALTER TABLE orders
    ADD CONSTRAINT fk_orders_customer
    FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE SET NULL;
