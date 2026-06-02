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

## Roadmap status

This repository implements the committed **v1 scope** (EP-0 … EP-10, Sprint 0 … 6) plus
scaffolds for the post-v1 **Enhancement Backlog** (EH-1 … EH-6) under
[`api/internal/enhancements`](api/internal/enhancements) and [`rag/mudahurus_rag/enhancements`](rag/mudahurus_rag/enhancements).
See [`docs/IMPLEMENTATION.md`](docs/IMPLEMENTATION.md) for the build log mapping each story (MH-/EH-) to code.

## Multi-tenancy

Every transactional query and every vector search is scoped by `tenant_id` (seller `user_id`),
injected server-side from the JWT (API) or resolved from `/store/{username}` (storefront).
Never trusted from the client. See `api/internal/tenancy` and `rag/.../retrieval/filters.py`.
