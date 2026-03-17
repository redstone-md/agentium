#!/usr/bin/env bash
set -euo pipefail

should_run_headless() {
  if [[ "${AGENTIUM_HEADLESS:-false}" == "true" ]]; then
    return 0
  fi

  local previous=""
  for arg in "$@"; do
    if [[ "${arg}" == "-headless=true" ]]; then
      return 0
    fi
    if [[ "${previous}" == "-headless" && "${arg}" == "true" ]]; then
      return 0
    fi
    previous="${arg}"
  done

  return 1
}

if should_run_headless "$@"; then
  exec /usr/local/bin/agentium "$@"
fi

Xvfb "${DISPLAY}" -screen 0 1280x800x24 -nolisten tcp &
XVFB_PID=$!

cleanup() {
  kill "${XVFB_PID}" >/dev/null 2>&1 || true
}

trap cleanup EXIT

exec /usr/local/bin/agentium "$@"
