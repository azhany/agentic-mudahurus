# MUDAHURUS 2.0 — Sprint Plan & User Stories

| | |
|---|---|
| **Document** | Sprint Plan v1.0 |
| **Status** | Draft for approval |
| **Date** | 2026-05-31 |
| **Related** | [PRD.md](./PRD.md), [ARCHITECTURE.md](./ARCHITECTURE.md) |
| **Cadence** | 2-week sprints |
| **Estimation** | Story points (Fibonacci); 1 SP ≈ ½ day |

---

## How to read this

- Work is grouped into **Epics** → **Stories**. Each story is self-contained with a user-story statement, acceptance criteria (AC), and dependencies.
- **Sprint 0–6** deliver the committed v1 scope (port + RAG foundation + read-only assistant).
- The **Enhancement Backlog** at the end is explicitly **not committed** — it exists so good ideas are captured without leaking into v1.

### Epic Index

| Epic | Title | Sprints |
|---|---|---|
| EP-0 | Foundations & Tooling | S0 |
| EP-1 | Auth & Tenancy | S1 |
| EP-2 | Catalog (Products & Categories) | S1–S2 |
| EP-3 | Orders, Payments & Invoicing | S2–S3 |
| EP-4 | Customers & Coupons | S3 |
| EP-5 | Public Storefront | S3–S4 |
| EP-6 | Vue Frontend (Admin + Store) | S2–S4 |
| EP-7 | Data Migration | S4 |
| EP-8 | RAG Ingestion Pipeline | S5 |
| EP-9 | Retrieval & Read-Only Assistant | S6 |
| EP-10 | Hardening, Observability & Launch | S6 |

---

## Sprint 0 — Foundations & Tooling

**Goal:** A deployable skeleton with CI/CD, infra, and conventions so every later story plugs in cleanly.

### MH-001 — Repository & monorepo structure
**As a** developer, **I want** an agreed repo layout (`/api` Go, `/web` Vue, `/rag` Python, `/infra`, `/docs`) **so that** teams work without collisions.
**AC:**
- Monorepo created with the four workspaces and READMEs.
- Code owners and branch protection configured.
- Conventional commits + lint hooks enabled.
**Points:** 3 · **Deps:** —

### MH-002 — Local dev environment (docker-compose)
**As a** developer, **I want** one command to run Postgres, MinIO, Qdrant, API, web, and FastAPI locally **so that** onboarding is minutes not days.
**AC:**
- `docker-compose up` brings up all services with seed config.
- `.env.example` documents every variable.
- Health endpoints reachable.
**Points:** 5 · **Deps:** MH-001

### MH-003 — CI/CD pipeline
**As a** team, **I want** GitHub Actions running lint/test/build for all workspaces and deploying to staging **so that** main is always shippable.
**AC:**
- PR pipeline: lint + unit tests + build for Go/Vue/Python.
- Migration dry-run check gate.
- Auto-deploy to staging on merge to main; manual prod promote.
**Points:** 5 · **Deps:** MH-002

### MH-004 — Echo API skeleton + middleware
**As a** developer, **I want** an Echo server with request-id, structured logging, error envelope, and OTel **so that** all handlers share conventions.
**AC:**
- `/healthz` and `/readyz` implemented.
- JSON error envelope + panic recovery.
- Structured logs include `request_id` (and `tenant_id` once auth lands).
**Points:** 5 · **Deps:** MH-002

### MH-005 — DB migration tooling + base schema
**As a** developer, **I want** `golang-migrate` + `sqlc` wired with a `tenants` table **so that** schema changes are versioned and queries are type-safe.
**AC:**
- Up/down migrations run in CI and locally.
- `sqlc generate` produces typed Go from SQL.
- `tenants` table created.
**Points:** 3 · **Deps:** MH-004

---

## Sprint 1 — Auth & Tenancy + Catalog (Backend)

**Goal:** Secure, tenant-scoped foundation and the first domain (products/categories).

### MH-101 — Password hashing & user model
**As a** seller, **I want** my credentials stored with Argon2id **so that** my account is secure (replacing legacy `ion_auth`/bcrypt).
**AC:**
- Argon2id hashing with sane params.
- User/tenant record with role (`seller`/`operator`).
- Unit tests for hash/verify.
**Points:** 3 · **Deps:** MH-005

