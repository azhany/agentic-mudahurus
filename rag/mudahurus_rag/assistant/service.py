"""Assistant service (MH-602, FR-9.5).

Grounding + refusal: synthesizes answers ONLY from retrieved chunks and returns
citations. On empty or low-confidence retrieval it REFUSES ("not found") rather
than free-generating. No tools, no writes.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from typing import List, Optional

from ..retrieval.service import RetrievalService, RetrievedChunk
from .llm import LLM

REFUSAL = (
    "I couldn't find that in your data. I can only answer questions grounded in "
    "your store's catalog and records."
)


@dataclass
class Citation:
    source_type: str
    source_id: str
    score: float


@dataclass
class Answer:
    answer: str
    grounded: bool
    citations: List[Citation] = field(default_factory=list)
    refused: bool = False


class AssistantService:
    def __init__(self, retrieval: RetrievalService, llm: LLM, min_score: float = 0.25) -> None:
        self._retrieval = retrieval
        self._llm = llm
        self._min_score = min_score

    def ask(self, tenant_id: str, question: str, scope: str = "admin",
            top_k: int = 5) -> Answer:
        # Storefront scope restricts retrieval to the product catalog only —
        # customers must never reach orders/customers data (defense in depth).
        source_types: Optional[List[str]] = ["product", "category"] if scope == "storefront" else None

        chunks: List[RetrievedChunk] = self._retrieval.retrieve(
            tenant_id, question, top_k=top_k, source_types=source_types
        )
        grounded = [c for c in chunks if c.score >= self._min_score]
        if not grounded:
            return Answer(answer=REFUSAL, grounded=False, refused=True)

        answer_text = self._llm.synthesize(question, [c.text for c in grounded])
        if not answer_text.strip():
            return Answer(answer=REFUSAL, grounded=False, refused=True)

        citations = [Citation(c.source_type, c.source_id, round(c.score, 4)) for c in grounded]
        return Answer(answer=answer_text, grounded=True, citations=citations)
