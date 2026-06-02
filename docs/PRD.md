# MUDAHURUS 2.0 — Product Requirements Document (PRD)

| | |
|---|---|
| **Product** | MUDAHURUS.MY — Online Order Management for micro-sellers |
| **Document** | PRD v1.0 |
| **Status** | Draft for approval |
| **Date** | 2026-05-31 |
| **Owner** | Product / Eng |
| **Supersedes** | Legacy CodeIgniter 3 / PHP 5.6 application (2015) |

---

## 1. Background & Problem

MUDAHURUS.MY ("Sistem Pengurusan Pesanan Online") is a multi-tenant SaaS that lets Malaysian micro-sellers run a simple storefront and manage orders. Each seller gets a public store at `/store/{username}`; customers place orders (guest checkout), upload payment proof, and the seller manages catalog, customers, coupons, and orders from an admin panel.

The current system is built on **CodeIgniter 3 / PHP 5.6** (2015) — both end-of-life and unsupported. Known issues:

- **Security**: raw string-interpolated SQL (e.g. `general_log()`), manual `xss_clean`, legacy `ion_auth`. SQL-injection and session-handling risk.
- **Maintainability**: framework EOL, no dependency management, no automated tests, no CI/CD.
- **Extensibility**: monolithic PHP views (server-rendered + jQuery), no API contract, hard to add modern capabilities (search, AI assistance).
- **Operability**: local-disk file uploads, no observability, manual deploys.

## 2. Goals

| # | Goal | Success Metric |
|---|---|---|
| G1 | Port 100% of existing features to a modern, supported stack with **no functional regression** | Feature parity checklist passes; legacy parity test suite green |
| G2 | Eliminate the known security defects | 0 high/critical findings in security scan; parameterized queries everywhere |
| G3 | Establish a **modern RAG foundation** (ingestion → vector store → retrieval) | Catalog + documents searchable semantically; retrieval API < 300ms p95 |
| G4 | Ship a **read-only RAG assistant** (storefront product Q&A + admin search) | Assistant answers grounded in seller's own data; deflects out-of-scope queries |
| G5 | Make the platform operable and testable | CI/CD green, >70% backend unit coverage on domain logic, structured logs + metrics |

## 3. Non-Goals (v1 Scope Guard)