### MH-102 — JWT auth (login/refresh/logout)
**As a** seller, **I want** to log in and receive access+refresh tokens **so that** I can use the admin securely.
**AC:**
- Login issues 15m access + 7d refresh; refresh rotation; logout invalidates refresh.
- Tokens carry `tenant_id` + role.
- Negative tests (bad creds, expired token).
**Points:** 5 · **Deps:** MH-101

### MH-103 — Registration, forgot/reset, change password
**As a** new seller, **I want** to register and recover my password **so that** I can self-serve (parity with FR-1.1–1.3).
**AC:**
- Register creates tenant; email verification token.
- Forgot→email token→reset flow; change password when authed.
- Tokens single-use + expiring.
**Points:** 5 · **Deps:** MH-102

### MH-104 — Tenant + RBAC middleware
**As a** platform, **I want** middleware that injects `tenant_id` into request context and enforces roles **so that** no handler can skip tenancy.
**AC:**
- Middleware extracts `tenant_id` from JWT; rejects if missing.
- Role guard for admin routes.
- Repository layer requires `tenant_id` (lint/test enforced).
**Points:** 5 · **Deps:** MH-102

### MH-105 — Categories CRUD API
**As a** seller, **I want** to manage product categories **so that** I can organize my catalog (FR-2.3).
**AC:**
- CRUD endpoints, tenant-scoped, validated.
- List endpoint powers SPA dropdowns.
- Tests incl. cross-tenant access denial.
**Points:** 3 · **Deps:** MH-104

### MH-106 — Products CRUD API
**As a** seller, **I want** to manage products **so that** I can sell them (FR-2.1, FR-2.4).
**AC:**
- CRUD for `sku, product_name, description, unit_price, category_id, url_slug, status`.
- Lookup by SKU; unique `(tenant_id, sku)`.
- Pagination + filter; tenant-scoped tests.
**Points:** 5 · **Deps:** MH-105

---

## Sprint 2 — Catalog Media, Orders Core + Admin SPA Shell

**Goal:** Finish catalog with images, start orders, stand up the Vue admin shell.

### MH-201 — Object storage service + signed URLs
**As a** developer, **I want** a storage abstraction over MinIO/S3 with signed URLs **so that** uploads/downloads are secure and swappable.
**AC:**
- Put/get with content-type + size limits.
- Time-limited signed URLs.
- Keys namespaced by `tenant_id`.
**Points:** 5 · **Deps:** MH-104

### MH-202 — Product image upload
**As a** seller, **I want** to upload a product image **so that** my storefront looks complete (FR-2.2).
**AC:**
- Upload returns `image_key`; product references it.
- Old image cleaned up on replace.
- Validation (type/size) + tests.
**Points:** 3 · **Deps:** MH-201, MH-106

### MH-203 — Orders domain & schema (normalized)
**As a** seller, **I want** orders modeled as `orders` + `order_items` + `payments` **so that** data is clean while preserving legacy fields.
**AC:**
- Migrations for the three tables with FKs.
- Parity view reproduces legacy flat shape.
- Repository methods tenant-scoped.
**Points:** 5 · **Deps:** MH-104

### MH-204 — Admin order CRUD + listing API
**As a** seller, **I want** to view/create/edit orders and statuses **so that** I can manage fulfillment (FR-3.3).
**AC:**
- CRUD + status transitions (`pending`→…); 3-day `expired_date` set on create.
- List with filters (status, date, pending) — powers `api/orders`, `api/pending_orders`.
- Tests for lifecycle + expiry.
**Points:** 5 · **Deps:** MH-203

### MH-205 — Vue admin app shell
**As a** seller, **I want** an admin SPA with auth, nav, and layout **so that** I have a home for all admin features (FR-8.1).
**AC:**
- Vite + Vue 3 + Pinia + Router + Tailwind scaffold.
- Login flow against MH-102; token storage + refresh interceptor.
- Sidebar nav (Dashboard, Products, Orders, Customers, Coupons) + BM/EN i18n scaffold.
**Points:** 5 · **Deps:** MH-102

