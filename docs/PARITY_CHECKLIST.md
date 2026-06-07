# Feature Parity Checklist (MH-606) — Legacy → v1

Verifies FR-1 … FR-8 reach parity with the legacy CodeIgniter app, plus the new
FR-9 RAG foundation. ✅ = implemented & exercised.

> JSON API endpoints below are served under the **`/api`** namespace (e.g.
> `/api/auth/login`); the bare paths (`/store/{username}`, `/invoice/{id}`,
> `/login`, `/admin/*`) are the Vue SPA's human-facing pages.

| FR | Legacy behaviour | v1 endpoint(s) | Status |
|---|---|---|---|
| FR-1.1 register/login/logout | `admin/auth/*` | `POST /auth/register`,`/auth/login`,`/auth/logout` | ✅ |
| FR-1.2 forgot/reset | `forgot`,`reset/:any` | `POST /auth/forgot-password`,`/auth/reset-password` | ✅ |
| FR-1.3 change pw / profile | `change`,`profile` | `POST /auth/change-password`, `PUT /me` | ✅ |
| FR-1.4 RBAC | ion_auth groups | JWT role + `RequireRole`, `/operator/*` | ✅ |
| FR-2.1 product CRUD | `admin/products` | `/products` CRUD | ✅ |
| FR-2.2 product image | `Upload.php` | `POST /products/:id/image` | ✅ |
| FR-2.3 category CRUD | `admin/categories` | `/categories` CRUD | ✅ |
| FR-2.4 lookup by SKU | `get_product_by_sku` | `GET /products/by-sku/:sku` | ✅ |
| FR-3.1 guest order | `store/submit` | `POST /store/:username/checkout` | ✅ |
| FR-3.2 pending + 3-day expiry | `expired_date=+3d` | order create sets `expired_date=now+72h` | ✅ |
| FR-3.3 admin order CRUD | `admin/orders` | `/orders` CRUD + `/orders/:id/status` | ✅ |
| FR-3.4 invoice | `invoice/:any` | `/invoice/:id` (JSON + `?format=pdf`) | ✅ |
| FR-3.5 payment proof | `store/upload_payment` | `POST /orders/:id/payment`, `PATCH /payments/:id/verify` | ✅ |
| FR-4.1 customer CRUD | `admin/customers` | `/customers` CRUD | ✅ |
| FR-4.2 loyalty lookup | `get_customer_by_code` | `GET /customers/by-loyalty/:code` | ✅ |
| FR-5.1 coupon CRUD | `admin/coupons` | `/coupons` CRUD | ✅ |
| FR-6.1 store landing | `store/:any` | `GET /store/:username` | ✅ |
| FR-6.2 product detail | `store/:any/:any` | `GET /store/:username/products/:id` | ✅ |
| FR-6.3 search | `search_v` | `GET /store/:username/search` (FTS) | ✅ |
| FR-6.4 marketing homepage | `home` | Vue storefront landing | ✅ |
| FR-7.1 dashboard KPIs | `M_counts` | `GET /dashboard/counts` | ✅ |
| FR-7.2 visit logging | `general_logs` | `accesslog` (parameterized) | ✅ |
| FR-8.1 admin JSON API | `admin/api/*` | all `/products`,`/category`,`/coupons`,`/customers`,`/orders`,`/pending_orders` (+legacy aliases) | ✅ |
| FR-9.1–9.5 RAG foundation | — (new) | `rag/` ingestion + `/retrieve` + `/assistant/ask` + proxy | ✅ |

## Cutover plan (from ARCHITECTURE §6.2)
1. Read-only freeze on legacy.
2. Final delta migration (`migration migrate` + `reconcile` → report approved).
3. DNS cutover to the Vue SPA + Go API.
4. Legacy app archived read-only; post-launch smoke (this checklist) + on-call.
