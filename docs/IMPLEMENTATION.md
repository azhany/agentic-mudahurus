# MUDAHURUS 2.0 — Implementation Log

Maps every epic/story (EP-/MH-) and backlog item (EH-) to the code that
implements it. Status legend: ✅ implemented & tested · 🟡 scaffold (post-v1).

## Sprint 0 — Foundations (EP-0)
| Story | Status | Where |
|---|---|---|
| MH-001 Repo/monorepo structure | ✅ | `api/ web/ rag/ migration/ infra/ docs/` + workspace READMEs |
| MH-002 Local dev (docker-compose) | ✅ | [`infra/docker-compose.yml`](../infra/docker-compose.yml), [`.env.example`](../.env.example) |
| MH-003 CI/CD pipeline | ✅ | [`.github/workflows/ci.yml`](../.github/workflows/ci.yml) (lint+test+build+migrate-check, staging deploy) |
| MH-004 Echo skeleton + middleware | ✅ | `api/internal/httpx` (envelope, request-id, access log, metrics), `cmd/server` (`/healthz`,`/readyz`,`/metrics`) |
| MH-005 Migrations + base schema | ✅ | `api/internal/db/migrate.go`, `api/internal/db/migrations/000001_tenants*` |

## Sprint 1 — Auth & Tenancy + Catalog (EP-1, EP-2)
| MH-101 Argon2id + user model | ✅ | `api/internal/auth/argon2.go` (+test) |
| MH-102 JWT login/refresh/logout | ✅ | `auth/jwt.go`, `auth/service.go` (refresh rotation, logout revoke) (+test) |
| MH-103 Register/forgot/reset/change | ✅ | `auth/service.go`, `auth/handler.go` (single-use expiring tokens) |
| MH-104 Tenant + RBAC middleware | ✅ | `auth/middleware.go`, `tenancy/tenancy.go`, repos require tenant_id |
| MH-105 Categories CRUD | ✅ | `catalog/` (tenant-scoped, validated) |
| MH-106 Products CRUD | ✅ | `catalog/` (unique (tenant,sku), pagination, SKU lookup) |

## Sprint 2 — Media, Orders core, Admin shell (EP-3, EP-6)
| MH-201 Object storage + signed URLs | ✅ | `storage/` (local + MinIO/S3 backends, HMAC-signed local URLs) |
| MH-202 Product image upload | ✅ | `catalog/handler.go` uploadImage (type/size validation, old-image cleanup) |
| MH-203 Orders normalized schema | ✅ | `db/migrations/000003_orders*` (orders+order_items+payments + `orders_legacy_flat` view) |
| MH-204 Admin order CRUD + listing | ✅ | `orders/` (status transitions, 3-day expiry, `/orders`,`/pending_orders`) |
| MH-205 Vue admin shell | ✅ | `web/src` (Vite+Vue3+Pinia+Router+Tailwind, refresh interceptor, BM/EN i18n) |
| MH-206 Products & Categories screens | ✅ | `web/src/views/Products.vue`, `Categories.vue` |

## Sprint 3 — Payments/Invoicing, Customers, Coupons (EP-3, EP-4, EP-6)
| MH-301 Invoice generation | ✅ | `orders/invoice.go` (self-contained PDF) + `/invoice/:id`, `/orders/:id/invoice.pdf` |
| MH-302 Payment-proof upload | ✅ | `orders/handler.go` PublicUploadPayment + VerifyPayment |
| MH-303 Customers CRUD + loyalty | ✅ | `customers/` (PII flagged, loyalty lookup) |
| MH-304 Coupons CRUD | ✅ | `coupons/` |
| MH-305 Admin screens | ✅ | `web/src/views/Orders.vue, Customers.vue, Coupons.vue, Dashboard.vue` |