### MH-206 — Admin: Products & Categories screens
**As a** seller, **I want** product/category management UIs **so that** I can run my catalog from the browser.
**AC:**
- Product list (paginated, searchable) + create/edit form with image upload.
- Category management UI.
- Wired to MH-105/106/202.
**Points:** 5 · **Deps:** MH-205, MH-106, MH-202

---

## Sprint 3 — Payments/Invoicing, Customers, Coupons + Admin UIs

**Goal:** Complete the order money-flow and the remaining admin domains.

### MH-301 — Invoice generation
**As a** customer, **I want** an invoice for my order **so that** I have a payment reference (FR-3.4).
**AC:**
- Invoice retrievable by token/order ref (`/invoice/{...}`).
- Renders line items, totals, seller + shipping info.
- PDF export.
**Points:** 5 · **Deps:** MH-204

### MH-302 — Payment-proof upload & submission
**As a** customer, **I want** to upload payment proof **so that** the seller can confirm my order (FR-3.5).
**AC:**
- Upload tied to order → `payments.proof_key`; status `submitted`.
- Seller can mark verified/rejected.
- Signed URLs; validation; tests.
**Points:** 5 · **Deps:** MH-301, MH-201

### MH-303 — Customers CRUD API + loyalty lookup
**As a** seller, **I want** to manage customers and look them up by loyalty code **so that** I can track buyers (FR-4.1, FR-4.2).
**AC:**
- CRUD for all customer fields; PII fields flagged.
- Lookup by `customer_loyalty_code` (tenant-scoped).
- Tests incl. cross-tenant denial.
**Points:** 5 · **Deps:** MH-104

### MH-304 — Coupons CRUD API
**As a** seller, **I want** to manage coupons **so that** I can run campaigns (FR-5.1).
**AC:**
- CRUD for `campaign, description, product_id, expired_date`.
- Expiry handling; tenant-scoped.
- Tests.
**Points:** 3 · **Deps:** MH-104

### MH-305 — Admin: Orders, Customers, Coupons, Dashboard screens
**As a** seller, **I want** UIs for orders, customers, coupons and a KPI dashboard **so that** I manage everything in one place (FR-3.3, FR-4.1, FR-5.1, FR-7.1).
**AC:**
- Order list + detail (status, payment proof viewer), customer + coupon CRUD UIs.
- Dashboard KPI cards (orders, pending, products, customers) from counts API.
- Wired to MH-204/302/303/304.
**Points:** 8 · **Deps:** MH-205, MH-204, MH-302, MH-303, MH-304

---

## Sprint 4 — Public Storefront + Data Migration

**Goal:** Ship the customer-facing store and migrate legacy data.

### MH-401 — Storefront listing & product detail API
**As a** customer, **I want** to view a seller's store and products **so that** I can shop (FR-6.1, FR-6.2).
**AC:**
- Public, read-only endpoints resolve tenant from `username`; only `active` products.
- Product detail by id/slug.
- No auth required; rate-limited.
**Points:** 5 · **Deps:** MH-106

### MH-402 — Storefront search
**As a** customer, **I want** to search a seller's catalog **so that** I find products fast (FR-6.3).
**AC:**
- Keyword search (Postgres FTS) scoped to tenant + active.
- Pagination; empty-state.
- Tests.
**Points:** 3 · **Deps:** MH-401

### MH-403 — Guest checkout / order placement
**As a** customer, **I want** to place an order without an account **so that** buying is frictionless (FR-3.1, FR-3.2).
**AC:**
- Public submit endpoint with full shipping payload + validation.
- Creates order (`pending`, +3-day expiry) + item.
- Anti-abuse (rate limit, basic bot guard).
**Points:** 5 · **Deps:** MH-203, MH-401

### MH-404 — Storefront Vue app (browse, search, checkout, invoice)
**As a** customer, **I want** a clean storefront UI **so that** I can browse, order, pay, and view my invoice (FR-6.x).
**AC:**
- Store landing, product grid, detail, search, checkout form, payment-proof upload, invoice view.
- BM-first i18n; responsive; LCP < 2.5s.
- Wired to MH-401/402/403/301/302.
**Points:** 8 · **Deps:** MH-401, MH-402, MH-403, MH-301, MH-302

