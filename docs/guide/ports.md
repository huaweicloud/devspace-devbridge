---
title: Manage ports
description: Configure tunnel ports, protocols, and anonymous-access policies.
---

# Manage ports

<p class="lead">A tunnel only forwards ports that have been configured. Each port has its own protocol and anonymous-access policy.</p>

## Create a port

Create an HTTP port that explicitly denies anonymous access (the default):

```bash
devbridge port create <tunnelId> -p 8080 --protocol http --deny-anonymous
```

Create an HTTP port that allows anonymous access:

```bash
devbridge port create <tunnelId> -p 8080 --protocol http -a
```

Create a port that uses the default `auto` protocol:

```bash
devbridge port create <tunnelId> -p 3000
```

You must provide a port when creating; the protocol defaults to `auto`. Ports range from `1` to `65535` and cannot be duplicated within the same tunnel.

## Choose a protocol

| Protocol | Use case                                                |
| -------- | ------------------------------------------------------- |
| `http`   | The local service receives plain HTTP.                  |
| `https`  | The local service itself serves HTTPS.                  |
| `auto`   | The connecting side detects the protocol automatically. |

The protocol declares the local service on the Host side. Choosing the wrong one can cause handshake failures, dropped connections, or unrecognizable data in responses.

## Control anonymous access

`-a` and `--allow-anonymous` mean the port can be accessed anonymously:

```bash
devbridge port create <tunnelId> -p 8080 --protocol http -a
```

`--deny-anonymous` means the port requires authentication:

```bash
devbridge port create <tunnelId> -p 8080 --protocol http --deny-anonymous
```

The following services should always deny anonymous access:

- admin consoles
- debugging and diagnostic endpoints
- services that contain user data
- interfaces capable of modifying data

::: warning
Anonymous access means a visitor who has the tunnel URL can access the port without a DevBridge identity. Enable it only for development content that clearly needs to be public.
:::

### Browser access behavior

When you open the tunnel URL `https://<tunnelId>-<port>.devbridge-s2.hwtunnel.com` directly in a browser, the behavior depends on the port's anonymous-access policy:

- **Allows anonymous access**: you can access it directly — no login or credentials required.
- **Denies anonymous access**: you're redirected to a sign-in page; complete authentication to obtain credentials, then you can access it.

You can access the tunnel URL directly through a browser without installing the CLI locally or establishing a Connect connection. See
[Host: expose local services](./host.md#access-the-tunnel-url).

## List ports

```bash
devbridge port list <tunnelId>
```

If a default tunnel is set, you can omit the tunnel ID:

```bash
devbridge port list
```

## View a port

```bash
devbridge port show <tunnelId> -p 8080
```

The result includes the tunnel ID, port, protocol, and anonymous-access status.

## Update a port

Change a port to deny anonymous access:

```bash
devbridge port update <tunnelId> -p 8080 --deny-anonymous
```

Change a port to allow anonymous access:

```bash
devbridge port update <tunnelId> -p 8080 -a
```

The update command only changes the anonymous-access policy. The current CLI doesn't support changing an existing port's protocol directly; to change it, delete the port and recreate it with the new protocol.

## Delete a port

```bash
devbridge port delete <tunnelId> -p 8080
```

There's no bulk-delete command for ports. To remove a whole tunnel and its ports, delete the tunnel.

## Next steps

- [Use Host to forward local ports](./host.md)
- [Use Connect to set up local mappings](./connect.md)
