# Agentium

Agentium is an AI-native browser engine written in Go. It drives Chromium through CDP using `go-rod`, exposes a REST API for browser sessions, and also ships MCP tools for agent-native integrations.

## Features

- Headful Chromium sessions designed for Xvfb-based container execution.
- Session isolation through incognito browser contexts.
- Browser pooling by proxy configuration with short idle reuse.
- Distilled DOM snapshots with compact `ref_id` mappings for LLM consumption.
- Action execution with humanized mouse movement, typing delays, and scroll behavior.
- Per-session network telemetry with request lifecycle tracking and `wait_network_idle`.
- MCP support over `stdio` and SSE.

## Requirements

- Go 1.25+
- Chromium or Chrome installed for local non-Docker execution
- Linux with Xvfb for containerized headful execution

## Local Run

Build:

```bash
go build -o agentium ./cmd/agentium
```

Run HTTP mode:

```bash
./agentium -mode http
```

Run MCP over stdio:

```bash
./agentium -mode mcp-stdio
```

Useful environment variables:

- `AGENTIUM_HTTP_ADDR` default: `:8080`
- `AGENTIUM_CHROME_BIN` default: empty
- `AGENTIUM_VIEWPORT_WIDTH` default: `1280`
- `AGENTIUM_VIEWPORT_HEIGHT` default: `800`

## Docker Run

The runtime image is currently based on Debian bookworm, not Ubuntu.
Reason: Ubuntu 24.04 ships `chromium-browser` as a transitional package to the Chromium snap,
which is not suitable for this minimal Xvfb container layout.

Build:

```bash
docker build -t agentium:latest .
```

Run:

```bash
docker run --rm -p 8080:8080 agentium:latest
```

The container starts `Xvfb`, launches headful Chromium inside the virtual display, and serves HTTP on port `8080`.

## Smoke Tests

REST smoke test:

```powershell
pwsh ./scripts/smoke-http.ps1
```

MCP stdio smoke test:

```powershell
pwsh ./scripts/smoke-mcp-stdio.ps1 -BinaryPath .\agentium.exe
```

## CI

GitHub Actions workflow is available at [`.github/workflows/ci.yml`](D:\code\Agentium\.github\workflows\ci.yml).

It runs:

- `go test ./...`
- `go build ./...`
- headful HTTP smoke test under `xvfb-run`
- MCP stdio smoke test
- `docker build`

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
    "timezone_id": "Europe/London",
    "locale": "en-GB",
    "user_agent": "Mozilla/5.0 ..."
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
./agentium -mode mcp-stdio
```

### SSE mode

When Agentium runs in HTTP mode, MCP SSE is exposed under:

- `GET /mcp`
- `POST /mcp/*`

## Claude Desktop Example

Example configuration is available in [`examples/claude_desktop_config.json`](D:\code\Agentium\examples\claude_desktop_config.json).

Replace `D:\\code\\Agentium\\agentium.exe` with the actual absolute path to your built binary.

## Current Notes

- Browser pooling is keyed by proxy configuration. Sessions with different proxies use different root Chromium processes.
- `wait_network_idle` currently focuses on `XHR` and `Fetch` activity, which is usually the right signal for form submissions and app mutations.
- The project has build and unit-test coverage, but full live browser and Docker smoke tests still need to be run in an environment with Chromium and Docker available.
