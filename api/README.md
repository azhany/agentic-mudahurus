# API — Transactional Core (Go / Echo)

Modular monolith exposing the REST API for admin + storefront (ARCHITECTURE §4).

```bash
go run ./cmd/server        # starts on :8080, self-migrates on boot
go run ./cmd/seed          # seed demo accounts + data (idempotent)
go test ./...              # unit tests
go run ./cmd/migratecheck  # apply migrations only (CI gate)
```

### Seeded test accounts (`cmd/seed`)
| Role | Login / password |
|---|---|
| operator (super admin) | `superadmin` / `superadmin123` |
| seller (store owner) | `kedaiali` / `kedaiali123`, `butiksiti` / `butiksiti123` |
| customer | no account — shop `/store/kedaiali` as a guest |

Re-running the seeder resets demo passwords and only re-seeds a store's catalog
when it is empty. See the root [README](../README.md#seed-demo-data--test-accounts).

Layout: `cmd/` entrypoints · `internal/<domain>` modules (auth, catalog, orders,
customers, coupons, storefront, dashboard, assistant) · `internal/{httpx,db,storage,
events,tenancy,accesslog,notify}` cross-cutting · `internal/db/migrations` SQL ·
`internal/enhancements` post-v1 scaffolds (EH-*, flags default off).

Every repository query is parameterized and tenant-scoped. Config is env-driven
(see [`../.env.example`](../.env.example)).
