# DevBridge

<p align="center">
  <img src="docs/assets/images/devbridge_logo.svg" alt="DevBridge Logo" width="160" />
</p>

<p align="center">
  <strong>Development Tunneling Service · Tunnel / Port / Host / Connect — reach local services remotely without a public inbound port</strong>
</p>

<p align="center">
  <a href="README.zh-CN.md">中文</a> · <a href="README.md">English</a> · <a href="https://huaweicloud.github.io/devspace-devbridge/">Online Docs</a> · <a href="https://huaweicloud.github.io/devspace-devbridge/guide/quickstart">Quick Start</a> · <a href="#architecture">Architecture</a> · <a href="cli/README.md">CLI</a> · <a href="go-sdk/README.md">Go SDK</a>
</p>

<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache--2.0-green.svg" alt="License" /></a>
  <img src="https://img.shields.io/badge/Go-1.26%20%2F%201.22-00ADD8.svg" alt="Go" />
  <img src="https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey.svg" alt="Platform" />
  <a href="https://huaweicloud.github.io/devspace-devbridge/"><img src="https://img.shields.io/badge/docs-online-blue.svg" alt="Docs" /></a>
</p>

DevBridge is Huawei Cloud's development tunneling service. It relays between a **Host** (a device
running a local service) and a **Connect** (a device that wants to reach that service) through a
**Gateway**, so local services can be accessed remotely without opening any public inbound port.
Both Host and Connect open outbound WebSocket connections to the relay — no inbound port, no
exposed firewall rule on the service host.

📖 Online documentation: <https://huaweicloud.github.io/devspace-devbridge/>

---

## Introduction

DevBridge is a development tunneling service that relays a local service through a cloud **Gateway**
so it can be reached remotely without opening an inbound port. It runs in two modes — **Host**
exposes local service ports to the relay, **Connect** opens a local listener that forwards to a
remote tunnel — and both connect outbound to the Gateway. Use it via the [CLI](cli/README.md) or
[Go SDK](go-sdk/README.md).

## Architecture

```text
┌─────────────┐    WebSocket + TLS 1.3    ┌──────────────┐    WebSocket + TLS 1.3    ┌─────────────┐
│  Host       │ ◄──────────────────────► │   Gateway    │ ◄──────────────────────► │ Connect     │
│  (Listen)   │    SSH data channel       │  (Relay)     │    SSH data channel       │  (Send)     │
└──────┬──────┘                          └──────────────┘                          └──────┬──────┘
       │                                                                                 │
   Local service ports                                                            Local TCP listener
   (8080, 3000...)                                                          (forward to Host ports)
```

- **Control channel**: WebSocket connection, carries rendezvous signaling
- **Data channel**: SSH over WebSocket, implements port forwarding (Host side `-R` mode)
- **Authentication**: JWT token passed via the `Sec-WebSocket-Protocol` header
- **Tunnel identity**: Passed via the TLS SNI subdomain (`{tunnelId}.{clusterId}.myhuaweicloud.com`)

Both Host and Connect initiate outbound connections to the Gateway, so the Host never needs a
public inbound port.

## Quick Start

