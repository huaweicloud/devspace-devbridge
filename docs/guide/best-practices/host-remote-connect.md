---
title: Hosting and remote connection
description: Host a service locally, then connect and access it from a remote device through Connect.
---

# Hosting and remote connection

<p class="lead">Start and host a service on device A, set up local port mappings on remote device B through Connect, and access the remote service via localhost.</p>

## Video demo

<VideoPlayer
  src="https://tools-artifact.developer.huaweicloud.com/sharedata/devbridge/video/devops-best-practice.mp4"
  poster="/images/videos/devops-best-practice-video.png"
  title="DevOps best practices demo"
  :caption="false"
/>

## Scenario overview

Goal: start an HTTP service on device A, host it through a DevBridge tunnel, connect to the tunnel from device B, and access device A's service at `http://localhost:8080` on device B.

| Device   | Role    | Responsibility                                                       |
| -------- | ------- | -------------------------------------------------------------------- |
| Device A | Host    | Runs the local service and hosts its ports with DevBridge.           |
| Device B | Connect | Connects to the tunnel and sets up local port mappings to access it. |

Both devices must have the DevBridge CLI installed and be logged in. If not, see [Install the DevBridge CLI](../install.md).

## Step 1: Start the local service on device A

In terminal 1 on device A, start an HTTP service listening on port `8080`:

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

## Step 2: Start Host on device A

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

Note the `<tunnelId>` here; device B needs it to connect.

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

## Step 3: Start Connect on device B

On device B, run the following command to connect to the tunnel:

```bash
devbridge connect <tunnelId>
```

Replace `<tunnelId>` with the tunnel ID that Host printed in step 2.

Connect reads the tunnel's port configuration, requests a Connect token, and sets up local port mappings. After the connection succeeds, port `8080` on device B maps to the service on device A.

Connect also keeps running in the foreground; keep the terminal open, and press `Ctrl+C` to stop the connection.

![Connect to the tunnel](/images/host-connect/connect-tunnel.png)

The image above shows the actual output of `devbridge connect ghirszae`: on success it shows `Forwarding localhost:8080 -> tunnel port: 8080`, meaning device B's local port `8080` is mapped to the tunnel port.

## Step 4: Access the remote service on device B

Once the connection is established, access device A's service on device B through the local port:

```bash
curl http://127.0.0.1:8080
```

![Access the remote service via curl](/images/host-connect/curl-access.png)

The HTML directory listing in the response is served by `python3 -m http.server 8080` on device A, which confirms that device B has successfully reached device A's service through the local port mapping.

You can also open it in a browser:

```text
http://localhost:8080
```

or use `127.0.0.1`:

```text
http://127.0.0.1:8080
```

Both are equivalent: they point to the local port mapping on device B, which DevBridge then forwards to the `8080` service on device A.

## Quick reference

| Step | Device   | Terminal   | Command                        | Description                              |
| ---- | -------- | ---------- | ------------------------------ | ---------------------------------------- |
| 1    | Device A | Terminal 1 | `python3 -m http.server 8080`  | Start the local HTTP service.            |
| 2    | Device A | Terminal 2 | `devbridge host -p 8080`       | Host with a temporary tunnel (Option 1). |
| 2    | Device A | Terminal 2 | `devbridge host <tunnelId>`    | Host with an existing tunnel (Option 2). |
| 3    | Device B | Terminal   | `devbridge connect <tunnelId>` | Connect and set up local mappings.       |
| 4    | Device B | Browser    | `http://localhost:8080`        | Access the remote service.               |

## FAQ

### Host exits immediately after starting

Confirm the local service in terminal 1 is still running. Host exits with an error when it detects that nothing is listening on the local port.

### Connect succeeds but access returns empty or times out

Check these in order:

1. the Host process on device A is still running
2. device A's local service is still reachable: `curl http://127.0.0.1:8080`
3. port `8080` on device B isn't in use by another process
4. both sides use the same tunnel ID

### Port already in use

If `8080` on device B is already in use by another process, Connect can't set up the mapping. Free the port first, or change the local service's listening port and run through the flow again.

### Forgot the tunnel ID

Look it up on device A:

```bash
devbridge list
```

Both temporary and persistent tunnels appear in the valid tunnel list.

## Cleanup

After verification, stop things in this order:

1. Device B: press `Ctrl+C` in the Connect terminal.
2. Device A: press `Ctrl+C` in the Host terminal.
3. Device A: press `Ctrl+C` in the service terminal to stop `python3 -m http.server`.

A temporary tunnel isn't deleted automatically after you stop Host. Clean it up once you're sure you no longer need it:

```bash
devbridge delete <tunnelId>
```

## Related content

- [Hosting and public access](./host-public-access.md)
- [Host: expose local services](../host.md)
- [Connect: connect to remote services](../connect.md)
- [What is DevBridge](../overview.md)
- [Troubleshoot Host and Connect issues](../../reference/troubleshooting.md)
