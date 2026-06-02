"""OCR worker for uploaded documents (MH-504, FR-9.2).

Tries pytesseract (and pdf2image for PDFs). When unavailable, returns empty
text with low confidence and the raw doc reference is kept as source of truth
(PRD risk mitigation). A confidence threshold flags low-quality extractions.
"""
from __future__ import annotations

from dataclasses import dataclass

CONFIDENCE_THRESHOLD = 0.5


@dataclass
class OCRResult:
    text: str
    confidence: float
    low_confidence: bool
    engine: str


def ocr_image_bytes(data: bytes, content_type: str = "image/png") -> OCRResult:
    try:
        return _tesseract(data, content_type)
    except Exception:
        # Fallback: no OCR available — keep raw doc as source of truth.
        return OCRResult(text="", confidence=0.0, low_confidence=True, engine="none")


def _tesseract(data: bytes, content_type: str) -> OCRResult:
    import io

    import pytesseract  # type: ignore
    from PIL import Image  # type: ignore

    images = []
    if content_type == "application/pdf":
        from pdf2image import convert_from_bytes  # type: ignore

        images = convert_from_bytes(data)
    else:
        images = [Image.open(io.BytesIO(data))]

    texts = []
    confidences = []
    for img in images:
        data_dict = pytesseract.image_to_data(img, output_type=pytesseract.Output.DICT)
        words = [w for w in data_dict["text"] if w.strip()]
        confs = [int(c) for c in data_dict["conf"] if str(c).lstrip("-").isdigit() and int(c) >= 0]
        texts.append(" ".join(words))
        if confs:
            confidences.append(sum(confs) / len(confs) / 100.0)

    text = "\n".join(t for t in texts if t)
    conf = sum(confidences) / len(confidences) if confidences else 0.0
    return OCRResult(text=text, confidence=conf, low_confidence=conf < CONFIDENCE_THRESHOLD, engine="tesseract")
