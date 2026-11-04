---
title: "Host: expose local services"
description: Use Host mode to connect one or more local ports to DevBridge.
---

# Host: expose local services

<p class="lead">Host runs on the device that provides the service, and registers the local ports with DevBridge through an outbound connection.</p>

## Host an existing tunnel

First, confirm the local service is listening:

```bash
python3 -m http.server 8080
```

Confirm that the corresponding port is configured on the same tunnel by running:

```bash
devbridge port list <tunnelId>
```

Then host the ports of that tunnel:

```bash
devbridge host <tunnelId>
```

Host automatically requests a new Host token, loads the tunnel's full port configuration, connects to the relay service, and begins forwarding. The command keeps running in the foreground; press `Ctrl+C` to stop it.

## Create and host immediately

When you don't have a reusable tunnel, you can have Host create a temporary tunnel:

```bash
devbridge host -p 8080 -d "tunnel description" -e 8
```

Here `-d` and `-e` apply to the new tunnel: they set the tunnel description and the expiration in hours, respectively. They are not port parameters.

If you need a stable address, multiple ports, or explicit access policies, configure them first with `devbridge create` and `devbridge port create`, then start Host.

## Use an existing token

When integrating other clients or skipping API calls, you can provide a JWT token directly:

```bash
devbridge host <tunnelId> --token <jwt>
```

The `--token` mode skips token issuance and port lookup; the gateway delivers the ports through the relay channel. You must specify the tunnel ID.

## Authenticate with an API key

Skip token issuance and authenticate directly with an API key over WebSocket:

```bash
devbridge host <tunnelId> --api-key <key>
```

This mode still queries the tunnel's port configuration through the API, but skips the token-issuance step.

## Access the tunnel URL

After Host succeeds, it prints the tunnel ID and access URL, in the following format:

```text
https://<tunnelId>-<port>.devbridge-s2.hwtunnel.com
```

In addition to the local mappings set up by Connect, you can open the URL directly in a browser to access the hosted service. The access behavior depends on the port's anonymous-access policy:

| Port policy                       | Browser behavior                                                                                            |
| --------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| Allows anonymous access (`-a`)    | Open the URL directly to access it — no DevBridge identity or credentials required.                         |
| Denies anonymous access (default) | You're redirected to a sign-in page; complete authentication to obtain credentials, then you can access it. |

::: tip Two ways to access

- **Access the tunnel URL directly**: open `https://<tunnelId>-<port>.devbridge-s2.hwtunnel.com` in a browser on any device. Best for quickly sharing and verifying.
- **Access via Connect**: run `devbridge connect <tunnelId>` on another device, then access `http://localhost:<port>`. Best when you need local port mappings.

The two ways are equivalent and both reach the service on the Host device. See [Connect: connect to remote services](./connect.md).
:::

A port's anonymous-access policy is set at creation time with `-a` or `--deny-anonymous`. See [Manage ports](./ports.md).

## Automatic reconnection

Host automatically tries to reconnect after brief network interruptions. The following states require manual handling:

- login credentials have expired
- the tunnel has expired or been deleted
- the Host token cannot be reissued
- no service is listening on the local port
- the current network can't reach DevBridge

Automatic reconnection doesn't recreate deleted tunnels or bypass port access policies.

## Stop Host

Press `Ctrl+C` in the Host terminal. Stopping Host only ends the current forwarding:

- persistent tunnels and port configuration are retained
- the Connect side can no longer access ports without a running Host
- run the Host command again to resume hosting

Delete tunnels you no longer use with `devbridge delete <tunnelId>`.

## Security recommendations

- Host only the ports you need for the current integration testing session.
- Confirm the port's anonymous-access policy before hosting.
- Don't print full tokens in Host output.
- Don't run ordinary development services as an administrator or root.
- Stop Host when integration testing is finished.

## Next steps

- [Connect to the tunnel from another device](./connect.md)
- [Troubleshoot Host connection issues](../reference/troubleshooting.md)
