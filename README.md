# Sub2API Fast Toggle

[简体中文](README.zh-CN.md)

An unofficial, local-only Fast-mode switch for OpenAI-compatible Responses API
traffic routed through [Sub2API](https://github.com/Wei-Shaw/sub2api).

The project runs a small reverse proxy in front of an existing Sub2API endpoint.
It does not patch Sub2API, access the Sub2API database, or store credentials.

> This project is independent and is not affiliated with Sub2API, Wei-Shaw,
> OpenAI, or Docker. See [NOTICE.md](NOTICE.md).

## Why It Exists

Some clients do not send a `service_tier` field or do not expose a Fast-mode
control when they use an API key or a custom provider. This proxy provides a
runtime switch:

| Mode | Matching Responses request |
| --- | --- |
| ON | Adds `"service_tier":"fast"` only when the field is missing. |
| OFF | Removes explicit `fast` or `priority`; other tiers remain unchanged. |

OpenAI renamed Priority processing to Fast mode on July 30, 2026. The official
API accepts both `fast` and `priority`; see the
[OpenAI Fast mode guide](https://developers.openai.com/api/docs/guides/fast-mode).

## Architecture

```text
API client / Codex
        |
        | http://127.0.0.1:18081/v1
        v
Sub2API Fast Toggle
        |
        | http://host.docker.internal:8081
        v
Existing Sub2API installation
```

Sub2API keeps its original port. Its management interface normally remains at
`http://localhost:8081`; only API clients that need this switch use port
`18081`.

## Requirements

- A working Sub2API installation reachable on the host, normally port `8081`.
- Docker Engine or Docker Desktop with Docker Compose v2.20 or later.
- A model and upstream account that support Fast processing.
- Windows PowerShell 5.1+ for the Windows helpers, or a POSIX shell for the
  shell helpers.

## Quick Start

### Windows

Open PowerShell in the repository directory:

```powershell
Copy-Item .env.example .env
notepad .env
.\scripts\Install.ps1
```

The installer creates `runtime/mode`, builds the proxy, starts it, and verifies
the saved mode.

### Linux

```sh
cp .env.example .env
chmod +x scripts/*.sh
./scripts/install.sh
```

On Linux, `host.docker.internal` is mapped to the Docker host through
`host-gateway` by the included Compose file.

## Client Configuration

Point the OpenAI-compatible client at:

```text
http://localhost:18081/v1
```

For a Codex custom provider, update only the provider base URL. Keep the API key
in Codex's existing authentication store; never put it in this repository.

```toml
[model_providers.custom]
name = "custom"
base_url = "http://localhost:18081/v1"
wire_api = "responses"
requires_openai_auth = true
```

Sub2API account management remains at its original address, normally
`http://localhost:8081`.

## Toggle Fast Mode

On Windows, double-click `Toggle Fast.cmd`, or use PowerShell:

```powershell
.\scripts\Toggle-Fast.ps1 -Set On
.\scripts\Toggle-Fast.ps1 -Set Off
.\scripts\Toggle-Fast.ps1 -Status
```

On Linux:

```sh
./scripts/toggle-fast.sh on
./scripts/toggle-fast.sh off
./scripts/toggle-fast.sh status
```

The switch is read for every new request. Active streaming responses keep the
tier chosen when they started.

## Verify

```powershell
Invoke-RestMethod http://127.0.0.1:18081/_sub2api-fast/status
```

Example response:

```json
{
  "mode": "on",
  "valid": true,
  "fast_service_tier": "fast",
  "models": ["gpt-5.6-sol"],
  "paths": ["/backend-api/codex/responses", "/responses", "/v1/responses"]
}
```

The status endpoint contains configuration only. It never returns API keys,
headers, prompts, responses, or account information.

## Configuration

Copy `.env.example` to `.env` and edit these values:

| Variable | Default | Purpose |
| --- | --- | --- |
| `FAST_PROXY_BIND` | `127.0.0.1` | Host interface exposed by the proxy. |
| `FAST_PROXY_PORT` | `18081` | Port used by API clients. |
| `SUB2API_URL` | `http://host.docker.internal:8081` | Existing Sub2API endpoint. |
| `PRIORITY_MODELS` | `gpt-5.6-sol` | Comma-separated model IDs; `*` matches all. |
| `FAST_SERVICE_TIER` | `fast` | Injected value: `fast` or `priority`. |
| `RESPONSE_PATHS` | three Sub2API aliases | Comma-separated Responses paths. |
| `MAX_BODY_BYTES` | `67108864` | Maximum inspected JSON request size. |

Do not add API keys, passwords, tokens, database URLs, or exported logs to
`.env`.

## Uninstall

Windows:

```powershell
.\scripts\Uninstall.ps1
```

Linux:

```sh
./scripts/uninstall.sh
```

The uninstall helpers preserve `.env` and `runtime/mode`. Restore the client's
base URL to the original Sub2API endpoint after uninstalling.

## Security and Privacy

- The proxy binds to localhost by default.
- It forwards authorization to Sub2API but never logs authorization headers.
- It does not log request or response bodies.
- It stores only one local mode file containing `on` or `off`.
- Matching compressed, malformed, or oversized requests are rejected rather
  than forwarded without inspection.

See [SECURITY.md](SECURITY.md) before exposing the proxy outside the host.

## Cost and Compatibility

Fast mode has premium per-token pricing and model eligibility requirements.
OpenAI may return the actual processed tier in the response, and traffic may be
downgraded under documented ramp-rate conditions. Review the current
[Fast mode guide](https://developers.openai.com/api/docs/guides/fast-mode) and
[pricing](https://developers.openai.com/api/docs/pricing?latest-pricing=fast)
before enabling it.

This project modifies JSON requests for configured Responses endpoints only.
It does not add Fast mode support to an upstream model or account that lacks it.

## License and Attribution

Original code in this repository is licensed under the MIT License. Sub2API is
a separate project and is not redistributed here. See [LICENSE](LICENSE),
[NOTICE.md](NOTICE.md), and [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