These are explicitly **out of scope for v1** to prevent scope creep. They are captured in [§10 Enhancement Roadmap](#10-enhancement-roadmap-out-of-scope-for-v1) and are not committed work.

- ❌ Full multi-agent orchestration (router + specialist agents).
- ❌ Autonomous / write-capable agents (auto order-chasing, auto coupon creation).
- ❌ POSLAJU live tracking automation and WhatsApp/email notification agents.
- ❌ Recommendation engine / personalization.
- ❌ New payment-gateway integrations or multi-currency (keep existing manual payment-proof flow).
- ❌ Mobile native apps.
- ❌ Re-platforming the data store beyond the one-time MySQL→PostgreSQL migration.

## 4. Target Users & Personas

| Persona | Description | Primary Needs |
|---|---|---|
| **Seller (Admin)** | Micro-business owner; authenticated tenant | Manage products, orders, customers, coupons; find info fast |
| **Customer (Public)** | Buyer on a seller's storefront; unauthenticated | Browse, search, ask product questions, place order, upload payment proof, view invoice |
| **Platform Operator** | MUDAHURUS staff | Tenant management, observability, support |

## 5. Functional Requirements (Feature Parity Inventory)

Derived from the legacy codebase. **All FRs below are in scope** and must reach parity.

### 5.1 Authentication & Account (replaces `ion_auth`)
- FR-1.1 Seller registration, login, logout.
- FR-1.2 Forgot password / reset via email token.
- FR-1.3 Change password; edit profile.
- FR-1.4 Role-based access (seller vs operator).

### 5.2 Product Catalog (`mudahurus_products`, `..._category`)
- FR-2.1 Product CRUD: `sku, product_name, description, unit_price, category_id, url_slug, image, status (active/inactive)`.
- FR-2.2 Product image upload.
- FR-2.3 Category CRUD (per tenant).
- FR-2.4 Product lookup by SKU.

### 5.3 Orders (`mudahurus_orders`)
- FR-3.1 Guest order placement from storefront with line item + shipping fields (`full_name, mailing_addr[2], city, postcode, state, email, contact_no, quantity, unit_price, total_price, additional_notes`).
- FR-3.2 Order lifecycle: `pending` on creation, **3-day expiry** (`expired_date = now + 3 days`), seller can transition status.
- FR-3.3 Admin order CRUD + listing.
- FR-3.4 Invoice generation (`/invoice/{...}`).
- FR-3.5 Payment-proof upload by customer; payment submission.

### 5.4 Customers (`mudahurus_customers`)
- FR-4.1 Customer CRUD: `full_name, ic_no, dob, email, contact_no, mailing_addr, city, postcode, state, customer_loyalty_code`.
- FR-4.2 Customer lookup by loyalty code.

### 5.5 Coupons (`mudahurus_coupons`)
- FR-5.1 Coupon CRUD: `campaign, description, product_id, expired_date`.

### 5.6 Storefront (Public)
- FR-6.1 Store landing per seller: `/store/{username}` lists active products.
- FR-6.2 Product detail: `/store/{username}/{product_id}`.
- FR-6.3 Search across a seller's catalog.
- FR-6.4 Marketing homepage.

### 5.7 Dashboard & Reporting
- FR-7.1 Seller dashboard with KPI counts (orders, pending, products, customers).
- FR-7.2 Visit/access logging (replaces `general_logs`, parameterized).

### 5.8 API
- FR-8.1 JSON API for all admin list/detail resources (powers the Vue SPA): products, categories, coupons, customers, orders, pending_orders.

### 5.9 RAG Foundation (new, in scope)
- FR-9.1 **Ingestion pipeline** (Python/DAG): extract products, categories, orders, customers, and uploaded documents (payment proofs, invoices) per tenant.
- FR-9.2 **OCR** of uploaded image/PDF documents to text.
- FR-9.3 **Chunk + embed** content; upsert into **Qdrant** with tenant-isolated payload filters.
- FR-9.4 **Retrieval API** (FastAPI) returning grounded, tenant-scoped results.
- FR-9.5 **Read-only assistant**: storefront product Q&A (one seller's catalog) and admin semantic search. No write actions. Must cite/ground answers and refuse out-of-scope queries.

## 6. Non-Functional Requirements

| Category | Requirement |
|---|---|
| **Multi-tenancy** | Every query and every vector search MUST be scoped by `tenant_id` (seller `user_id`). No cross-tenant leakage — enforced in middleware and Qdrant payload filters. |
| **Security** | Parameterized queries only; Argon2id password hashing; JWT auth with short-lived access + refresh; input validation at the boundary; signed URLs for object storage; OWASP Top 10 review. |
| **Performance** | API p95 < 250ms (non-RAG); retrieval p95 < 300ms; storefront LCP < 2.5s. |
| **Availability** | 99.5% target; stateless API for horizontal scaling. |
| **Observability** | Structured JSON logs, request tracing, Prometheus metrics, health/readiness probes. |
| **Privacy** | PII (IC number, DOB, contact) encrypted at rest where feasible; PII excluded from embeddings unless required and access-controlled. |
| **i18n** | Bahasa Melayu primary, English secondary (parity with legacy BM UI). |
| **Testing** | Unit + integration tests in CI; parity acceptance suite vs legacy. |

## 7. Constraints & Assumptions

- One-time data migration MySQL → PostgreSQL; schema redesigned with FKs and `tenant_id`.
- Existing manual payment-proof workflow is retained as-is (no gateway expansion in v1).
- Embedding model and LLM provider are pluggable; default to a self-hostable embedding model to control cost/PII.
- POSLAJU tracking remains **manual** in v1 (display only), automation deferred.

## 8. Release Criteria (Definition of Done for v1)

1. All FR-1…FR-8 at feature parity; parity suite green.
2. FR-9 RAG foundation live; assistant answers grounded & tenant-scoped.
3. 0 high/critical security findings.
4. CI/CD deploying to staging + prod; rollback documented.
5. Observability dashboards live; runbook written.
6. Data migrated and reconciled (row counts + spot checks).

## 9. Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Hidden legacy business rules not in code comments | Regression | Parity test suite; shadow-run against legacy |
| Data quality on migration (loose legacy types) | Migration failure | Profiling + cleansing step; reconciliation reports |
| RAG hallucination / cross-tenant leak | Trust/security | Strict payload filters, grounding + refusal, eval set per tenant |
| Scope creep into agents | Schedule slip | Non-Goals fenced; roadmap items gated behind v1 sign-off |
| OCR quality on payment proofs | Poor retrieval | Confidence thresholds; keep raw doc link as source of truth |

## 10. Enhancement Roadmap (Out of Scope for v1)

> **Scope-guard note:** Items below are intentionally deferred. They build on the v1 RAG foundation and require explicit prioritization + sign-off before any work starts. None are dependencies of v1 release.

- **E1 — Multi-agent orchestration**: router + Seller Copilot (write-capable) + Storefront Assistant + Fulfillment Agent (per prior architecture study).
- **E2 — Autonomous fulfillment**: POSLAJU tracking polling, pending-order auto-chase before 3-day expiry, exception-only human-in-loop.
- **E3 — Notifications**: WhatsApp/email reminders and order updates.
- **E4 — Content generation**: AI product descriptions, SEO, auto-categorization (as a Copilot tool).
- **E5 — Recommendations & analytics**: "why did sales drop", upsell suggestions.
- **E6 — Payments**: real gateway integrations, multi-currency.

Each enhancement, when approved, gets its own PRD slice and is sized independently so it cannot silently expand v1.