## Sprint 4 — Storefront + Migration (EP-5, EP-7)
| MH-401 Storefront listing/detail | ✅ | `storefront/` (public, tenant-by-username, active-only) |
| MH-402 Storefront search | ✅ | `storefront` FTS (`000006_product_fts`), tenant+active scoped |
| MH-403 Guest checkout | ✅ | `storefront.checkout` → `orders.GuestCheckout` (rate-limited) |
| MH-404 Storefront Vue app | ✅ | `web/src/views/Storefront.vue, ProductDetail.vue, Invoice.vue` |
| MH-405 Access logging (parameterized) | ✅ | `accesslog/` (async, parameterized, `000005_access_logs`) |
| MH-406 Legacy data migration | ✅ | `migration/` (profile→transform→load→reconcile) (+transform tests) |

## Sprint 5 — RAG Ingestion (EP-8)
| MH-501 Qdrant + tenant filters | ✅ | `rag/.../vectorstore.py` (Qdrant + in-mem fallback, mandatory tenant filter) |
| MH-502 Airflow DAG scaffold | ✅ | `rag/airflow/dags/mudahurus_ingest.py` |
| MH-503 Structured extractor | ✅ | `rag/.../ingestion/extract.py` (PII excluded, incremental) |
| MH-504 OCR worker | ✅ | `rag/.../ingestion/ocr.py` (tesseract + confidence threshold + fallback) |
| MH-505 Chunk + embed + upsert | ✅ | `rag/.../chunking.py`, `embeddings.py`, `ingestion/pipeline.py` (deterministic ids) |
| MH-506 Event-driven triggers | ✅ | `api/internal/events` emits on product/order create + payment upload → `/ingest` |

## Sprint 6 — Assistant & Launch (EP-9, EP-10)
| MH-601 FastAPI retrieval | ✅ | `rag/.../retrieval/service.py`, `api/main.py` `/retrieve` |
| MH-602 Read-only assistant | ✅ | `rag/.../assistant/service.py` (grounding + refusal) (+tests) |
| MH-603 Assistant proxy + UI | ✅ | `api/internal/assistant` (server-injected tenant), `web` AssistantPanel + store ask widget |
| MH-604 Security review | ✅ | [`SECURITY_REVIEW.md`](./SECURITY_REVIEW.md) |
| MH-605 Observability + runbook | ✅ | Prometheus/Grafana in `infra/`, [`RUNBOOK.md`](./RUNBOOK.md) |
| MH-606 Parity acceptance + cutover | ✅ | [`PARITY_CHECKLIST.md`](./PARITY_CHECKLIST.md) |

## Enhancement Backlog (EH-1 … EH-6) — POST-V1 scaffolds (flags default OFF)
| EH-1 Multi-agent orchestration | 🟡 | `api/internal/enhancements/eh1_orchestration.go`, `rag/.../enhancements/orchestration.py` |
| EH-2 Autonomous fulfillment | 🟡 | `enhancements/eh2_fulfillment.go` (auto-chase decision, tracking provider) |
| EH-3 Notifications | 🟡 | `enhancements/eh3_notifications.go` (WhatsApp/email channels, BM templates) |
| EH-4 AI content | 🟡 | `enhancements/eh4_ai_content.go` (descriptions/SEO/auto-categorize) |
| EH-5 Recommendations & analytics | 🟡 | `enhancements/eh5_recommendations.go` (sales-drop insight, upsell iface) |
| EH-6 Payments + multi-currency | 🟡 | `enhancements/eh6_payments.go` (gateway iface, manual gateway = v1) |

All EH items are gated by `enhancements.Enabled()` / `MH_*` env flags, default OFF,
and are NOT mounted by `server.Mount` — protecting v1 scope per PRD §10 / SPRINT_PLAN.

## Verification done in this build
- `go build ./...`, `go vet ./...`, `go test ./...` green.
- RAG `pytest` (8) + migration `pytest` (5) green.
- `npm run build` (Vue SPA) succeeds.
- End-to-end against real Postgres 16: migrations apply; register→login→catalog→
  storefront FTS→guest checkout (3-day expiry)→invoice JSON+PDF; tenant isolation
  verified (cross-tenant list empty); `orders_legacy_flat` parity view returns the
  flat legacy shape.
