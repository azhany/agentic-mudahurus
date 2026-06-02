# Security Review (MH-604) — OWASP-aligned

Reviews the v1 surface against the legacy defects (PRD §1) and OWASP Top 10.
Goal: 0 high/critical findings at launch.

## Legacy defects — remediated
| Legacy issue | v1 remediation |
|---|---|
| Raw string-interpolated SQL (`general_log()`) | All queries use pgx `$1..$n` placeholders; `accesslog` insert is parameterized. No string-built SQL anywhere. |
| Manual `xss_clean` | API is JSON-only; output not HTML-rendered server-side; Vue escapes by default. |
| Legacy `ion_auth` / bcrypt | Argon2id hashing (`auth/argon2.go`); JWT access (15m) + rotating refresh (hash-at-rest). |
| Local-disk uploads, no validation | Storage abstraction with content-type + size limits; signed, expiring URLs; tenant-namespaced keys. |

## OWASP Top 10 pass
| Category | Control |
|---|---|
| A01 Broken Access Control | tenant_id injected server-side from JWT/username, never client; repos require tenant_id; RBAC `RequireRole`; storefront read-only; Qdrant mandatory tenant filter. |
| A02 Cryptographic Failures | Argon2id; HS256 JWT with separate access/refresh secrets; refresh tokens stored as SHA-256 hashes; signed object URLs. |
| A03 Injection | Parameterized SQL throughout; input validation at handler boundary. |
| A04 Insecure Design | Read-only assistant with grounding+refusal; write agents deferred (EH-1) behind flags. |
| A05 Misconfiguration | Secrets via env; consistent JSON error envelope (no stack/SQL leakage); CORS configurable. |
| A06 Vulnerable Components | Pinned deps (`go.mod`, `package.json`, `pyproject.toml`); CI builds on every PR. |
| A07 Auth Failures | Short-lived access tokens; refresh rotation + logout revocation; single-use expiring reset/verify tokens; generic login + forgot-password responses (no user enumeration). |
| A08 Integrity | Deterministic idempotent vector upserts; migrations versioned in `schema_migrations`. |
| A09 Logging | Structured JSON logs with request_id + tenant_id; access logging async/parameterized; Prometheus metrics. |
| A10 SSRF | Assistant proxy targets a fixed configured RAG base URL only; no client-controlled URLs. |

## Multi-tenant isolation (explicit)
- Transactional: every domain table carries `tenant_id`; every repo query filters by it; verified by an end-to-end cross-tenant test (tenant-2 cannot list tenant-1 products).
- RAG: every Qdrant/in-memory search requires `tenant_id`; the store raises if it is missing; storefront scope further restricts to product/category sources.

## Residual / deferred (tracked, not v1-blocking)
- AV scan on uploads — deferred (ARCHITECTURE §9 "AV scan hook").
- PostgreSQL Row-Level Security as defense-in-depth — optional add-on.
- Real SMTP/WhatsApp delivery — EH-3.
