---
title: Troubleshooting
description: Troubleshoot DevBridge installation, login, tunnel, Host, and Connect issues.
---

# Troubleshooting

<p class="lead">First confirm the local command, identity, tunnel status, and port configuration, then check the network connectivity of Host and Connect.</p>

## devbridge command not found

Confirm the default install directory exists and add it to the current terminal's `PATH`:

```bash
export PATH="$HOME/.huawei/bin:$PATH"
devbridge version
```

If you use PowerShell, run:

```powershell
$env:Path = "$HOME/.huawei/bin;$env:Path"
devbridge version
```

If it still fails, rerun the [official installation command](../guide/install.md) and check the output.

## Install script fails to download

Check:

1. whether the current network can reach the install address
2. whether DNS and the HTTPS proxy work
3. if you're using the Bash method, whether `curl` is available
4. if you're using the PowerShell method, whether `irm` is available
5. whether the install address is complete and not wrapped or truncated

Don't switch to an install script of unknown origin when a download fails.

## Login fails or credentials expire

First check the status:

```bash
devbridge auth status
```

Then log in again:

```bash
devbridge auth logout
devbridge auth login
```

When logging in with an API key, confirm the key is valid and hasn't expired. When passing it via an environment variable, check that `HW_API_KEY` is set correctly.

## Tunnel not found

```bash
devbridge list
devbridge show <tunnelId>
```

Common causes:

- tunnel ID entered incorrectly
- the current login identity doesn't belong to the tunnel's workspace
- the tunnel has expired or been deleted
- the current environment is connected to a different cluster or region

## Can't create more tunnels

Each workspace holds up to 10 valid tunnels by default. Run:

```bash
devbridge list
```

Reuse an existing tunnel, or delete tunnels you no longer need.

## Host can't connect

Confirm in order:

1. `devbridge auth status` shows valid credentials
2. `devbridge show <tunnelId>` can read the tunnel
3. `devbridge port list <tunnelId>` includes the ports to host
4. the local service is actually listening on that port
5. the current network can reach DevBridge

Check the local port:

```bash
curl -v http://127.0.0.1:8080
```

If the port uses `https`, check over HTTPS and confirm the certificate and local service configuration are correct.

## Connect can't access the service

Confirm:

- the Host process is still running
- Connect and Host use the same tunnel
- the port still exists
- the current identity can access the non-anonymous port
- the local port on the Connect device isn't in use
- the port protocol matches the local service on the Host side

Connect's automatic reconnection only handles brief network interruptions; it can't recover tunnels that have been deleted or expired.

## Incorrect HTTP and HTTPS behavior

View the port protocol:

```bash
devbridge port show <tunnelId> -p 8080
```

If the local service is plain HTTP, the protocol should be `http`; use `https` when the local service handles TLS itself; use `auto` only when auto-detection is genuinely needed.

The current CLI doesn't support changing an existing port's protocol directly. To change it, delete the port and recreate it with the correct protocol:

```bash
devbridge port delete <tunnelId> -p 8080
devbridge port create <tunnelId> -p 8080 --protocol http
```

## Still unresolved

Collect the following non-sensitive information:

- `devbridge version`
- operating system and architecture
- the commands you ran, excluding API keys and tokens
- tunnel ID, port, and protocol
- the time the error occurred
- the error message, already redacted

Don't submit full credentials, JWTs, the configuration directory, or terminal transcripts containing secrets.
