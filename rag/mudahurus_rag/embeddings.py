"""Pluggable embedding models (ARCHITECTURE: bge-m3 default, swappable).

In production `sentence-transformers` loads the configured model. When it is not
installed (dev/test/offline), a deterministic hashing embedder is used so the
whole pipeline still runs end-to-end — clearly a fallback, not for production.
"""
from __future__ import annotations

import hashlib
import math
from typing import List, Protocol


class Embedder(Protocol):
    dim: int

    def embed(self, texts: List[str]) -> List[List[float]]:
        ...


class HashingEmbedder:
    """Deterministic, dependency-free embedder for dev/test.

    Maps token hashes into a fixed-dim vector (bag-of-hashed-tokens) and
    L2-normalizes. Not semantically strong, but stable and good enough to
    exercise upsert/search/grounding logic offline.
    """

    def __init__(self, dim: int = 1024) -> None:
        self.dim = dim

    def embed(self, texts: List[str]) -> List[List[float]]:
        out: List[List[float]] = []
        for text in texts:
            vec = [0.0] * self.dim
            for tok in _tokenize(text):
                h = int(hashlib.sha1(tok.encode("utf-8")).hexdigest(), 16)
                idx = h % self.dim
                sign = 1.0 if (h >> 8) & 1 else -1.0
                vec[idx] += sign
            out.append(_l2_normalize(vec))
        return out


class SentenceTransformerEmbedder:
    """Production embedder backed by sentence-transformers."""

    def __init__(self, model_name: str, dim: int) -> None:
        from sentence_transformers import SentenceTransformer  # type: ignore

        self._model = SentenceTransformer(model_name)
        self.dim = self._model.get_sentence_embedding_dimension() or dim

    def embed(self, texts: List[str]) -> List[List[float]]:
        vecs = self._model.encode(texts, normalize_embeddings=True)
        return [list(map(float, v)) for v in vecs]


def build_embedder(model_name: str, dim: int) -> Embedder:
    try:
        return SentenceTransformerEmbedder(model_name, dim)
    except Exception:
        return HashingEmbedder(dim)


def _tokenize(text: str) -> List[str]:
    return [t for t in "".join(c.lower() if c.isalnum() else " " for c in text).split() if t]


def _l2_normalize(vec: List[float]) -> List[float]:
    norm = math.sqrt(sum(v * v for v in vec))
    if norm == 0:
        return vec
    return [v / norm for v in vec]
