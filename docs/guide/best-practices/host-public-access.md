---
title: Hosting and public access
description: Start a local service, host it with Host, and access the tunnel URL directly in a browser.
---

# Hosting and public access

<p class="lead">Start a service locally, host the port with Host to generate a public access URL, then simply open the URL in a browser.</p>

## Video demos

<VideoPlayer
  src="https://tools-artifact.developer.huaweicloud.com/sharedata/devbridge/video/ci-best-practice.mp4"
  poster="/images/videos/ci-best-practice-video.png"
  title="CI best practices demo"
  :caption="false"
/>

<VideoPlayer
  src="https://tools-artifact.developer.huaweicloud.com/sharedata/devbridge/video/ai-developer-space-best-practice.mp4"
  poster="/images/videos/ai-developer-space-best-practice-video.png"
  title="AI Developer Space best practices demo"
  :caption="false"
/>

## Scenario overview

Goal: start an HTTP service locally, host it through a DevBridge tunnel, get a public access URL, and access it directly in a browser.

You don't need to install the CLI or establish a Connect connection on another device — this is ideal for quickly sharing local development content.

## Step 1: Start the local service

In terminal 1, start an HTTP service listening on port `8080`:

```bash
python3 -m http.server 8080
```

Once started, this terminal keeps running; the service listens on `0.0.0.0:8080` by default. Keep this terminal open.

![Start the local HTTP service](/images/host-connect/python-http-server.png)

Verify the service works:

```bash
curl http://127.0.0.1:8080
```

If it returns a listing of the files in the current directory, the service is working.

## Step 2: Start Host to host the port

There are two ways to run Host, depending on whether you reuse an existing tunnel.

### Option 1: Use a temporary tunnel

If you don't have a reusable tunnel, let Host automatically create a temporary tunnel and host port `8080`:

```bash
devbridge host -p 8080
```

To set a description and expiration:

```bash
devbridge host -p 8080 -d "tunnel description" -e 8
```

- `-p 8080`: hosts local port `8080`.
- `-d`: tunnel description, optional.
- `-e`: expiration in hours, optional, defaults to 72 hours.

After Host succeeds, it prints the tunnel ID and access URL, in the following format:

```text
https://<tunnelId>-<port>.devbridge-s2.hwtunnel.com
```

![Start Host with a temporary tunnel](/images/host-connect/host-temporary-tunnel.png)

The image above shows the actual output of `devbridge host -p 8080`: it automatically creates tunnel `yf7oqwea`, hosts port `8080`, and prints the corresponding tunnel URL.

::: tip Temporary tunnel lifecycle
A tunnel created by `devbridge host -p 8080` is retained after you stop Host — it isn't deleted automatically. Clean it up with `devbridge delete <tunnelId>` when you no longer need it.
:::

### Option 2: Use an existing tunnel

Before reusing an existing tunnel, confirm that it exists and that port `8080` is already configured.

1. List the valid tunnels in the current workspace:

   ```bash
   devbridge list
   ```

   Confirm the target tunnel ID is in the list.

2. View the tunnel's port configuration:

   ```bash
   devbridge port list <tunnelId>
   ```

   Confirm port `8080` already exists. If it doesn't, create it first:

   ```bash
   devbridge port create <tunnelId> -p 8080 --protocol http
   ```

3. Start Host to host all the ports of that tunnel:

   ```bash
   devbridge host <tunnelId>
   ```

   Host loads all the ports configured in the tunnel and begins forwarding; you don't need to specify `-p` again.

   ![Start Host with an existing tunnel](/images/host-connect/host-existing-tunnel.png)

   The image above shows the actual output of `devbridge host ghirszae`: Host successfully hosts port `8080` and prints the tunnel URL; during the run it triggers an automatic reconnection and resumes forwarding after recovering.

Either way, Host keeps running in the foreground. Keep terminal 2 open; press `Ctrl+C` to stop hosting.

## Step 3: Access the tunnel URL in a browser

The tunnel URL that Host prints looks like:

```text
https://<tunnelId>-<port>.devbridge-s2.hwtunnel.com
```

Open the URL directly in a browser on any device to access the local service. The behavior depends on the port's anonymous-access policy:

| Port policy                       | Browser behavior                                                                                            |
| --------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| Allows anonymous access (`-a`)    | Open the URL directly to access it — no DevBridge identity or credentials required.                         |
| Denies anonymous access (default) | You're redirected to a sign-in page; complete authentication to obtain credentials, then you can access it. |

A port's anonymous-access policy is set at creation time with `-a` or `--deny-anonymous`. See [Manage ports](../ports.md).

## Quick reference

| Step | Terminal   | Command                                               | Description                              |
| ---- | ---------- | ----------------------------------------------------- | ---------------------------------------- |
| 1    | Terminal 1 | `python3 -m http.server 8080`                         | Start the local HTTP service.            |
| 2    | Terminal 2 | `devbridge host -p 8080`                              | Host with a temporary tunnel (Option 1). |
| 2    | Terminal 2 | `devbridge host <tunnelId>`                           | Host with an existing tunnel (Option 2). |
| 3    | Browser    | `https://<tunnelId>-<port>.devbridge-s2.hwtunnel.com` | Access the tunnel URL directly.          |

## FAQ

### Host exits immediately after starting

Confirm the local service in terminal 1 is still running. Host exits with an error when it detects that nothing is listening on the local port.

### Browser access returns a sign-in page

When the port is configured to deny anonymous access, the browser redirects to a sign-in page. Complete authentication to access it, or use `-a` when creating the port to allow anonymous access.

### Forgot the tunnel ID

To look it up:

```bash
devbridge list
```

Both temporary and persistent tunnels appear in the valid tunnel list.

## Cleanup

After verification, stop things in this order:

1. Press `Ctrl+C` in the Host terminal.
2. Press `Ctrl+C` in the service terminal to stop `python3 -m http.server`.

A temporary tunnel isn't deleted automatically after you stop Host. Clean it up once you're sure you no longer need it:

```bash
devbridge delete <tunnelId>
```

## Related content

- [Hosting and remote connection](./host-remote-connect.md)
- [Host: expose local services](../host.md)
- [Manage ports](../ports.md)
- [What is DevBridge](../overview.md)
- [Troubleshoot Host connection issues](../../reference/troubleshooting.md)
