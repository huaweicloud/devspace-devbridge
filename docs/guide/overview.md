---
title: What is DevBridge
description: Understand the relationships between DevBridge tunnels, ports, Host, and Connect.
---

# What is DevBridge

<p class="lead">DevBridge connects Host and Connect through a relay service, so your local services don't need to accept inbound connections directly from the public internet.</p>

DevBridge is suited to remote integration testing, sharing web pages under development, receiving webhooks, and accessing local services from another device.

## Video

<VideoPlayer
  src="https://tools-artifact.developer.huaweicloud.com/sharedata/devbridge/video/promotion-video.mp4"
  poster="/images/videos/promotion-video.png"
  title="DevBridge promotional video"
  :caption="false"
/>

## How it works

A complete connection involves the following parts:

1. **Tunnel** stores the name, description, expiration time, assigned cluster, and stable access URL.
2. **Port policy** declares the allowed ports, protocols, and anonymous-access rules.
3. **Host** establishes an outbound connection to the relay service from the device running the local service, and forwards local ports.
4. **Connect** connects to the same tunnel from the device accessing it, and sets up local port mappings.

Both Host and Connect initiate outbound connections to DevBridge, so the device running Host typically doesn't need to open public inbound ports.

## Core concepts

| Concept       | Description                                                                          |
| ------------- | ------------------------------------------------------------------------------------ |
| Tunnel ID     | An eight-character lowercase Base32 identifier used in CLI commands and tunnel URLs. |
| Cluster       | The isolation and routing scope the tunnel belongs to.                               |
| Port          | A port the tunnel is allowed to forward, from `1` to `65535`.                        |
| Protocol      | The application protocol for a port: `http`, `https`, or `auto`.                     |
| Expiration    | How long the tunnel stays valid; once it expires, it can no longer be used.          |
| Host          | The side that hosts local services.                                                  |
| Connect       | The side that accesses remote services.                                              |
| Host token    | A short-lived credential that only allows establishing Host connections.             |
| Connect token | A short-lived credential that only allows establishing Connect connections.          |

## Key capabilities

- A tunnel can be configured with multiple ports.
- Host can host multiple local ports at the same time.
- Connect can set up the corresponding local port mappings.
- Ports can deny anonymous access on demand.
- Login credentials are stored in a protected local credential store.
- Host and Connect support automatic reconnection after network interruptions.
- Persistent tunnels can reuse the same tunnel ID and access URL.

## Isolation and access

Tunnels belong to the workspace associated with the currently logged-in identity. The CLI only shows tunnels that are still valid within that workspace.

Each tunnel also belongs to a cluster. Host, Connect, and the gateway must use the same cluster as the tunnel to avoid cross-cluster access.

Whether a port allows anonymous access is determined by its port policy. Admin consoles, debugging endpoints, and services that handle user data should use `--deny-anonymous`.

## Usage boundaries

DevBridge is intended for development and testing workflows. When using it, follow these boundaries:

- Don't transfer unnecessary production data through the tunnel.
- Don't enable anonymous access for management interfaces.
- Don't store API keys or tunnel tokens in scripts, logs, or code repositories.
- Don't treat short-lived tokens as long-term identity credentials.
- When you're done sharing, stop Host and delete any tunnels you no longer use.

## Next steps

- [Install the DevBridge CLI](./install.md)
- [Create and host your first tunnel](./quickstart.md) (skip ahead if already installed)
