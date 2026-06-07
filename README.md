# MUDAHURUS 2.0

Modern re-platform of the legacy CodeIgniter 3 / PHP 5.6 **MUDAHURUS.MY** order-management SaaS for Malaysian micro-sellers.

See [`docs/PRD.md`](docs/PRD.md), [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md), [`docs/SPRINT_PLAN.md`](docs/SPRINT_PLAN.md).

## Monorepo layout

| Path | Plane | Stack |
|---|---|---|
| [`api/`](api) | Transactional core | Go 1.22+, Echo v4, pgx, golang-migrate |
| [`web/`](web) | Frontend (admin + storefront) | Vue 3, Vite, Pinia, Tailwind |
| [`rag/`](rag) | RAG plane | Python 3.12, FastAPI, Qdrant, Airflow |
| [`migration/`](migration) | Legacy MySQL → PostgreSQL | Python |
| [`infra/`](infra) | Ops | docker-compose, Prometheus/Grafana, GitHub Actions |
| [`docs/`](docs) | Product/Arch docs | Markdown |

## Quick start (dev)

```bash
cp .env.example .env
cd infra && docker compose up -d        # Postgres, MinIO, Qdrant, API, web, FastAPI
# API health
curl localhost:8080/healthz
```

Run the API standalone:

```bash
cd api && go run ./cmd/server
```

Run the RAG plane:

```bash
cd rag && pip install -e . && uvicorn mudahurus_rag.api.main:app --reload
```

Run the web SPA:

```bash
cd web && npm install && npm run dev
```

## Seed demo data & test accounts

Populate the database with demo accounts and a fully-stocked store (idempotent —
safe to re-run; re-running resets the demo passwords):

```bash
cd api && go run ./cmd/seed          # local: uses DATABASE_URL / .env defaults
# or, with the docker stack running:
cd infra && docker compose run --rm seed
```

This creates the three personas from the PRD (§4):

| Persona | Role | Login | Password | Where |
|---|---|---|---|---|
| **Super Admin** | `operator` | `superadmin` | `superadmin123` | Admin SPA + `/operator/*` routes |
| **Store Owner / Admin** | `seller` | `kedaiali` | `kedaiali123` | Admin SPA — pre-loaded with 5 products, 2 customers, 1 coupon, 2 orders |
| **Store Owner / Admin** | `seller` | `butiksiti` | `butiksiti123` | A second store (proves tenant isolation) |
| **Customer** | _none (public)_ | — | — | Shop **`/store/kedaiali`** and check out as a guest |

> **Customers are not user accounts** in this system — buyers use the public
> storefront (guest checkout + payment-proof upload). To "view as a customer",
> open `/store/kedaiali` in the SPA (no login). Customer *records* (Aminah, Hafiz)
> are seeded under the seller for the admin Customers screen.

> **API base path:** every JSON endpoint is namespaced under **`/api`** so the
> Vue SPA can own the human-facing routes (`/store/{username}`, `/invoice/{id}`,
> `/admin/*`, `/login`). Health/readiness/metrics stay at the root
> (`/healthz`, `/readyz`, `/metrics`). The SPA dev-proxy (Vite) and prod nginx
> forward `/api` → the Go service.

### Test each persona quickly (curl)
```bash
# Super Admin — operator-only route
TOKEN=$(curl -s -X POST localhost:8080/api/auth/login -H 'Content-Type: application/json' \
  -d '{"username":"superadmin","password":"superadmin123"}' | jq -r .access_token)
curl -s localhost:8080/api/operator/tenants/count -H "Authorization: Bearer $TOKEN"

# Store Owner — dashboard KPIs
TOKEN=$(curl -s -X POST localhost:8080/api/auth/login -H 'Content-Type: application/json' \
  -d '{"username":"kedaiali","password":"kedaiali123"}' | jq -r .access_token)
curl -s localhost:8080/api/dashboard/counts -H "Authorization: Bearer $TOKEN"

# Customer — public storefront DATA (the API returns JSON)
curl -s localhost:8080/api/store/kedaiali
```

**To view the storefront as a customer, open the SPA page (not the API):**
run `cd web && npm run dev` and visit **`http://localhost:5173/store/kedaiali`**
in a browser — that renders the Vue storefront page (browse → checkout → invoice).
Admins log in at `http://localhost:5173/login`.

## Roadmap status

This repository implements the committed **v1 scope** (EP-0 … EP-10, Sprint 0 … 6) plus
the post-v1 **Enhancement Backlog** (EH-1 … EH-6), now implemented and mounted under
[`api/internal/enhancements`](api/internal/enhancements) and [`rag/mudahurus_rag/enhancements`](rag/mudahurus_rag/enhancements).
See [`docs/IMPLEMENTATION.md`](docs/IMPLEMENTATION.md) for the story→code build log and
[`docs/ENHANCEMENTS.md`](docs/ENHANCEMENTS.md) for the EH endpoints + feature flags.

## Multi-tenancy

Every transactional query and every vector search is scoped by `tenant_id` (seller `user_id`),
injected server-side from the JWT (API) or resolved from `/store/{username}` (storefront).
Never trusted from the client. See `api/internal/tenancy` and `rag/.../retrieval/filters.py`.
