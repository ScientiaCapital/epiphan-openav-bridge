# openav-mcp — MCP server fronting the OpenAV↔Epiphan bridges.
FROM python:3.12-slim

WORKDIR /app
COPY pyproject.toml README.md ./
COPY openav_mcp ./openav_mcp
RUN pip install --no-cache-dir .

# stdio MCP server. Config via OPENAV_* env vars (see README).
ENTRYPOINT ["python", "-m", "openav_mcp"]
