---
title: Create and host a tunnel
description: Install the DevBridge CLI, log in, host a local service through DevBridge, and then connect from another device.
---

# Create and host a tunnel

<p class="lead">In this quickstart, you'll host port `8080` on your local machine, then connect to the tunnel from another device.</p>

## Prerequisites

- Install the DevBridge CLI and confirm that `devbridge version` runs properly. See [Install the DevBridge CLI](./install.md).
- Log in: use interactive login for personal environments and an API key for automated environments. See [Login and credentials](./authentication.md).
- Have a second device ready to verify Connect mode (it must also have the CLI installed and be logged in).

::: info When to use DevBridge
DevBridge is for development, integration testing, and temporary sharing. Don't use it as a long-term way to expose production services.
:::

## Host a local service

### 1. Start a test service

Start a local HTTP server in terminal 1:

```bash
python3 -m http.server 8080
```

### 2. Start Host

In terminal 2, create a tunnel that's valid for 8 hours and host the local port:

```bash
devbridge host -p 8080 -e 8
```

After Host starts successfully, it prints the tunnel ID and access URL. The tunnel URL uses the following format:

```text
https://<tunnelId>-<port>.devbridge-s2.hwtunnel.com
```

Keep the Host process running. If the network drops briefly, the CLI automatically tries to reconnect; press `Ctrl+C` to end this hosting session.

::: tip Reuse a tunnel
When you need a fixed address or want to configure multiple ports, first create a persistent tunnel, then use
`devbridge host <tunnelId>` to host all the ports configured for it. See [Manage tunnels](./tunnels.md).
:::

## Connect from another device

On the other device, run:

```bash
devbridge connect <tunnelId>
```

Connect reads the tunnel's port configuration and sets up local mappings. Once the connection is established, access the service on the Host device through the corresponding local port:

```text
http://localhost:8080
```

Keep the Connect process running; press `Ctrl+C` to stop the connection. For details about multiple ports and reconnection behavior, see
[Connect: connect to remote services](./connect.md).

## Next steps

- [Create and manage persistent tunnels](./tunnels.md)
- [Configure port protocols and anonymous access](./ports.md)
- [Best practices: hosting and public access](./best-practices/host-public-access.md)
- [View the CLI command reference](../reference/cli.md)
