# MUDAHURUS 2.0 — Operations Runbook (MH-605)

## Services & ports
| Service | Port | Health |
|---|---|---|
| Go API | 8080 | `/healthz`, `/readyz`, `/metrics` |
| RAG FastAPI | 8000 | `/healthz`, `/readyz`, `/metrics` |
| Web SPA | 5173 (dev) / 80 (prod) | nginx |
| PostgreSQL | 5432 | `pg_isready` |
| MinIO | 9000 / 9001 (console) | `mc ready` |
| Qdrant | 6333 | HTTP |
| Prometheus | 9090 | — |
| Grafana | 3000 | dashboard `mudahurus-overview` |

## Start / stop
```bash
cd infra && docker compose up -d        # start all
docker compose logs -f api              # tail a service
docker compose down                     # stop (keeps volumes)
```

## Migrations
- The API self-migrates on boot (embedded migrations, `schema_migrations` table).
- CI gate: `go run ./cmd/migratecheck` (fails the build on a bad migration).
- Rollback a migration manually with the golang-migrate CLI pointed at
  `api/internal/db/migrations` (down files provided).

## Common incidents
| Symptom | Check | Action |
|---|---|---|
| `/readyz` 500 | DB reachable? | restart postgres; check `DATABASE_URL` |
| Assistant always refuses | Qdrant empty? | run `POST /ingest {tenant_id}`; check embedder |
| Uploads fail | MinIO health, bucket exists | API auto-creates bucket on boot; check creds |
| High API p95 | Grafana latency panel | scale API replicas (stateless) |
| Cross-tenant data fear | n/a | tenancy enforced in middleware + repo + Qdrant filter; run leakage eval |

## Rollback (deploy)
Images are pinned by tag. To roll back: re-point the deployment to the previous
image tag and redeploy. Migrations are additive; a rollback that requires a
schema down-migration is a manual, reviewed operation.

## SLOs (NFR)
- API p95 < 250ms (non-RAG); retrieval p95 < 300ms; storefront LCP < 2.5s.
- Availability 99.5%. API is stateless → horizontal scale behind the LB.

## Load test
```bash
# example using hey/k6 against staging
k6 run infra/loadtest/storefront.js   # asserts p95 thresholds
```
