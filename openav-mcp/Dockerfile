# openav-mcp: stdio MCP server in front of the OpenAV orchestrator and the Epiphan services.
FROM python:3.12-alpine@sha256:c5ba3deef00c7758561a0f03c26f44f10589263d40e732b8748e0d1f38f6dd89

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

WORKDIR /app

# Dependencies first, from the lock, hash-checked. Regenerate requirements.lock with:
#   uv export --frozen --no-dev --no-emit-project --format requirements-txt -o requirements.lock
COPY requirements.lock ./
RUN apk upgrade --no-cache && pip install --no-cache-dir --require-hashes -r requirements.lock

COPY pyproject.toml README.md ./
COPY openav_mcp ./openav_mcp
# pip is not needed at runtime; removing it also removes its CVEs from the image.
RUN pip install --no-cache-dir --no-deps . \
 && python -m pip uninstall -y pip \
 && adduser -S -D -H -u 10001 svc

USER svc

# Config comes from OPENAV_* environment variables (see README). Add --read-only to drop control tools.
ENTRYPOINT ["python", "-m", "openav_mcp"]
