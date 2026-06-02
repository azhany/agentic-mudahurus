-- Tenants = sellers (and operators). Maps legacy `users.id` -> tenant_id (UUID).
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE tenants (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username        TEXT NOT NULL UNIQUE,
    email           TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    role            TEXT NOT NULL DEFAULT 'seller' CHECK (role IN ('seller','operator')),
    full_name       TEXT NOT NULL DEFAULT '',
    store_name      TEXT NOT NULL DEFAULT '',
    phone           TEXT NOT NULL DEFAULT '',
    email_verified  BOOLEAN NOT NULL DEFAULT FALSE,
    legacy_user_id  BIGINT,                 -- provenance for migration reconciliation
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tenants_legacy_user_id ON tenants(legacy_user_id);

-- Refresh-token store (rotation + logout invalidation, MH-102).
CREATE TABLE refresh_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,       -- store only the SHA-256 hash
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_refresh_tokens_tenant ON refresh_tokens(tenant_id);

-- Single-use, expiring tokens for verify-email / forgot-password (MH-103).
CREATE TABLE auth_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    purpose     TEXT NOT NULL CHECK (purpose IN ('verify_email','reset_password')),
    token_hash  TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_auth_tokens_tenant ON auth_tokens(tenant_id);
