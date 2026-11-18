---
title: DevBridge
description: "The entry point to DevBridge: a five-minute quickstart, core concepts, and the command reference."
---

# DevBridge

<p class="lead">Use DevBridge to securely expose your local development services to remote devices without opening any public inbound ports, and control how each port is accessed.</p>

::: warning 📢 Notice to users
DevBridge will release a new version on **Thursday, November 19, 19:00–23:00 Beijing time (UTC+8)**. During the release window, the service may experience brief connection interruptions. Please be aware in advance and plan accordingly.

**Impact**

1. Tunnel connections may drop briefly; the system will reconnect automatically
2. Long-running tasks or interactive sessions in progress may be interrupted and need to be restarted
3. Visitor URLs may be briefly unreachable

**Recommendations**

1. Avoid scheduling critical long-running tasks during the release window
2. Complete any important operations before the release if possible
3. If you experience connection issues after the release, wait a moment and reconnect — it usually recovers automatically

For questions or coordination, please [report an issue](https://github.com/huaweicloud/devspace-devbridge/issues).
:::

## Getting started

- [Create and host your first tunnel in five minutes](./guide/quickstart.md) — walks you through installing, logging in, hosting, and connecting
- [Learn how DevBridge works](./guide/overview.md) — the relationship between tunnels, ports, Host, and Connect — with an introductory video
- [CLI command reference](./reference/cli.md) — every command and its arguments

## Popular pages

- [Manage tunnels](./guide/tunnels.md)
- [Manage ports](./guide/ports.md)
- [Host: expose local services](./guide/host.md)
- [Connect: connect to remote services](./guide/connect.md)
- [Best practices](./guide/best-practices/host-public-access.md)
- [Troubleshooting](./reference/troubleshooting.md)