### MH-405 — Access logging (parameterized)
**As a** platform, **I want** safe visit logging **so that** we keep analytics without SQLi (FR-7.2, fixes legacy `general_log`).
**AC:**
- Async, parameterized insert of ip/referrer/url/uri/time.
- No raw string interpolation; load-tested.
**Points:** 2 · **Deps:** MH-004

### MH-406 — Legacy data migration & reconciliation
**As a** platform, **I want** MySQL data migrated to PostgreSQL + object storage **so that** v1 launches with real data.
**AC:**
- Profiling → transform (`user_id`→`tenant_id`, normalize orders) → load.
- Upload files backfilled to object storage.
- Reconciliation report (counts/checksums/spot checks) approved.
**Points:** 8 · **Deps:** MH-203, MH-303, MH-304, MH-201

---

## Sprint 5 — RAG Ingestion Pipeline

**Goal:** Stand up the Python data plane: extract → OCR → embed → Qdrant.

### MH-501 — Qdrant setup & tenant-isolated collections
**As a** data engineer, **I want** Qdrant provisioned with payload schema and mandatory tenant filters **so that** retrieval is isolated and fast.
**AC:**
- Collection(s) with payload `tenant_id, source_type, source_id, chunk_no`.
- Index config tuned; helper enforces `tenant_id` filter on every query.
- Smoke test upsert/search.
**Points:** 5 · **Deps:** MH-002

### MH-502 — Airflow deployment & DAG scaffold
**As a** data engineer, **I want** Airflow running with a parameterized ingestion DAG skeleton **so that** runs are scheduled, observable, and backfillable.
**AC:**
- Airflow up in dev + staging; DAG with extract→ocr→chunk→embed→upsert tasks (stubbed).
- Per-tenant run params; retries + alerting.
**Points:** 5 · **Deps:** MH-002

### MH-503 — Structured extractor (products/orders/customers)
**As a** data engineer, **I want** to extract and normalize structured records to text **so that** they can be embedded (FR-9.1).
**AC:**
- Pulls per-tenant records from Postgres; normalizes to text docs.
- PII policy applied (IC/DOB excluded by default).
- Incremental (changed-since) support.
**Points:** 5 · **Deps:** MH-502, MH-406

### MH-504 — OCR worker for uploaded documents
**As a** data engineer, **I want** OCR on payment proofs/invoices **so that** their text is searchable (FR-9.2).
**AC:**
- PaddleOCR/Tesseract extracts text from images/PDFs in object storage.
- Confidence threshold; low-confidence flagged, raw doc link kept as source.
- Unit tests on sample docs.
**Points:** 5 · **Deps:** MH-502, MH-201

### MH-505 — Chunk + embed + upsert
**As a** data engineer, **I want** to chunk, embed, and upsert content idempotently **so that** Qdrant stays consistent (FR-9.3).
**AC:**
- Deterministic point IDs `(tenant_id, source_type, source_id, chunk_no)`; re-run upserts, no dupes.
- Pluggable embedding model (`bge-m3` default).
- Index health check task + basic eval.
**Points:** 5 · **Deps:** MH-501, MH-503, MH-504

### MH-506 — Event-driven ingestion triggers
**As a** platform, **I want** ingestion to fire on create/upload events **so that** the index is fresh **without** waiting for the schedule.
**AC:**
- Go API emits events on product/order create + payment-proof upload.
- Airflow API trigger (or queue) consumes them; deduped.
- End-to-end test: create product → searchable within SLA.
**Points:** 5 · **Deps:** MH-505, MH-106, MH-302

---

## Sprint 6 — Retrieval, Read-Only Assistant & Launch

**Goal:** Grounded assistant for storefront + admin, then hardening and go-live.

### MH-601 — FastAPI retrieval service
**As a** developer, **I want** a retrieval API that returns grounded, tenant-scoped chunks **so that** clients get trustworthy context (FR-9.4).
**AC:**
- `POST /retrieve` with server-injected `tenant_id` filter; top-k + scores + source refs.
- p95 < 300ms on staging data.
- Rejects/ignores any client-supplied tenant override.
**Points:** 5 · **Deps:** MH-505

