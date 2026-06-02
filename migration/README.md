# Legacy Data Migration (EP-7 / MH-406)

One-time migration of the legacy CodeIgniter MySQL database → PostgreSQL 16,
plus object-storage backfill, per ARCHITECTURE §6.2.

Pipeline: **profile → transform → load → reconcile**

```
mysql (mudahurus_*, users, general_logs)
        │ profile   (types/nulls/orphans)
        ▼ transform (user_id→tenant_id, normalize orders, cleanse types)
postgres (tenants, products, orders, order_items, payments, customers, coupons)
        │ backfill   (legacy upload dirs → MinIO/S3)
        ▼ reconcile  (row counts + checksums + spot checks → report)
```

## Usage

```bash
cd migration
python3 -m pip install -e '.[full]'   # pymysql + psycopg + boto3 (optional extras)

# Dry-run profile only (no writes):
python3 -m mudahurus_migrate profile  --mysql-url mysql://... 

# Full migration:
python3 -m mudahurus_migrate migrate \
  --mysql-url   mysql://user:pass@host/mudahurus \
  --postgres-url postgresql://mudahurus:mudahurus@localhost:5432/mudahurus

# Reconciliation report:
python3 -m mudahurus_migrate reconcile --mysql-url ... --postgres-url ...
```

Field mapping is documented in [`mapping.md`](mapping.md). The migration is
idempotent on re-run (uses `legacy_user_id` / natural keys to upsert).
