# API — Transactional Core (Go / Echo)

Modular monolith exposing the REST API for admin + storefront (ARCHITECTURE §4).

```bash
go run ./cmd/server        # starts on :8080, self-migrates on boot
go test ./...              # unit tests
go run ./cmd/migratecheck  # apply migrations only (CI gate)
```

Layout: `cmd/` entrypoints · `internal/<domain>` modules (auth, catalog, orders,
customers, coupons, storefront, dashboard, assistant) · `internal/{httpx,db,storage,
events,tenancy,accesslog,notify}` cross-cutting · `internal/db/migrations` SQL ·
`internal/enhancements` post-v1 scaffolds (EH-*, flags default off).

Every repository query is parameterized and tenant-scoped. Config is env-driven
(see [`../.env.example`](../.env.example)).
