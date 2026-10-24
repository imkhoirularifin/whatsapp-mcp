#!/bin/sh
# Refuse to start without a strong bearer token, so a missing/unset Coolify variable can
# never turn the gateway into an open (or "Bearer "-matching) proxy.
set -eu
tok="${MCP_AUTH_TOKEN:-}"
if [ "${#tok}" -lt 48 ]; then
	echo "gateway: MCP_AUTH_TOKEN missing or shorter than 48 chars; refusing to start" >&2
	exit 1
fi
case "$tok" in
	*[!A-Za-z0-9._~-]*) echo "gateway: MCP_AUTH_TOKEN has characters outside [A-Za-z0-9._~-]; refusing to start" >&2; exit 1 ;;
esac
exec caddy run --config /etc/caddy/Caddyfile --adapter caddyfile
