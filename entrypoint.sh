#!/bin/sh
set -eu

# Defaults kept explicit so Coolify / compose env overrides are the only opt-outs.
export BIND_ADDR="${BIND_ADDR:-127.0.0.1}"
export WHATSAPP_READ_ONLY="${WHATSAPP_READ_ONLY:-true}"

BRIDGE_PID=""
PROXY_PID=""
STOPPING=0
# Pause before exiting non-zero. `docker` resets its restart back-off after 10s of uptime, and an
# unpaired bridge exits after the 3-minute QR timeout, so without this the container would restart
# immediately, forever. Interruptible: SIGTERM during the pause still runs term_handler.
RESTART_DELAY="${RESTART_DELAY:-30}"

term_handler() {
	STOPPING=1
	echo "entrypoint: received signal, stopping children" >&2
	if [ -n "$BRIDGE_PID" ]; then kill -TERM "$BRIDGE_PID" 2>/dev/null || true; fi
	if [ -n "$PROXY_PID" ]; then kill -TERM "$PROXY_PID" 2>/dev/null || true; fi
	wait "$BRIDGE_PID" 2>/dev/null || true
	wait "$PROXY_PID" 2>/dev/null || true
	exit 0
}
trap term_handler INT TERM HUP

delay_then_fail() {
	echo "entrypoint: waiting ${RESTART_DELAY}s before exit (restart throttle)" >&2
	sleep "$RESTART_DELAY" &
	wait $! || true
	exit 1
}

# 1) Go bridge. REST API (:8080) is bound to loopback INSIDE the container only
#    (no auth on it — never publish this port). QR codes go to stdout → docker logs.
cd /app/whatsapp-bridge
./whatsapp-bridge &
BRIDGE_PID=$!

# 2) stdio MCP server → HTTP (streamable /mcp and SSE /sse) via mcp-proxy
cd /app/whatsapp-mcp-server
/opt/proxy/bin/mcp-proxy --host 0.0.0.0 --port 8096 --pass-environment -- \
	/app/whatsapp-mcp-server/.venv/bin/python main.py &
PROXY_PID=$!

# If either dies, exit non-zero so the orchestrator restarts the container
# (and re-prints a fresh QR while the device is not paired yet).
while [ "$STOPPING" -eq 0 ]; do
	if ! kill -0 "$BRIDGE_PID" 2>/dev/null; then
		echo "whatsapp-bridge exited, stopping container" >&2
		kill -TERM "$PROXY_PID" 2>/dev/null || true
		wait "$PROXY_PID" 2>/dev/null || true
		delay_then_fail
	fi
	if ! kill -0 "$PROXY_PID" 2>/dev/null; then
		echo "mcp-proxy exited, stopping container" >&2
		kill -TERM "$BRIDGE_PID" 2>/dev/null || true
		wait "$BRIDGE_PID" 2>/dev/null || true
		delay_then_fail
	fi
	sleep 2
done
