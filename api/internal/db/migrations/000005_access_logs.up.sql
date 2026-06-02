-- Parameterized access logging replacing legacy general_log()'s string-interpolated
-- SQL (FR-7.2, MH-405). Inserts are async + parameterized; no interpolation.
CREATE TABLE access_logs (
    id         BIGSERIAL PRIMARY KEY,
    tenant_id  UUID REFERENCES tenants(id) ON DELETE SET NULL,
    ip         INET,
    referrer   TEXT NOT NULL DEFAULT '',
    url        TEXT NOT NULL DEFAULT '',
    uri        TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_access_logs_tenant_time ON access_logs(tenant_id, created_at DESC);
