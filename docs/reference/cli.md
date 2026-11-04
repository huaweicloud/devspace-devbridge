---
title: CLI command reference
description: A quick reference for the DevBridge CLI's authentication, tunnel, port, Host, and Connect commands.
---

# CLI command reference

<p class="lead">This page summarizes the main DevBridge CLI commands. Use command-level <code>--help</code> to see the full set of parameters for your installed version.</p>

## Global commands

| Command                      | Description                                    |
| ---------------------------- | ---------------------------------------------- |
| `devbridge --help`           | Display CLI help.                              |
| `devbridge version`          | Display the CLI version.                       |
| `devbridge <command> --help` | Display the parameters for a specific command. |

### Global flags

| Flag              | Description                                     |
| ----------------- | ----------------------------------------------- |
| `--help`, `-h`    | Display help for the current command.           |
| `--verbose`, `-v` | Output more detailed logs to help troubleshoot. |

## Authentication

| Command                                | Description                          |
| -------------------------------------- | ------------------------------------ |
| `devbridge auth login`                 | Interactive login (opens a browser). |
| `devbridge auth login --api-key <key>` | Log in with an API key.              |
| `devbridge auth status`                | View the current login status.       |
| `devbridge auth logout`                | Clear local login credentials.       |

### Login flags

| Flag        | Description                                            |
| ----------- | ------------------------------------------------------ |
| `--api-key` | API key; skip browser interaction and log in directly. |

The CLI also reads the `HW_API_KEY` environment variable automatically.

## Tunnels

| Command                                 | Description                                      |
| --------------------------------------- | ------------------------------------------------ |
| `devbridge create <name>`               | Create a tunnel.                                 |
| `devbridge list`                        | List the valid tunnels in the current workspace. |
| `devbridge show <tunnelId>`             | View a tunnel's details.                         |
| `devbridge update <tunnelId>`           | Update a tunnel.                                 |
| `devbridge delete <tunnelId>`           | Delete a tunnel.                                 |
| `devbridge delete-all`                  | Delete all tunnels in the current workspace.     |
| `devbridge token <tunnelId> -s host`    | Issue a new Host token.                          |
| `devbridge token <tunnelId> -s connect` | Issue a new Connect token.                       |
| `devbridge set <tunnelId>`              | Set the local default tunnel.                    |
| `devbridge unset`                       | Clear the local default tunnel.                  |

### Tunnel flags

| Flag | Applies to                                     | Description                       |
| ---- | ---------------------------------------------- | --------------------------------- |
| `-n` | `update`                                       | Update the tunnel name.           |
| `-d` | `create`, `update`, `host` without a tunnel ID | Set the tunnel description.       |
| `-e` | `create`, `update`, `host` without a tunnel ID | Set the expiration, in hours.     |
| `-s` | `token`                                        | Token scope: `host` or `connect`. |

## Ports

| Command                                                            | Description                         |
| ------------------------------------------------------------------ | ----------------------------------- |
| `devbridge port create <tunnelId> -p <port> --protocol <protocol>` | Create a port.                      |
| `devbridge port list <tunnelId>`                                   | List tunnel ports.                  |
| `devbridge port show <tunnelId> -p <port>`                         | View a port's details.              |
| `devbridge port update <tunnelId> -p <port>`                       | Update the anonymous-access policy. |
| `devbridge port delete <tunnelId> -p <port>`                       | Delete a port.                      |

### Port flags

| Flag                      | Description                                                                 |
| ------------------------- | --------------------------------------------------------------------------- |
| `-p`, `--port-number`     | Port number, from `1` to `65535`.                                           |
| `--protocol`              | `http`, `https`, or `auto`. Available on creation only; defaults to `auto`. |
| `--deny-anonymous`        | Deny anonymous access on create or update.                                  |
| `-a`, `--allow-anonymous` | Allow anonymous access on create or update.                                 |

The current CLI doesn't support changing an existing port's protocol. Port commands don't support the `-d` description flag, and there's no bulk-delete command for ports.

## Host

