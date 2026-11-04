---
title: Quota query and debug tools
description: Use limits to query quota, and echo and ping to debug the tunnel link.
---

# Quota query and debug tools

<p class="lead">The DevBridge CLI provides quota query and link debugging commands to help you confirm resource usage and troubleshoot connection issues.</p>

## View quota and usage

Use `devbridge limits` to view the current account's quota and usage:

```bash
devbridge limits
```

The output includes:

| Item                             | Description                                                |
| -------------------------------- | ---------------------------------------------------------- |
| Reset time                       | The point in time when quota metering resets.              |
| Traffic quota / traffic used     | The traffic limit and current usage of tunnel forwarding.  |
| Active tunnels                   | The number of valid tunnels in the current workspace.      |
| Tunnel / port / Host limits      | The maximum allowed count for each resource.               |
| Tunnel bandwidth limit           | The bandwidth limit of a single tunnel.                    |
| Per-port HTTP request-rate limit | The HTTP request rate allowed for a single port.           |
| Per-port connection limit        | The concurrent connection count allowed for a single port. |

::: tip When to check quotas

- When creating a tunnel or port fails, check whether you've hit the count limit.
- When Host or Connect traffic looks abnormal, check whether you've hit the traffic or bandwidth limit.
- When requests are rate-limited, check whether you've hit the request-rate limit.

:::

## Use echo to verify the link

`devbridge echo` starts an HTTP echo service that returns detailed request information, for verifying that the Host and Connect link is working.

### Start the echo service

```bash
# Random port by default, listening on 127.0.0.1
devbridge echo

# Specify the port and listening address
devbridge echo -p 8080 -i 0.0.0.0
```

| Flag | Description                             |
| ---- | --------------------------------------- |
| `-p` | Listening port; random by default.      |
| `-i` | Listening address, default `127.0.0.1`. |

### Verify together with Host

1. Start the echo service in terminal 1:

   ```bash
   devbridge echo -p 8080
   ```

2. Start Host in terminal 2 to host port `8080`:

   ```bash
   devbridge host -p 8080
   ```

3. On another device, access the tunnel URL through Connect or a browser. If you get the echo service's response back, the link is working.

The echo service returns request details (method, path, headers, and so on) rather than a directory listing like `python3 -m http.server`, making it useful for confirming whether a request arrives correctly.

## Use ping to probe connectivity

`devbridge ping` sends HTTP ping probes to a URI and prints the status code and latency, for checking whether a tunnel URL is reachable.

### Basic usage

```bash
# Probe a tunnel URL, default interval 1000 ms
devbridge ping https://<tunnelId>-8080.devbridge-s2.hwtunnel.com

# Specify the probe interval (milliseconds)
devbridge ping https://<tunnelId>-8080.devbridge-s2.hwtunnel.com -i 500
```

You can also probe a local port (for example, after Connect sets up mappings):

```bash
devbridge ping http://127.0.0.1:8080 -i 3000
```

Output looks like:

```text
HTTP 200 OK -- 4 ms
```

| Flag | Description                                   |
| ---- | --------------------------------------------- |
| `-i` | Probe interval in milliseconds, default 1000. |

### Probe scenarios

| Probe target                          | Description                                                  |
| ------------------------------------- | ------------------------------------------------------------ |
| Tunnel public URL                     | Verify that Host is running and the tunnel URL is reachable. |
| `http://127.0.0.1:<port>`             | Verify that Connect's local mapping works.                   |
| `http://127.0.0.1:<port>` (Host side) | Confirm the local service is listening.                      |

Press `Ctrl+C` to stop probing.

## Debug logging

Append `-v` / `--verbose` to any command to enable debug-level logging, which prints diagnostic information such as the request address, retry count, and time taken:

```bash
devbridge -v host <tunnelId>
devbridge connect <tunnelId> --verbose
devbridge -v list
```

`--verbose` doesn't print API keys or tokens in plaintext.

## Related content

- [CLI command reference](../reference/cli.md)
- [Host: expose local services](./host.md)
- [Connect: connect to remote services](./connect.md)
- [Troubleshooting](../reference/troubleshooting.md)
