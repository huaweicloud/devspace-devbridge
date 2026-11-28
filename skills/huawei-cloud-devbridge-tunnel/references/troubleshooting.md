# Troubleshooting Guide

This document covers common issues and their solutions when using DevBridge CLI (v0.2.x).

## Table of Contents

1. [Installation Issues](#installation-issues)
2. [Authentication Issues](#authentication-issues)
3. [Tunnel Creation Issues](#tunnel-creation-issues)
4. [Port Management Issues](#port-management-issues)
5. [Host Issues](#host-issues)
6. [Connect Issues](#connect-issues)
7. [Network Issues](#network-issues)
8. [Permission / Auth Failure Issues](#permission--auth-failure-issues)

---

## Installation Issues

### Problem: `command not found: devbridge`

**Cause:** The CLI binary is not in PATH.

**✅ Correct fix:**

```bash
# Check if binary exists
ls -la ~/.huawei/bin/devbridge

# If exists, add to PATH
echo 'export PATH="$HOME/.huawei/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

**❌ Incorrect fix (forgetting to source the config):**

```bash
echo 'export PATH="$HOME/.huawei/bin:$PATH"' >> ~/.bashrc
# Forgot to run 'source ~/.bashrc' — PATH not updated in current session
```

### Problem: `permission denied: devbridge`

**Cause:** The binary lacks execute permission.

**Solution:**

```bash
chmod +x ~/.huawei/bin/devbridge
```

### Problem: Installation script fails with `unbound variable` / `/dev/tty: No such device`

**Cause:** Script detected an existing installation and prompted "overwrite?" reading `/dev/tty`; no terminal exists in headless/CI environments.

**Solution:** Always run the installer with the silent flag:

```bash
curl -fsSL https://gitcode.com/CloudDeveloperDepartment/devbrige/releases/download/latest/install.sh -o /tmp/devbridge_install.sh
bash /tmp/devbridge_install.sh -s
```

### Problem: Version is 0.1.x (old client cannot connect after 2026-09-20)

**Cause:** The installed client predates the tunnel domain switch from `*.cn-north-4-bridge.myhuaweicloud.com` to the new cluster domain.

**Solution:** Reinstall from the official channel (see [CLI Installation Guide](cli-installation-guide.md)) to get v0.2.x, then **re-create tunnels** (old tunnels were invalidated).

### Problem: Installation script fails to download

**Cause:** Network issues or unsupported platform.

**Solution:**

1. Verify network connectivity to the installation source.
2. Check if a proxy is needed: `export https_proxy=<proxy-url>`
3. Verify OS and architecture: `uname -m` (should be `x86_64`/`amd64` or `aarch64`/`arm64`)

---

## Authentication Issues

### Problem: `devbridge auth login` does not open browser

**Cause:** No default browser configured or headless environment.

**Solution:**

1. The CLI will display a URL — copy and open it manually in a browser.
2. For headless environments, use an **API Key**:

```bash
export HW_API_KEY="<your-api-key>"
devbridge auth login --api-key "$HW_API_KEY"
```

### Problem: `Authentication failed: invalid credentials`

**Cause:** Invalid, expired, or deleted API Key.

**Solution:**

1. Re-create an API Key at `https://devstation.connect.huaweicloud.com/space/devbridge?tab=apikey` (full value shown only at creation; up to 20 per scenario).
2. Re-inject via environment variable and re-login:

```bash
export HW_API_KEY="<new-api-key>"
devbridge auth logout
devbridge auth login --api-key "$HW_API_KEY"
```

### Problem: `Authentication failed: access denied`

**Cause:** Credential lacks permission / wrong scenario.

**Solution:**

1. Check auth status: `devbridge auth status`.
2. Delete the old key and create a new DevBridge-scenario API Key at the management page.
3. Re-authenticate after the new key is injected.

**✅ Correct approach:**

```bash
# Check auth status, identify credential issue, re-create key, retry
devbridge auth status
# → Re-create API Key if deleted/expired
# → Inject via HW_API_KEY
devbridge auth login --api-key "$HW_API_KEY"
```

**❌ Incorrect approach:**

```bash
# Repeatedly retrying without fixing credentials
devbridge create my-tunnel -d "测试" -e 8
# → Error: authentication failed
devbridge create my-tunnel -d "测试" -e 8
# → Error: authentication failed (still fails)
```

---

## Tunnel Creation Issues

### Problem: `Failed to create tunnel: quota exceeded`

**Cause:** Account has reached the maximum number of tunnels (default 10 per workspace).

**Solution:**

1. List existing tunnels: `devbridge list`
2. Check usage: `devbridge limits`
3. Delete unused tunnels: `devbridge delete <tunnelId>`

### Problem: `Failed to create tunnel: invalid name`

**Cause:** Tunnel name contains invalid characters.

**Solution:**

- Use letters, numbers, and hyphens; no spaces in descriptions.
- Example: `devbridge create my-tunnel-001 -d "测试隧道"`

### Problem: `Failed to create tunnel: network error`

**Cause:** Cannot reach DevBridge service.

**Solution:**

1. Check network connectivity.
2. Run `devbridge ping <uri>` to test connectivity.
3. Confirm your client is v0.2.x (old clients cannot connect after the domain switch).

---

## Port Management Issues

### Problem: `Port already exists`

**Cause:** The port number is already in use on the tunnel.

**Solution:**

1. List existing ports: `devbridge port list <tunnelId>`
2. Use a different port number, or delete the existing port first:

```bash
devbridge port delete <tunnelId> -p 8080
devbridge port create <tunnelId> -p 8080 --protocol auto
```

### Problem: `Invalid port number`

**Cause:** Port number is out of valid range.

**Solution:**

- Use port numbers between 1 and 65535.
- Ports below 1024 may require elevated privileges.

### Problem: Need to change a port's protocol

**Cause:** CLI does not support modifying an existing port's protocol.

**Solution:** Delete the port and recreate with the new protocol:

```bash
devbridge port delete <tunnelId> -p 8080
devbridge port create <tunnelId> -p 8080 --protocol https
```

---

## Host Issues

### Problem: `Host failed: local service not reachable`

**Cause:** No service is running on the specified local port, or the service listens only on the loopback address.

**Solution:**

1. Check the listening address first — not just whether the process is alive:

```bash
ss -tlnp | grep 8080
# ✅ Expected: LISTEN 0 511 0.0.0.0:8080 0.0.0.0:*
# ❌ Problem:  LISTEN 0 511 127.0.0.1:8080 127.0.0.1:*   (loopback only)
```

1. If the address is `127.0.0.1`/`::1`, rebind to `0.0.0.0` (see the entry below) and restart the service before running `devbridge host`.
1. If the address is already `0.0.0.0`, confirm the service responds:

```bash
curl http://localhost:8080
```

1. Start the service before running `devbridge host`.

### Problem: 隧道已创建但公网地址无法访问（`devbridge host` 正常，浏览器/`curl` 打不开公网 URL）

**Cause:** 本地服务只监听了回环地址（`127.0.0.1`/`localhost`/`::1`），没有监听 `0.0.0.0`。`devbridge host` 本身可以成功建立隧道并返回公网 URL，但中继转发到本机非回环接口的请求无法到达服务，表现为「隧道创建成功、公网打不开」的静默失败。

> ⚠️ `curl http://localhost:<port>` 通过**不能**证明监听正确——回环地址上的 `curl` 即使服务只绑 `127.0.0.1` 也会成功。必须直接查看监听地址。

**Solution:**

1. 查看服务实际监听地址：

```bash
ss -tlnp | grep <port>
# ✅ Expected: LISTEN 0 511 0.0.0.0:<port> 0.0.0.0:*
# ❌ Problem:  LISTEN 0 511 127.0.0.1:<port> 127.0.0.1:*   (loopback only)
```

1. 若为 `127.0.0.1`/`::1`，修改服务启动参数显式绑定 `0.0.0.0`。示例：`python3 -m http.server <port> --bind 0.0.0.0`；Flask `app.run(host="0.0.0.0")`；Django `runserver 0.0.0.0:<port>`；Node `server.listen(<port>, "0.0.0.0")`；Go `http.ListenAndServe("0.0.0.0:<port>", nil)`；Vite `npm run dev -- --host 0.0.0.0`。
1. 停止旧服务并重启，使其按新绑定地址监听；再 `ss -tlnp | grep <port>` 确认已变为 `0.0.0.0:<port>`。
1. 重新托管隧道：必要时先 `db_process_stop <pid>` 停掉旧 host 进程，再执行 `db_host <tunnelId>`。
1. 用 `db_tunnel_url <tunnelId>` 获取权威公网地址（不要手工拼接域名），再次访问验证。

### Problem: `Host failed: tunnel expired`

**Cause:** The tunnel has reached its expiration time.

**Solution:**

1. Check tunnel status: `devbridge show <tunnelId>`
2. If expired, create a new tunnel and reconfigure ports.
3. Host can also create a temporary tunnel: `devbridge host -p 8080 -e 8`.

### Problem: `Host disconnected unexpectedly`

**Cause:** Network interruption or service crash.

**Solution:**

1. Check local service status.
2. Check network connectivity.
3. Restart host: `devbridge host <tunnelId>`
4. Use debug logging for details:

```bash
devbridge -v host <tunnelId>
```

### Problem: Host output shows old domain (`cn-north-4-bridge`)

**Cause:** Client is outdated (v0.1.x); the 2026-09-20 domain switch invalidated old endpoints.

**Solution:** Upgrade to v0.2.x and recreate tunnels.

---

## Connect Issues

### Problem: `Connect failed: tunnel not found`

**Cause:** The tunnel ID is incorrect or the tunnel has been deleted.

**Solution:**

1. Verify the tunnel ID: `devbridge show <tunnelId>`
2. If deleted, request the correct tunnel ID from the host operator.

### Problem: `Connect failed: local port in use`

**Cause:** The local port is already occupied by another process.

**Solution:**

1. Find and stop the process using the port:

```bash
lsof -i :8080
kill <PID>
```

2. Release the port or change the local service port, then reconnect.

### Problem: `Connection drops frequently`

**Cause:** Unstable network or idle timeout.

**Solution:**

1. Check network stability.
2. The CLI automatically reconnects on transient failures.
3. If issues persist, restart the connect session.

---

## Network Issues

### Problem: `Connection timeout`

**Cause:** Firewall or network restrictions blocking DevBridge traffic.

**Solution:**

1. Check firewall rules for outbound HTTPS (443) access.
2. Configure proxy if needed:

```bash
export https_proxy=http://<proxy-host>:<proxy-port>
devbridge host <tunnelId>
```

### Problem: `SSL certificate verification failed`

**Cause:** Corporate proxy intercepting SSL traffic.

**Solution:**

1. Configure the proxy's CA certificate.
2. Contact your network administrator for the CA certificate.
3. Set the certificate path:

```bash
export SSL_CERT_FILE=/path/to/ca-bundle.crt
```

### Problem: Old domain no longer works

**Cause:** 2026-09-20 domain switch: old `*.cn-north-4-bridge.myhuaweicloud.com` endpoints stopped working; old clients cannot connect.

**Solution:** Upgrade client to v0.2.x and recreate tunnels. New addresses use the new cluster domain, e.g. `https://<tunnelId>.<clusterId>.myhuaweicloud.com` (in practice `*.devbridge-s2.hwtunnel.com`).

---

## Permission / Auth Failure Issues

### Problem: `401 Unauthorized` / `token not found`

**Cause:** Missing or invalid API Key.

**Solution:**

1. Re-create an API Key at `https://devstation.connect.huaweicloud.com/space/devbridge?tab=apikey`.
2. Inject and re-login: `export HW_API_KEY=<key>` then `devbridge auth login --api-key "$HW_API_KEY"`.

### Problem: `403 Forbidden: token expired`

**Cause:** API Key deleted/revoked.

**Solution:**

```bash
devbridge auth logout
export HW_API_KEY="<new-api-key>"
devbridge auth login --api-key "$HW_API_KEY"
```

---

## Getting Help

If the issue persists after trying the solutions above:

1. **Enable verbose logging** for detailed logs:

```bash
devbridge -v <command>
```

2. **Check version** — ensure you are using v0.2.x:

```bash
devbridge version
```

3. **Collect diagnostic information:**

```bash
devbridge auth status
devbridge list
devbridge -v <failing-command>
```

4. **Contact Huawei Cloud support** with the diagnostic information.
5. Report issues to the DevBridge team: DevBridge team Issue feedback (see official docs).
