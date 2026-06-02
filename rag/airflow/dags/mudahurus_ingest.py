"""MUDAHURUS ingestion DAG (MH-502).

Parameterized, per-tenant ingestion: extract → ocr → chunk → embed → upsert →
index-health-check (ARCHITECTURE §7). Scheduled hourly and triggerable via the
Airflow REST API for event-driven freshness (MH-506).

If Airflow is not installed (the package's `airflow` extra), importing this file
is a no-op so the rest of the repo still imports cleanly.
"""
from __future__ import annotations

import os

try:
    from datetime import datetime, timedelta

    from airflow import DAG  # type: ignore
    from airflow.operators.python import PythonOperator  # type: ignore

    _AIRFLOW = True
except Exception:  # pragma: no cover - airflow optional
    _AIRFLOW = False


def _run_ingest(**context):
    """Task callable: ingest one tenant. tenant_id comes from dag_run.conf
    (event trigger) or params (scheduled backfill over all tenants)."""
    from mudahurus_rag.config import get_settings
    from mudahurus_rag.engine import get_engine
    from mudahurus_rag.ingestion.extract import list_tenant_ids
    from mudahurus_rag.ingestion.pipeline import ingest_tenant

    settings = get_settings()
    eng = get_engine()
    conf = (context.get("dag_run").conf or {}) if context.get("dag_run") else {}
    changed_since = conf.get("changed_since")

    tenant_ids = [conf["tenant_id"]] if conf.get("tenant_id") else list_tenant_ids(settings.rag_database_url)
    reports = []
    for tid in tenant_ids:
        reports.append(ingest_tenant(settings.rag_database_url, tid, eng.embedder, eng.store, changed_since).__dict__)
    return reports


if _AIRFLOW:
    default_args = {
        "owner": "data-eng",
        "retries": 2,
        "retry_delay": timedelta(minutes=5),
    }

    with DAG(
        dag_id="mudahurus_ingest",
        description="Per-tenant RAG ingestion (extract→ocr→chunk→embed→upsert)",
        default_args=default_args,
        schedule_interval="@hourly",
        start_date=datetime(2026, 1, 1),
        catchup=False,
        max_active_runs=1,
        tags=["mudahurus", "rag"],
    ) as dag:
        PythonOperator(
            task_id="ingest",
            python_callable=_run_ingest,
        )
else:
    dag = None
    if os.environ.get("MUDAHURUS_DEBUG"):
        print("airflow not installed; DAG definition skipped")
