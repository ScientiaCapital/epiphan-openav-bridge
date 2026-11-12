# openav-mcp: stdio MCP server in front of the OpenAV orchestrator and the Epiphan services.
FROM python:3.12-slim@sha256:5e2bdb1b0d542d5039cffb8181c65930093949194a65c365dfb1b0fe13a4fa5b

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

WORKDIR /app

# Dependencies first, from the lock, hash-checked. Regenerate requirements.lock with:
#   uv export --frozen --no-dev --no-emit-project --format requirements-txt -o requirements.lock
COPY requirements.lock ./
RUN pip install --no-cache-dir --require-hashes -r requirements.lock

COPY pyproject.toml README.md ./
COPY openav_mcp ./openav_mcp
RUN pip install --no-cache-dir --no-deps . \
 && useradd --system --uid 10001 --no-create-home --shell /usr/sbin/nologin svc

USER svc

# Config comes from OPENAV_* environment variables (see README). Add --read-only to drop control tools.
ENTRYPOINT ["python", "-m", "openav_mcp"]
