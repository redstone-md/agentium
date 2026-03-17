#!/usr/bin/env bash
set -euo pipefail

Xvfb "${DISPLAY}" -screen 0 1280x800x24 -nolisten tcp &
XVFB_PID=$!

cleanup() {
  kill "${XVFB_PID}" >/dev/null 2>&1 || true
}

trap cleanup EXIT

exec /usr/local/bin/agentium "$@"
