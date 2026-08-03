# sub2api-fast-toggle

Language: **English** · [简体中文](README.zh-CN.md)

> Add a local, runtime Fast-mode switch to an existing Sub2API setup without patching Sub2API or moving account management.

[![CI](https://github.com/SeriesApple/sub2api-fast-toggle/actions/workflows/ci.yml/badge.svg)](https://github.com/SeriesApple/sub2api-fast-toggle/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

This project places a small reverse proxy in front of an existing [Sub2API](https://github.com/Wei-Shaw/sub2api) endpoint. It changes only the `service_tier` field on configured Responses API requests.

- **API clients use port `18081`** for the Fast-mode switch.
- **Sub2API stays on port `8081`** for account management and normal administration.
- **Fast mode changes immediately** for new requests, with no container restart.
- **Credentials stay where they are**; the proxy stores no API keys or account data.

**Core idea:** client request → local Fast proxy → existing Sub2API → upstream provider.

> This is an independent, unofficial interoperability project. It is not affiliated with Sub2API, Wei-Shaw, OpenAI, or Docker. See [NOTICE.md](NOTICE.md).

---

## ✨ Features

- **One-click switching** — double-click `Toggle Fast.cmd` on Windows, or use the PowerShell and POSIX helpers.
- **Hot switching** — changes apply to the next matching request with no container restart.
- **Global by design** — one mode applies to the whole proxy instance; per-conversation switching is not supported.
- **Precise rewriting** — touches only configured Responses paths and selected model IDs.
- **Safe ON behavior** — adds `"service_tier":"fast"` only when the request has no tier.
- **Safe OFF behavior** — removes explicit `fast` or `priority` while preserving other tiers.
- **Streaming-friendly** — forwards SSE responses without waiting for the stream to finish.
- **Local by default** — binds to `127.0.0.1` and exposes a configuration-only status endpoint.
- **Hardened container** — runs read-only as a non-root user with all Linux capabilities dropped.

---

## 🚀 Quick start

### 1. Configure and install

Windows PowerShell:

```powershell
git clone https://github.com/SeriesApple/sub2api-fast-toggle.git
cd sub2api-fast-toggle
Copy-Item .env.example .env
notepad .env
.\scripts\Install.ps1
```

Linux:

```sh
git clone https://github.com/SeriesApple/sub2api-fast-toggle.git
cd sub2api-fast-toggle
cp .env.example .env
chmod +x scripts/*.sh
./scripts/install.sh
```

The installer creates the local mode file, builds and starts the proxy, and verifies the saved mode. It requires Docker Engine or Docker Desktop with Docker Compose v2.20 or later.

### 2. Point the API client to the proxy

Use this OpenAI-compatible base URL:

```text
http://localhost:18081/v1
```

For a Codex custom provider, change only the provider base URL. Keep the API key in Codex's existing authentication store.

```toml
[model_providers.custom]
name = "custom"
base_url = "http://localhost:18081/v1"
wire_api = "responses"
requires_openai_auth = true
```

### 3. Switch Fast mode

On Windows, double-click `Toggle Fast.cmd`, or run:

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

Each new request reads the current switch state. A streaming response that has already started keeps the tier it started with.

---

## 🔌 Which port should I use?

| Address | Use it for |
| --- | --- |
| `http://localhost:8081` | Existing Sub2API management, accounts, and administration |
| `http://localhost:18081/v1` | API clients that should obey the Fast-mode switch |
| `http://localhost:18081/_sub2api-fast/status` | Local switch status and configuration check |

Managing accounts on `8081` does not disable or bypass the Fast switch. The switch applies when an API client sends its Responses traffic through `18081`.

## 🔁 How the switch behaves

| Mode | Matching Responses request |
| --- | --- |
| **ON** | Adds the configured Fast tier only when `service_tier` is missing. An explicit tier is left unchanged. |
| **OFF** | Removes explicit `fast` or `priority`. Other values, such as `default`, remain unchanged. |

Only configured paths and models are inspected. Other requests pass through unchanged.

The switch is global to this proxy instance. It hot-switches new requests, but cannot assign different Fast modes to individual conversations.

## 🧭 Request flow

```text
Codex / API client
        │
        │  http://127.0.0.1:18081/v1
        ▼
Sub2API Fast Toggle
        │
        │  http://host.docker.internal:8081
        ▼
Existing Sub2API installation
        │
        ▼
Upstream provider
```

The proxy does not patch Sub2API, access its database, or replace its management interface.

## ⚙️ Configuration

Copy `.env.example` to `.env`, then edit only the values you need:

| Variable | Default | Purpose |
| --- | --- | --- |
| `FAST_PROXY_BIND` | `127.0.0.1` | Host interface exposed by the proxy |
| `FAST_PROXY_PORT` | `18081` | Port used by API clients |
| `SUB2API_URL` | `http://host.docker.internal:8081` | Existing Sub2API endpoint as seen from the container |
| `PRIORITY_MODELS` | `gpt-5.6-sol` | Comma-separated model IDs; `*` matches every model |
| `FAST_SERVICE_TIER` | `fast` | Injected value: `fast` or `priority` |
| `RESPONSE_PATHS` | Three Sub2API aliases | Comma-separated Responses endpoint paths |
| `MAX_BODY_BYTES` | `67108864` | Maximum JSON request size inspected by the proxy |

Never add API keys, passwords, tokens, database URLs, exported logs, or account data to `.env`.

## ✅ Verify

```powershell
Invoke-RestMethod http://127.0.0.1:18081/_sub2api-fast/status
```

Example:

```json
{
  "mode": "on",
  "valid": true,
  "fast_service_tier": "fast",
  "models": ["gpt-5.6-sol"],
  "paths": ["/backend-api/codex/responses", "/responses", "/v1/responses"]
}
```

The status endpoint returns configuration only. It never returns API keys, headers, prompts, responses, or account information.

## 🧪 Verification coverage

CI runs race-enabled Go tests, `go vet`, and a clean container build. The test suite covers ON/OFF rewriting, explicit tiers, live mode changes, fail-safe behavior, request rejection, authorization forwarding, non-matching passthrough, status output, and SSE streaming.

## 🔒 Security and privacy

- The proxy binds to localhost by default.
- Authorization is forwarded to Sub2API but never logged.
- Request and response bodies are never logged.
- Local state contains only `on` or `off`.
- Compressed, malformed, or oversized matching requests are rejected instead of bypassing inspection.

Read [SECURITY.md](SECURITY.md) before exposing the proxy outside the local host.

## 🗑️ Uninstall

Windows:

```powershell
.\scripts\Uninstall.ps1
```

Linux:

```sh
./scripts/uninstall.sh
```

The helpers preserve `.env` and `runtime/mode`. After uninstalling, restore the client's base URL to the original Sub2API endpoint.

## ⚠️ Cost and compatibility

Fast mode has premium per-token pricing and model eligibility requirements. OpenAI may report the actual processed tier in the response, and traffic may be downgraded under documented ramp-rate conditions.

Review the current [Fast mode guide](https://developers.openai.com/api/docs/guides/fast-mode) and [pricing](https://developers.openai.com/api/docs/pricing?latest-pricing=fast) before enabling it. This proxy cannot add Fast capability to a model or upstream account that does not support it.

## 🤝 Contributing

Contributions are welcome. Keep the proxy small, auditable, provider-neutral, and free of credentials or personal data. See [CONTRIBUTING.md](CONTRIBUTING.md).

## 📄 License and attribution

Original code in this repository is licensed under the [MIT License](LICENSE). Sub2API is a separate project and is not redistributed here. See [NOTICE.md](NOTICE.md) and [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) for attribution and third-party license details.
