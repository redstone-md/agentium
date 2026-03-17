# Agentium Agent Playbook

This guide is for MCP hosts and LLM agents that use Agentium as a browser tool.

## Launch Modes

- `http`: REST API on `AGENTIUM_HTTP_ADDR`, plus MCP SSE on `/mcp`
- `mcp-stdio`: local MCP server over stdio

## Recommended Session Strategy

- Use `session_mode: "persistent"` for fingerprint-sensitive sites, login flows, BrowserScan, and Pixelscan.
- Use `session_mode: "incognito"` for short-lived isolated tasks when anti-detect quality matters less.
- Pass `timezone_id`, `locale`, and `user_agent` only when you need a fixed profile. Otherwise let Agentium auto-resolve them from IP/proxy.

## Recommended Tool Workflow

1. Call `agentium_create_session`.
2. Navigate with `agentium_perform_action` using `action: "navigate"`.
3. Call `agentium_perform_action` with `action: "wait_network_idle"` after navigation or form submission.
4. Call `agentium_get_snapshot` before any `click`, `fill`, or `type`.
5. Use returned `ref_id` values for interactions.
6. Re-snapshot after each page-changing action.
7. Close the session with `agentium_close_session`.

## Practical Defaults

- Prefer persistent sessions on modern anti-bot sites.
- Prefer headful runtime for stealth-sensitive work.
- Use headless mode for CI, fast scraping, and non-stealth local automation.
- If a page looks stalled after navigation, wait for network idle and then snapshot again.

## Example MCP Inputs

Create a persistent session:

```json
{
  "session_mode": "persistent"
}
```

Create a geo-fixed session:

```json
{
  "proxy": "http://user:pass@host:8080",
  "timezone_id": "Europe/Prague",
  "locale": "cs-CZ",
  "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36",
  "session_mode": "persistent"
}
```

Navigate:

```json
{
  "session_id": "your-session-id",
  "action": "navigate",
  "value": "https://example.com"
}
```

Wait for app traffic:

```json
{
  "session_id": "your-session-id",
  "action": "wait_network_idle"
}
```