### MH-602 — Read-only assistant (grounding + refusal)
**As a** user, **I want** an assistant that answers from my data and refuses otherwise **so that** I can trust it (FR-9.5).
**AC:**
- Synthesizes answers **only** from retrieved chunks; returns citations.
- Refuses / says "not found" on empty or low-confidence retrieval (no free-gen).
- No tools, no writes; per-tenant eval set passes leakage + grounding checks.
**Points:** 8 · **Deps:** MH-601

### MH-603 — Assistant proxy in Go + UI integration
**As a** user, **I want** a chat panel in storefront and admin **so that** I can ask questions naturally.
**AC:**
- Go proxy resolves `tenant_id` server-side, forwards to assistant, streams response.
- Storefront: product Q&A widget (one seller's catalog). Admin: semantic search box.
- Loading/refusal/citation states handled in Vue.
**Points:** 5 · **Deps:** MH-602, MH-404, MH-305

### MH-604 — Security review & pen-test fixes
**As a** platform, **I want** an OWASP-aligned review **so that** we launch with 0 high/critical findings (NFR security).
**AC:**
- Authn/z, tenancy isolation, file handling, SQLi/XSS reviewed.
- Findings triaged; high/critical fixed and re-tested.
**Points:** 5 · **Deps:** all feature stories

### MH-605 — Observability, runbook & load test
**As an** operator, **I want** dashboards, alerts, and a runbook **so that** I can run v1 in prod (NFR observability/availability).
**AC:**
- Prometheus/Grafana dashboards (API, retrieval, ingestion); alerts wired.
- Load test meets p95 targets; runbook + rollback documented.
**Points:** 5 · **Deps:** MH-004, MH-601

### MH-606 — Parity acceptance & production cutover
**As a** stakeholder, **I want** verified feature parity and a clean cutover **so that** we retire the legacy app safely (Release Criteria).
**AC:**
- Parity checklist (FR-1…FR-8) green; shadow comparison spot-checked.
- Final delta migration; DNS cutover; legacy set read-only then archived.
- Post-launch smoke + on-call coverage.
**Points:** 8 · **Deps:** MH-406, MH-604, MH-605

---

## Velocity & Timeline (indicative)

| Sprint | Theme | Points |
|---|---|---|
| S0 | Foundations | 21 |
| S1 | Auth + Catalog backend | 26 |
| S2 | Media, Orders core, Admin shell | 28 |
| S3 | Payments, Customers, Coupons, Admin UIs | 26 |
| S4 | Storefront + Migration | 31 |
| S5 | RAG ingestion | 30 |
| S6 | Assistant + Launch | 36 |

> ~12 weeks at 2-week cadence assuming a small cross-functional team. Adjust to actual velocity after S0–S1.

---

## Enhancement Backlog (NOT in v1 — requires separate sign-off)

> **Scope guard:** these mirror PRD §10. They are parked here so the team stops adding "just one more thing" to v1. Each becomes its own epic with its own PRD slice and estimate only after v1 ships and it's prioritized.

| ID | Enhancement | Builds on |
|---|---|---|
| EH-1 | Multi-agent orchestration: router + Seller Copilot (write) + Storefront Assistant + Fulfillment Agent | RAG plane, retrieval API |
| EH-2 | Autonomous fulfillment: POSLAJU polling + pending-order auto-chase before 3-day expiry | Orders, events |
| EH-3 | Notifications: WhatsApp/email order updates & reminders | Orders, events |
| EH-4 | AI content: product descriptions, SEO, auto-categorization (Copilot tool) | Catalog, LLM |
| EH-5 | Recommendations & analytics ("why did sales drop", upsell) | Orders, RAG |
| EH-6 | Payment gateway integrations + multi-currency | Payments |

**Definition of "not scope creep":** a backlog item enters a sprint only when (a) v1 is released, (b) it has an approved PRD slice, and (c) it's independently estimated. Until all three hold, it stays in this table.