| Command                                                | Description                                            |
| ------------------------------------------------------ | ------------------------------------------------------ |
| `devbridge host <tunnelId>`                            | Host all the ports configured in an existing tunnel.   |
| `devbridge host -p <port> -d <description> -e <hours>` | Create a tunnel and host its ports immediately.        |
| `devbridge host <tunnelId> --token <jwt>`              | Host with an existing JWT token, skipping API calls.   |
| `devbridge host <tunnelId> --api-key <key>`            | Authenticate with an API key, skipping token issuance. |

Host is a long-running foreground command. `-d` and `-e` apply to the new tunnel only when Host creates one at the same time.

### Host flags

| Flag                  | Description                                                                          |
| --------------------- | ------------------------------------------------------------------------------------ |
| `-p`, `--ports`       | Local port number(s); can pass multiple. Used only when creating a temporary tunnel. |
| `-d`, `--description` | Description of the new tunnel.                                                       |
| `-e`, `--expiration`  | Expiration of the new tunnel, in hours.                                              |
| `-t`, `--token`       | Provide a JWT token directly, skipping token issuance and port lookup.               |
| `-k`, `--api-key`     | Authenticate with an API key, skipping token issuance, via `X-API-Key`.              |

## Connect

| Command                                        | Description                                             |
| ---------------------------------------------- | ------------------------------------------------------- |
| `devbridge connect <tunnelId>`                 | Connect to a tunnel and set up local port mappings.     |
| `devbridge connect <tunnelId> --token <jwt>`   | Connect with an existing JWT token, skipping API calls. |
| `devbridge connect <tunnelId> --api-key <key>` | Authenticate with an API key, skipping token issuance.  |

### Connect flags

| Flag              | Description                                                             |
| ----------------- | ----------------------------------------------------------------------- |
| `-t`, `--token`   | Provide a JWT token directly, skipping token issuance and port lookup.  |
| `-k`, `--api-key` | Authenticate with an API key, skipping token issuance, via `X-API-Key`. |

## Quota query

| Command            | Description                           |
| ------------------ | ------------------------------------- |
| `devbridge limits` | View account quota and current usage. |

The output includes:

- reset time
- traffic quota and traffic used
- the number of active tunnels
- limits on the number of tunnels, ports, and Hosts
- tunnel bandwidth limit
- per-port HTTP request-rate limit
- per-port connection limit

## Debug tools

### echo

Starts an HTTP echo service for verifying that the tunnel link works.

| Command                             | Description                                                        |
| ----------------------------------- | ------------------------------------------------------------------ |
| `devbridge echo`                    | Start the echo service on a random port, listening on `127.0.0.1`. |
| `devbridge echo -p 8080`            | Specify the listening port.                                        |
| `devbridge echo -p 8080 -i 0.0.0.0` | Specify the listening port and address.                            |

### ping

Sends HTTP ping probes to a URI to check the connectivity and latency of a tunnel URL.

| Command                       | Description                                     |
| ----------------------------- | ----------------------------------------------- |
| `devbridge ping <uri>`        | Probe a URI with a default interval of 1000 ms. |
| `devbridge ping <uri> -i 500` | Specify the probe interval in milliseconds.     |

Examples:

```bash
devbridge ping https://<tunnelId>-8080.devbridge-s2.hwtunnel.com
devbridge ping http://127.0.0.1:8080 -i 3000
```

Output looks like:

```text
HTTP 200 OK -- 4 ms
```

### Debug logging

Append `-v` / `--verbose` to any command to enable debug-level logging:

```bash
devbridge -v host <tunnelId>
devbridge connect <tunnelId> --verbose
```

## Completion

| Command                                              | Description                                                         |
| ---------------------------------------------------- | ------------------------------------------------------------------- |
| `devbridge completion [bash\|zsh\|fish\|powershell]` | Generate a shell auto-completion script (a built-in Cobra command). |

## Related content

- [Manage tunnels](../guide/tunnels.md)
- [Manage ports](../guide/ports.md)
- [Host: expose local services](../guide/host.md)
- [Connect: connect to remote services](../guide/connect.md)
