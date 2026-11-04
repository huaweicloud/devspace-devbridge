---
title: Manage tunnels
description: Create, query, update, and delete DevBridge tunnels, and set a default tunnel.
---

# Manage tunnels

<p class="lead">Persistent tunnels are useful when you need to host services repeatedly, keep a stable address, or pre-configure multiple ports.</p>

## Create a tunnel

```bash
devbridge create frontend -d "tunnel description" -e 24
```

Parameters:

| Parameter  | Description                |
| ---------- | -------------------------- |
| `frontend` | Tunnel name.               |
| `-d`       | Optional description.      |
| `-e`       | Expiration time, in hours. |

If `-e` is omitted, it defaults to 72 hours; the allowed range is 1 to 720 hours.

Expired tunnels don't appear in the list of valid tunnels, and you can no longer issue tokens for them or manage their ports. The system cleans up expired data after a retention period.

On success, the command returns the tunnel ID, name, description, and expiration time.

A workspace can hold up to 10 valid tunnels by default. Once you reach the quota, reuse or delete existing tunnels.

## List tunnels

```bash
devbridge list
```

The list only includes tunnels in the current workspace that haven't been deleted and haven't expired. The list shows:

- tunnel ID and name
- description
- expiration time
- the number of configured ports

## View details

```bash
devbridge show <tunnelId>
```

The details show the tunnel ID, name, description, and expiration time of the specified tunnel, as well as the current Host and Connect connection counts and upload/download traffic statistics. Tunnels that have expired or don't belong to the current workspace are not returned as valid resources.

## Update a tunnel

```bash
devbridge update <tunnelId> -n "frontend-v2" -d "tunnel description" -e 48
```

Pass only the options you want to change. `-e` is still in hours, with a maximum of 720 hours. Updating a tunnel or port refreshes the current expiration time.

## Set a default tunnel

If you often work with the same tunnel, you can save a local default context:

```bash
devbridge set <tunnelId>
```

Some subsequent commands can then omit the tunnel ID:

```bash
devbridge port list
devbridge host -p 8080
```

Clear the default context:

```bash
devbridge unset
```

The default tunnel is stored only on your local machine. It doesn't change tunnel ownership or affect other users.

## Issue connection tokens

Host and Connect usually obtain their corresponding tokens automatically. When integrating other clients, you can issue tokens manually:

```bash
devbridge token <tunnelId> -s host
devbridge token <tunnelId> -s connect
```

Each execution generates a new token. Tokens have a fixed, short validity period that doesn't change with the tunnel's expiration time.

::: warning Protect tokens
Tokens should only be given to the corresponding Host or Connect. Don't write them into logs, URLs, code repositories, or long-term configuration files.
:::

## Delete a tunnel

Delete a tunnel:

```bash
devbridge delete <tunnelId>
```

Delete all tunnels in the current workspace:

```bash
devbridge delete-all
```

`delete-all` affects every tunnel in the current workspace. Deleting a tunnel also removes its port configuration, and any existing Host or Connect connections to that tunnel will stop working.

## Next steps

- [Configure ports for a tunnel](./ports.md)
- [Use Host to forward ports](./host.md)
- [View the full CLI reference](../reference/cli.md)
