# Agentium

Agentium is an AI-native browser engine written in Go. It drives Chrome or Chromium through CDP with `go-rod`, exposes a REST API for browser sessions, and ships MCP tools for agent-native integrations.

## Quick Start

Build:

```bash
go build -o agentium ./cmd/agentium
```

Run REST + MCP SSE in the default headful mode:

```bash
./agentium
```

Run headless:

```bash
./agentium -headless=true
```

Run MCP over stdio:

```bash
./agentium -mode mcp-stdio
```

Print the effective startup config:

```bash
./agentium -print-config
```

## Runtime Controls

Agentium supports both environment variables and CLI flags. CLI flags override env values.

### CLI flags

- `-mode=http|mcp-stdio`
- `-http-addr=:8080`
- `-chrome-bin=/absolute/path/to/chrome`
- `-headless=true|false`
- `-leakless=true|false`
- `-viewport-width=1280`
- `-viewport-height=800`
- `-print-config`

### Environment variables

- `AGENTIUM_HTTP_ADDR` default: `:8080`
- `AGENTIUM_CHROME_BIN` default: empty
- `AGENTIUM_HEADLESS` default: `false`
- `AGENTIUM_LEAKLESS` default: `true`
- `AGENTIUM_VIEWPORT_WIDTH` default: `1280`
- `AGENTIUM_VIEWPORT_HEIGHT` default: `800`
- `AGENTIUM_GEOIP_ENDPOINT` default: `https://ipwho.is/`
- `AGENTIUM_GEOIP_TIMEOUT_SECONDS` default: `8`

A ready-to-copy example is available in [`agentium.env.example`](D:\code\Agentium\examples\agentium.env.example).

## Session Modes

Session creation supports two isolation modes:

- `incognito`: default. Lightweight isolated browser contexts.
- `persistent`: dedicated browser profile directory per session. Use this for stealth-sensitive work and sites that penalize incognito mode.

Recommended:

- Use `persistent` for Pixelscan, BrowserScan, login flows, and long-lived profiles.
- Use `incognito` for short isolated tasks when stealth quality matters less than throughput.

## Local Run

Headful HTTP server on a custom port:

```bash
./agentium -http-addr :9090 -headless=false
```

Headless MCP stdio server:

```bash
./agentium -mode mcp-stdio -headless=true
```

Use a specific Chrome binary:

```bash
./agentium -chrome-bin "/usr/bin/google-chrome"
```

On Windows PowerShell:

```powershell
$env:AGENTIUM_CHROME_BIN="C:\Program Files\Google\Chrome\Application\chrome.exe"
.\agentium.exe -headless=false
```

## Docker Run

The runtime image is based on Debian bookworm, not Ubuntu. Ubuntu 24.04 moves Chromium to a snap-oriented package flow, which is a poor fit for this minimal Xvfb container.

Build:

```bash
docker build -t agentium:latest .
```

Run headful:

```bash
docker run --rm -p 8080:8080 agentium:latest
```

Run headless:

```bash
docker run --rm -p 8080:8080 -e AGENTIUM_HEADLESS=true agentium:latest
```

The container starts `Xvfb` only when `AGENTIUM_HEADLESS=false`.

## REST API

Health check:

```bash
curl http://127.0.0.1:8080/healthz
```

Create session:

```bash
curl -X POST http://127.0.0.1:8080/v1/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "session_mode": "persistent",
    "timezone_id": "Europe/Prague",
    "locale": "cs-CZ"
  }'
```

Get snapshot:

```bash
curl http://127.0.0.1:8080/v1/sessions/<session_id>/snapshot
```

Perform action:

```bash
curl -X POST http://127.0.0.1:8080/v1/sessions/<session_id>/action \
  -H "Content-Type: application/json" \
  -d '{
    "action": "navigate",
    "value": "https://example.com"
  }'
```

Close session:

```bash
curl -X DELETE http://127.0.0.1:8080/v1/sessions/<session_id>
```

## MCP

### Stdio mode

Available tools:

- `agentium_create_session`
- `agentium_get_snapshot`
- `agentium_perform_action`
- `agentium_close_session`

Launch:

```bash
./agentium -mode mcp-stdio -headless=false
```

### SSE mode

When Agentium runs in HTTP mode, MCP SSE is exposed under:

- `GET /mcp`
- `POST /mcp/*`

## Agent Instructions

The recommended agent workflow is documented in [`agent-playbook.md`](D:\code\Agentium\docs\agent-playbook.md).

Ready-made Claude Desktop configs:

- headful: [`claude_desktop_config.json`](D:\code\Agentium\examples\claude_desktop_config.json)
- headless: [`claude_desktop_config.headless.json`](D:\code\Agentium\examples\claude_desktop_config.headless.json)

## Smoke Tests

REST smoke test:

```powershell
pwsh ./scripts/smoke-http.ps1
```

MCP stdio smoke test:

```powershell
pwsh ./scripts/smoke-mcp-stdio.ps1 -BinaryPath .\agentium.exe
```

MCP SSE smoke test:

```powershell
pwsh ./scripts/smoke-mcp-sse.ps1
```

Local HTTP + SSE smoke run:

```powershell
pwsh ./scripts/run-local-http-smokes.ps1 -BinaryPath .\agentium.exe
```

If local Windows antivirus blocks Rod's `leakless` helper:

```powershell
pwsh ./scripts/run-local-http-smokes.ps1 -BinaryPath .\agentium.exe -DisableLeakless
```

Stealth validation against `bot.sannysoft.com`:

```powershell
pwsh ./scripts/run-local-http-smokes.ps1 -BinaryPath .\agentium.exe -DisableLeakless -TargetUrl https://bot.sannysoft.com/ -DelayAfterNavigateMs 8000
```

Stealth validation against `browserscan.net`:

```powershell
pwsh ./scripts/run-local-stealth-browserscan.ps1 -BinaryPath .\agentium.exe -DisableLeakless
```

Stealth validation against `pixelscan.net`:

```powershell
pwsh ./scripts/run-local-stealth-pixelscan.ps1 -BinaryPath .\agentium.exe -DisableLeakless
```

## Notes

- Browser pooling is keyed by proxy configuration.
- `persistent` sessions launch dedicated browser profiles and intentionally trade some startup cost for better real-world browser parity.
- `wait_network_idle` focuses on XHR and Fetch activity, which is usually the useful signal for form submissions and SPA mutations.
