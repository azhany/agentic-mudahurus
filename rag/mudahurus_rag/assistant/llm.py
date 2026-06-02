"""Pluggable LLM provider behind an interface (ARCHITECTURE: swappable).

v1 ships an `echo` provider that synthesizes a grounded answer purely by
extracting from the retrieved chunks — no external calls, fully deterministic,
and impossible to hallucinate beyond context. OpenAI/Anthropic providers can be
slotted in via the same interface for richer synthesis, still constrained to the
provided context by the system prompt.
"""
from __future__ import annotations

from typing import List, Protocol

SYSTEM_PROMPT = (
    "You are MUDAHURUS Assistant. Answer ONLY using the provided context. "
    "If the context does not contain the answer, say you don't have that "
    "information. Never invent products, prices, orders or customer details. "
    "Cite the source ids you used."
)


class LLM(Protocol):
    def synthesize(self, question: str, context_chunks: List[str]) -> str:
        ...


class EchoLLM:
    """Deterministic, grounding-only synthesizer (default v1 provider)."""

    def synthesize(self, question: str, context_chunks: List[str]) -> str:
        if not context_chunks:
            return ""
        joined = " ".join(context_chunks)
        snippet = joined[:600].strip()
        return f"Based on your data: {snippet}"


class OpenAILLM:
    def __init__(self, api_key: str, model: str) -> None:
        self._api_key = api_key
        self._model = model or "gpt-4o-mini"

    def synthesize(self, question: str, context_chunks: List[str]) -> str:
        import httpx  # local import; provider is optional

        context = "\n\n".join(f"[{i}] {c}" for i, c in enumerate(context_chunks))
        resp = httpx.post(
            "https://api.openai.com/v1/chat/completions",
            headers={"Authorization": f"Bearer {self._api_key}"},
            json={
                "model": self._model,
                "messages": [
                    {"role": "system", "content": SYSTEM_PROMPT},
                    {"role": "user", "content": f"Context:\n{context}\n\nQuestion: {question}"},
                ],
                "temperature": 0.0,
            },
            timeout=30,
        )
        resp.raise_for_status()
        return resp.json()["choices"][0]["message"]["content"]


def build_llm(provider: str, api_key: str, model: str) -> LLM:
    if provider == "openai" and api_key:
        return OpenAILLM(api_key, model)
    return EchoLLM()
