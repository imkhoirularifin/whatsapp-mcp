# ---------- stage 1: build Go bridge from THIS repo ----------
FROM golang:1.26.10-bookworm AS bridge
WORKDIR /src/whatsapp-bridge
RUN apt-get update \
 && apt-get install -y --no-install-recommends gcc libc6-dev \
 && rm -rf /var/lib/apt/lists/*
COPY whatsapp-bridge/go.mod whatsapp-bridge/go.sum ./
RUN go mod download
COPY whatsapp-bridge/ ./
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/whatsapp-bridge .

# ---------- stage 2: runtime (bridge + Python MCP server + mcp-proxy) ----------
FROM python:3.12.15-slim-bookworm

# Pin mcp-proxy so Coolify rebuilds stay reproducible
ARG MCP_PROXY_VERSION=0.13.0

ENV UV_PYTHON_DOWNLOADS=never \
    PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    WHATSAPP_READ_ONLY=true \
    BIND_ADDR=127.0.0.1

RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates tini ffmpeg \
 && rm -rf /var/lib/apt/lists/* \
 && pip install --no-cache-dir "uv==0.13.5" \
 && python -m venv /opt/proxy \
 && /opt/proxy/bin/pip install --no-cache-dir "mcp-proxy==${MCP_PROXY_VERSION}"

# Layout must match what the MCP server hardcodes:
#   ../whatsapp-bridge/store/messages.db  and  http://localhost:8080/api
COPY whatsapp-mcp-server /app/whatsapp-mcp-server
COPY --from=bridge /out/whatsapp-bridge /app/whatsapp-bridge/whatsapp-bridge

RUN cd /app/whatsapp-mcp-server && uv sync --frozen --no-dev \
 && useradd -r -u 10001 -d /app -s /usr/sbin/nologin app \
 && mkdir -p /app/whatsapp-bridge/store \
 && chown -R app:app /app/whatsapp-bridge /app/whatsapp-mcp-server

COPY entrypoint.sh /entrypoint.sh
RUN chmod 755 /entrypoint.sh

USER app
WORKDIR /app
VOLUME /app/whatsapp-bridge/store
EXPOSE 8096

# mcp-proxy listens on 8096; the unauthenticated bridge stays on 127.0.0.1:8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=3 \
  CMD python -c "import socket; socket.create_connection(('127.0.0.1', 8096), 3)"

ENTRYPOINT ["/usr/bin/tini", "--", "/entrypoint.sh"]