The fastest way to try DevBridge is the CLI. Full command reference:
[cli/README.md](cli/README.md) · online [quickstart](https://huaweicloud.github.io/devspace-devbridge/guide/quickstart).

```bash
# 1. Install (Linux / macOS; Windows: irm .../install.ps1 | iex)
curl -fsSL https://github.com/huaweicloud/devspace-devbridge/releases/latest/download/install.sh | bash

# 2. Log in (Huawei Cloud SSO browser login, or --api-key / HW_API_KEY env)
devbridge auth login

# 3. Host a local port — auto-creates a tunnel and binds the port
devbridge host -p 8080 -d "dev-env" -e 8
#   → your service is now reachable at https://<tunnel-id>-8080.<cluster>.myhuaweicloud.com

# 4. From another machine, connect and forward to localhost
devbridge connect <tunnel-id>
#   → http://localhost:8080 now hits the remote service
```

To embed the same flow in Go code, see the [Go SDK quick start](go-sdk/README.md#quick-start).

## Core Capabilities

| Capability | CLI | Go SDK | Details |
| --- | --- | --- | --- |
| Tunnel management (create / list / update / delete, default tunnel) | ✅ | ✅ | [cli](cli/README.md#tunnel-management) · [sdk](go-sdk/README.md#features) |
| Port management (bind ports, HTTP/HTTPS/Auto, anonymous access) | ✅ | ✅ | [cli](cli/README.md#port-management) · [sdk](go-sdk/README.md#features) |
| Host mode — expose local ports to the relay | ✅ | ✅ | [cli](cli/README.md#remote-port-forwarding) · [sdk](go-sdk/README.md#quick-start) |
| Connect mode — reach a remote tunnel via a local listener | ✅ | ✅ | [cli](cli/README.md#remote-port-forwarding) · [sdk](go-sdk/README.md#quick-start) |
| Token issuance (Host / Connect JWT) | ✅ | ✅ | [cli](cli/README.md#tunnel-management) · [sdk](go-sdk/README.md#quick-start) |
| Quota query & usage | ✅ | — | [cli](cli/README.md#quota-query) |
| Debugging tools (HTTP echo, URI ping) | ✅ | — | [cli](cli/README.md#debugging-tools) |
| Auth — SSO browser / API Key / env var / direct JWT | ✅ | ✅ | [cli](cli/README.md#login) · [sdk](go-sdk/README.md#client-configuration) |
| Auto-reconnect (backoff + SSH session resumption) | ✅ | ✅ | [cli](cli/README.md#features) |
| Bilingual output (EN default, `DEVBRIDGE_LANG=zh`) | ✅ | — | [cli](cli/README.md#language-switching) |

## Repository Structure

```text
.
├── .github/              # CI, release and mirror-sync workflows
├── cli/                  # DevBridge CLI — Go 1.26, end-user tool
├── go-sdk/               # DevBridge Go SDK — Go 1.22, integration API
├── docs/                 # VitePress documentation site (Node.js toolchain)
├── LICENSE               # Apache License 2.0
└── README.md             # repository overview (this file)
```

Each component lives in its own top-level directory with its own build config and dependencies,
avoiding toolchain conflicts.

| Component | Description | Details |
| --- | --- | --- |
| CLI | `devbridge` remote tunnel port-forwarding CLI (Go 1.26) | [cli/README.md](cli/README.md) |
| Go SDK | CLI core capabilities as a Go API for integration (Go 1.22) | [go-sdk/README.md](go-sdk/README.md) |
| Docs | VitePress site, published to GitHub Pages | [docs/README.md](docs/README.md) |

## Documentation

The documentation site source lives in `docs/` and is published to GitHub Pages:
<https://huaweicloud.github.io/devspace-devbridge/>. Working on the site (local preview, adding
pages, dependency policy, publishing) is documented in [docs/README.md](docs/README.md).

Key pages:

| Topic | Page |
| --- | --- |
| Overview | [guide/overview](https://huaweicloud.github.io/devspace-devbridge/guide/overview) |
| Install & quick start | [guide/install](https://huaweicloud.github.io/devspace-devbridge/guide/install) · [guide/quickstart](https://huaweicloud.github.io/devspace-devbridge/guide/quickstart) |
| Host / Connect | [guide/host](https://huaweicloud.github.io/devspace-devbridge/guide/host) · [guide/connect](https://huaweicloud.github.io/devspace-devbridge/guide/connect) |
| Tunnels / Ports | [guide/tunnels](https://huaweicloud.github.io/devspace-devbridge/guide/tunnels) · [guide/ports](https://huaweicloud.github.io/devspace-devbridge/guide/ports) |
| Authentication | [guide/authentication](https://huaweicloud.github.io/devspace-devbridge/guide/authentication) |
| CLI / API reference | [reference/cli](https://huaweicloud.github.io/devspace-devbridge/reference/cli) · [reference/api](https://huaweicloud.github.io/devspace-devbridge/reference/api) |
| Troubleshooting | [reference/troubleshooting](https://huaweicloud.github.io/devspace-devbridge/reference/troubleshooting) |

## Contributing

Contributions are welcome. See the
[contributing guide](https://huaweicloud.github.io/devspace-devbridge/reference/contributing)
(source: [docs/reference/contributing.md](docs/reference/contributing.md)) for the branch model,
code style, CI and release flow.

## Disclaimer

DevBridge is a development/testing tunneling service provided "as is" without an SLA — do not use
it in production or for critical data; you are responsible for securing exposed services and must
comply with applicable laws and your organization's requirements. Full terms:
[disclaimer](https://huaweicloud.github.io/devspace-devbridge/reference/disclaimer).

## License

[Apache License 2.0](LICENSE).
