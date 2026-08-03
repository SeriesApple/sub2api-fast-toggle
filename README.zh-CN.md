# Sub2API Fast Toggle

[English](README.md)

这是一个非官方、仅本机使用的 Sub2API Fast 模式开关，用于处理经过
[Sub2API](https://github.com/Wei-Shaw/sub2api) 的 OpenAI 兼容 Responses API
请求。

项目通过一个很小的反向代理运行，不修改 Sub2API 源码、不访问 Sub2API
数据库，也不保存任何 API Key。

> 本项目与 Sub2API、Wei-Shaw、OpenAI、Docker 没有隶属或官方合作关系。
> 详细归属说明见 [NOTICE.md](NOTICE.md)。

## 用途

部分客户端在使用 API Key 或自定义 Provider 时，不会发送 `service_tier`，
也不会显示 Fast 开关。本项目提供运行时开关：

| 状态 | 对匹配的 Responses 请求执行的操作 |
| --- | --- |
| ON | 仅在字段缺失时加入 `"service_tier":"fast"`。 |
| OFF | 删除显式的 `fast` 或 `priority`；不改动其他 tier。 |

OpenAI 已在 2026 年 7 月 30 日将 Priority processing 更名为 Fast mode。
官方 API 同时接受 `fast` 和 `priority`，详见
[OpenAI Fast mode 官方文档](https://developers.openai.com/api/docs/guides/fast-mode)。

## 架构

```text
API 客户端 / Codex
        |
        | http://127.0.0.1:18081/v1
        v
Sub2API Fast Toggle
        |
        | http://host.docker.internal:8081
        v
现有 Sub2API
```

Sub2API 保持原端口不变。账号管理通常仍然使用
`http://localhost:8081`；只有需要 Fast 开关的 API 客户端改用 `18081`。

## 环境要求

- 已正常运行的 Sub2API，默认宿主机端口为 `8081`。
- Docker Engine 或 Docker Desktop，Docker Compose v2.20 或更高版本。
- 支持 Fast processing 的模型与上游账号。
- Windows 辅助脚本需要 PowerShell 5.1 或更高版本；Linux 使用 POSIX shell。

## 快速安装

### Windows

在仓库目录打开 PowerShell：

```powershell
Copy-Item .env.example .env
notepad .env
.\scripts\Install.ps1
```

安装脚本会创建 `runtime/mode`、构建代理、启动容器并核对当前状态。

### Linux

```sh
cp .env.example .env
chmod +x scripts/*.sh
./scripts/install.sh
```

Compose 已通过 `host-gateway` 为 Linux 添加 `host.docker.internal` 映射。

## 客户端配置

把 OpenAI 兼容客户端的 API Base URL 改为：

```text
http://localhost:18081/v1
```

Codex 自定义 Provider 只需要修改 `base_url`。API Key 继续保存在 Codex
原有认证存储中，不要写入本仓库。

```toml
[model_providers.custom]
name = "custom"
base_url = "http://localhost:18081/v1"
wire_api = "responses"
requires_openai_auth = true
```

Sub2API 账号管理地址保持不变，通常仍为 `http://localhost:8081`。

## 开关 Fast

Windows 可以双击根目录的 `Toggle Fast.cmd`，或使用 PowerShell：

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

每个新请求都会读取一次开关状态。已经开始的流式回复不会中途切换 tier。

## 验证状态

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

状态接口只返回配置，不返回 API Key、请求头、提示词、回复或账号信息。

## 配置项

将 `.env.example` 复制为 `.env` 后可修改：

| 变量 | 默认值 | 用途 |
| --- | --- | --- |
| `FAST_PROXY_BIND` | `127.0.0.1` | 代理绑定的宿主机接口。 |
| `FAST_PROXY_PORT` | `18081` | API 客户端连接端口。 |
| `SUB2API_URL` | `http://host.docker.internal:8081` | 现有 Sub2API 地址。 |
| `PRIORITY_MODELS` | `gpt-5.6-sol` | 逗号分隔的模型；`*` 表示全部模型。 |
| `FAST_SERVICE_TIER` | `fast` | 注入值，可设为 `fast` 或 `priority`。 |
| `RESPONSE_PATHS` | 三个 Sub2API 别名 | 逗号分隔的 Responses 路径。 |
| `MAX_BODY_BYTES` | `67108864` | 最大 JSON 检查大小。 |

不要把 API Key、密码、Token、数据库地址或导出的日志放进 `.env`。

## 卸载

Windows：

```powershell
.\scripts\Uninstall.ps1
```

Linux：

```sh
./scripts/uninstall.sh
```

卸载脚本会保留 `.env` 和 `runtime/mode`。卸载后需要把客户端 Base URL
改回原来的 Sub2API 地址。

## 安全与隐私

- 默认只绑定 `127.0.0.1`。
- 认证头只转发给 Sub2API，不写入日志。
- 不记录请求正文、回复正文或提示词。
- 本地只保存内容为 `on` 或 `off` 的模式文件。
- 对匹配路径的压缩、畸形或超大请求会直接拒绝，避免 OFF 被绕过。

将代理暴露到其他设备前，请先阅读 [SECURITY.md](SECURITY.md)。

## 费用与兼容性

Fast mode 通常按 Token 收取高于 Standard 的费用，并有模型资格限制。上游
可能在响应中返回实际处理 tier，也可能在官方规定的流量爬升限制下回落到
Standard。开启前请阅读最新的
[Fast mode 文档](https://developers.openai.com/api/docs/guides/fast-mode)和
[官方价格页面](https://developers.openai.com/api/docs/pricing?latest-pricing=fast)。

本项目只修改配置路径上的 Responses JSON 请求，不能让原本不支持 Fast 的
模型或上游账号获得 Fast 能力。

## 许可证与引用

本仓库原创代码使用 MIT License。Sub2API 是独立项目，本仓库不分发其源码。
请阅读 [LICENSE](LICENSE)、[NOTICE.md](NOTICE.md) 和
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。
