# FastAPI RAG plane. Heavy extras (embeddings/ocr) installed on demand via
# build args so the base image stays lean for CI.
FROM python:3.12-slim AS base
WORKDIR /app
ENV PIP_NO_CACHE_DIR=1
COPY pyproject.toml ./
COPY mudahurus_rag ./mudahurus_rag
ARG EXTRAS=""
RUN pip install --upgrade pip && \
    if [ -n "$EXTRAS" ]; then pip install ".[$EXTRAS]"; else pip install .; fi
EXPOSE 8000
HEALTHCHECK --interval=30s --timeout=3s CMD python -c "import urllib.request,sys; urllib.request.urlopen('http://localhost:8000/healthz')" || exit 1
CMD ["uvicorn", "mudahurus_rag.api.main:app", "--host", "0.0.0.0", "--port", "8000"]
