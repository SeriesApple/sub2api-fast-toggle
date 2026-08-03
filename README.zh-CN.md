# sub2api-fast-toggle

语言：[English](README.md) · **简体中文**

> 在不修改 Sub2API、也不改变账号管理入口的前提下，为现有 Sub2API 增加一个本地、可随时切换的 Fast 模式开关。

[![CI](https://github.com/SeriesApple/sub2api-fast-toggle/actions/workflows/ci.yml/badge.svg)](https://github.com/SeriesApple/sub2api-fast-toggle/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![许可证：MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

本项目在现有 [Sub2API](https://github.com/Wei-Shaw/sub2api) 前增加一个轻量反向代理，只修改指定 Responses API 请求中的 `service_tier` 字段。

- **API 客户端使用 `18081` 端口**，通过 Fast 开关发送请求。
- **Sub2API 仍使用 `8081` 端口**，账号管理和后台操作完全不变。
- **开关即时生效**，新请求无需重启容器。
- **凭据留在原处**，代理不保存 API Key 或账号数据。

**核心思路**：客户端请求 → 本地 Fast 代理 → 现有 Sub2API → 上游服务商。

> 这是一个独立、非官方的兼容性项目，与 Sub2API、Wei-Shaw、OpenAI 或 Docker 不存在隶属或官方合作关系。详见 [NOTICE.md](NOTICE.md)。

---

## ✨ 功能

- **一键切换**——Windows 双击 `Toggle Fast.cmd`，也可使用 PowerShell 或 POSIX Shell 脚本。
- **支持热切换**——无需重启容器，下一个匹配请求就会使用新的模式。
- **全局开关**——同一代理实例统一使用一个模式，不支持对单个对话分别切换。
- **精确改写**——只处理配置的 Responses 路径和指定模型。
- **安全开启**——仅在请求没有 tier 时添加 `"service_tier":"fast"`。
- **安全关闭**——删除显式的 `fast` 或 `priority`，保留其他 tier。
- **支持流式响应**——SSE 内容到达后立即转发，不等待整段响应结束。
- **默认仅限本机**——绑定 `127.0.0.1`，状态接口只返回配置。
- **容器加固**——只读文件系统、非 root 用户运行，并移除全部 Linux capabilities。

---

## 🚀 快速开始

### 1. 配置并安装

Windows PowerShell：

```powershell
git clone https://github.com/SeriesApple/sub2api-fast-toggle.git
cd sub2api-fast-toggle
Copy-Item .env.example .env
notepad .env
.\scripts\Install.ps1
```

Linux：

```sh
git clone https://github.com/SeriesApple/sub2api-fast-toggle.git
cd sub2api-fast-toggle
cp .env.example .env
chmod +x scripts/*.sh
./scripts/install.sh
```

安装脚本会创建本地模式文件、构建并启动代理，然后核对保存的开关状态。环境需要 Docker Engine 或 Docker Desktop，以及 Docker Compose v2.20 或更高版本。

### 2. 把 API 客户端指向代理

OpenAI 兼容客户端使用以下 Base URL：

```text
http://localhost:18081/v1
```

Codex 自定义 Provider 只需修改 `base_url`。API Key 继续保存在 Codex 原有的认证存储中。

```toml
[model_providers.custom]
name = "custom"
base_url = "http://localhost:18081/v1"
wire_api = "responses"
requires_openai_auth = true
```

### 3. 切换 Fast 模式

Windows 可以双击 `Toggle Fast.cmd`，也可以执行：

```powershell
.\scripts\Toggle-Fast.ps1 -Set On
.\scripts\Toggle-Fast.ps1 -Set Off
.\scripts\Toggle-Fast.ps1 -Status
```

Linux：

```sh
./scripts/toggle-fast.sh on
./scripts/toggle-fast.sh off
./scripts/toggle-fast.sh status
```

每个新请求都会读取一次当前开关状态。已经开始的流式响应会保持启动时所使用的 tier。

---

## 🔌 到底该用哪个端口？

| 地址 | 用途 |
| --- | --- |
| `http://localhost:8081` | 原有 Sub2API 管理后台、账号与日常管理 |
| `http://localhost:18081/v1` | 需要服从 Fast 开关的 API 客户端 |
| `http://localhost:18081/_sub2api-fast/status` | 查看本地开关状态与配置 |

在 `8081` 管理账号不会关闭或绕过 Fast 开关。只有客户端把 Responses 请求发到 `18081` 时，开关才会参与处理。

## 🔁 开关如何处理请求

| 模式 | 对匹配的 Responses 请求执行的操作 |
| --- | --- |
| **ON** | 仅在缺少 `service_tier` 时添加配置的 Fast tier；已有的显式 tier 保持不变。 |
| **OFF** | 删除显式的 `fast` 或 `priority`；`default` 等其他值保持不变。 |

只有配置的路径和模型会被检查，其他请求全部原样转发。

开关对当前代理实例全局生效：它支持新请求热切换，但不能为单个对话分别设置不同的 Fast 模式。

## 🧭 请求路径

```text
Codex / API 客户端
        │
        │  http://127.0.0.1:18081/v1
        ▼
Sub2API Fast Toggle
        │
        │  http://host.docker.internal:8081
        ▼
现有 Sub2API
        │
        ▼
上游服务商
```

代理不会修改 Sub2API 源码、访问其数据库，也不会替代原有管理界面。

## ⚙️ 配置

将 `.env.example` 复制为 `.env`，然后只修改需要的项目：

| 变量 | 默认值 | 用途 |
| --- | --- | --- |
| `FAST_PROXY_BIND` | `127.0.0.1` | 代理绑定的主机接口 |
| `FAST_PROXY_PORT` | `18081` | API 客户端连接端口 |
| `SUB2API_URL` | `http://host.docker.internal:8081` | 容器内看到的现有 Sub2API 地址 |
| `PRIORITY_MODELS` | `gpt-5.6-sol` | 逗号分隔的模型 ID；`*` 表示全部模型 |
| `FAST_SERVICE_TIER` | `fast` | 注入值：`fast` 或 `priority` |
| `RESPONSE_PATHS` | 三个 Sub2API 路径别名 | 逗号分隔的 Responses 接口路径 |
| `MAX_BODY_BYTES` | `67108864` | 代理检查的最大 JSON 请求大小 |

不要把 API Key、密码、Token、数据库地址、导出日志或账号数据写入 `.env`。

## ✅ 验证状态

```powershell
Invoke-RestMethod http://127.0.0.1:18081/_sub2api-fast/status
```

示例：

```json
{
  "mode": "on",
  "valid": true,
  "fast_service_tier": "fast",
  "models": ["gpt-5.6-sol"],
  "paths": ["/backend-api/codex/responses", "/responses", "/v1/responses"]
}
```

状态接口只返回配置，不会返回 API Key、请求头、提示词、响应内容或账号信息。

## 🧪 验证范围

CI 会运行带 race detector 的 Go 测试、`go vet` 和干净的容器构建。测试覆盖 ON/OFF 改写、显式 tier、实时切换、失效保护、异常请求拒绝、认证头转发、非匹配请求直通、状态输出和 SSE 流式响应。

## 🔒 安全与隐私

- 默认仅绑定本机 `127.0.0.1`。
- 认证头只转发给 Sub2API，绝不写入日志。
- 不记录请求正文和响应正文。
- 本地状态只保存 `on` 或 `off`。
- 匹配路径上的压缩、畸形或超大请求会被拒绝，避免绕过检查。

将代理暴露到本机之外前，请先阅读 [SECURITY.md](SECURITY.md)。

## 🗑️ 卸载

Windows：

```powershell
.\scripts\Uninstall.ps1
```

Linux：

```sh
./scripts/uninstall.sh
```

卸载脚本会保留 `.env` 和 `runtime/mode`。卸载后，请把客户端 Base URL 改回原来的 Sub2API 地址。

## ⚠️ 费用与兼容性

Fast 模式通常采用更高的按 Token 计费，并且存在模型和上游账号资格限制。OpenAI 可能在响应中返回实际处理 tier，也可能在官方规定的流量爬升限制下回落。

开启前请阅读最新的 [Fast mode 文档](https://developers.openai.com/api/docs/guides/fast-mode)和[官方价格页面](https://developers.openai.com/api/docs/pricing?latest-pricing=fast)。本代理无法让原本不支持 Fast 的模型或上游账号获得 Fast 能力。

## 🤝 贡献

欢迎提交 PR。请保持代理小巧、可审计、与上游服务商无关，并确保提交中不含凭据或个人信息。详见 [CONTRIBUTING.md](CONTRIBUTING.md)。

## 📄 许可证与引用

本仓库原创代码采用 [MIT License](LICENSE)。Sub2API 是独立项目，本仓库不分发其源码。引用和第三方许可证信息请参阅 [NOTICE.md](NOTICE.md) 与 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。
