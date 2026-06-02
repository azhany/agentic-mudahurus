"""Chunking with metadata (MH-505). Word-window chunks with overlap."""
from __future__ import annotations

from dataclasses import dataclass
from typing import List


@dataclass
class Chunk:
    chunk_no: int
    text: str


def chunk_text(text: str, max_words: int = 180, overlap: int = 30) -> List[Chunk]:
    words = text.split()
    if not words:
        return []
    if len(words) <= max_words:
        return [Chunk(0, " ".join(words))]
    chunks: List[Chunk] = []
    start = 0
    n = 0
    step = max(1, max_words - overlap)
    while start < len(words):
        window = words[start:start + max_words]
        chunks.append(Chunk(n, " ".join(window)))
        n += 1
        start += step
    return chunks
