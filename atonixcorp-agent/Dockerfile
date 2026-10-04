FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1

WORKDIR /app

RUN groupadd --system --gid 10001 agent \
    && useradd --system --uid 10001 --gid agent --home-dir /data agent \
    && mkdir -p /data \
    && chown agent:agent /data

COPY --chown=agent:agent api ./api
COPY --chown=agent:agent db ./db
COPY --chown=agent:agent mcp ./mcp
COPY --chown=agent:agent ml ./ml
COPY --chown=agent:agent model/__init__.py ./model/__init__.py
COPY --chown=agent:agent model/core ./model/core
COPY --chown=agent:agent rag ./rag
COPY --chown=agent:agent rl ./rl
COPY --chown=agent:agent skills ./skills

USER 10001:10001

EXPOSE 8000

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["python", "-c", "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8000/healthz', timeout=2)"]

CMD ["python", "-m", "api.server", "--host", "0.0.0.0", "--port", "8000", "--agent-root", "/data"]
