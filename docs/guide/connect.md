---
title: "Connect: connect to remote services"
description: Use Connect mode to access remote ports in a DevBridge tunnel.
---

# Connect: connect to remote services

<p class="lead">Connect runs on the device that accesses the service and sets up local mappings for the remote ports in the tunnel.</p>

## Establish a connection

On a device where you've installed the DevBridge CLI and logged in, run:

```bash
devbridge connect <tunnelId>
```

Connect reads the tunnel's port configuration, requests a new Connect token, and connects to the currently available Host.

Once the connection is established, access the remote service through the same port on your local machine. For example, if the tunnel has port `8080`:

```text
http://localhost:8080
```

## Connect to multiple ports

A tunnel can be configured with multiple ports. Connect sets up the corresponding mappings based on the tunnel's port configuration; you don't need to use a `--port` flag to filter ports.

View the list of remote ports:

```bash
devbridge port list <tunnelId>
```

## Use an existing token

When integrating other clients or skipping API calls, you can provide a JWT token directly:

```bash
devbridge connect <tunnelId> --token <jwt>
```

The `--token` mode skips token issuance and port lookup; the Host side negotiates and delivers the ports through the relay channel. You must specify the tunnel ID.

## Authenticate with an API key

Skip token issuance and authenticate directly with an API key over WebSocket:

```bash
devbridge connect <tunnelId> --api-key <key>
```

This mode still queries the tunnel's port configuration through the API, but skips the token-issuance step.

## Protected ports

When a port is configured with `--deny-anonymous`, Connect must use a valid identity, along with a Connect token. The CLI requests the token automatically, so you usually don't need to run `devbridge token` manually.

A Host token can't be used for Connect, and a Connect token can't be used for Host.

## Automatic reconnection

Connect is a long-running foreground command. It automatically tries to reconnect after brief network interruptions; press `Ctrl+C` to stop.

The following cases require manual handling:

- login credentials have expired
- the tunnel has expired or been deleted
- Host is not currently running
- the port has been deleted or its protocol has changed
- the local port is already in use by another process

## Next steps

- [Manage port access policies](./ports.md)
- [Troubleshoot connection issues](../reference/troubleshooting.md)
