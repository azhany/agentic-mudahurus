"""POST-V1 enhancement scaffolds for the RAG plane (EH-1 … EH-6, PRD §10).

Disabled by default and NOT mounted by the v1 FastAPI app. Each builds on the v1
foundation (retrieval, assistant) and can be enabled independently once it has an
approved PRD slice. Mirrors api/internal/enhancements on the Go side.
"""
import os


def enabled(flag: str) -> bool:
    return os.getenv("MH_" + flag, "").lower() == "true"
