---
title: 问题排查
description: 排查 DevBridge 安装、登录、隧道、Host 和 Connect 问题。
---

# 问题排查

<p class="lead">先确认本地命令、身份、隧道状态和端口配置，再检查 Host 与 Connect 的网络连接。</p>

## 找不到 devbridge 命令

确认默认安装目录存在，并加入当前终端的 `PATH`：

```bash
export PATH="$HOME/.huawei/bin:$PATH"
devbridge version
```

如果你使用 PowerShell，则执行：

```powershell
$env:Path = "$HOME/.huawei/bin;$env:Path"
devbridge version
```

如果仍然失败，重新运行[官方安装命令](../guide/install.md)并检查安装输出。

## 安装脚本下载失败

检查：

1. 当前网络是否允许访问安装地址；
2. DNS 和 HTTPS 代理是否正常；
3. 如果使用 Bash 安装方式，`curl` 是否可用；
4. 如果使用 PowerShell 安装方式，`irm` 是否可用；
5. 安装地址是否完整且没有被换行或截断。

不要在下载失败时改用来源不明的安装脚本。

## 登录失败或凭证过期

先查看状态：

```bash
devbridge auth status
```

然后重新登录：

```bash
devbridge auth logout
devbridge auth login
```

使用 API Key 登录时，确认 API Key 有效且未过期。通过环境变量传入时，检查 `HW_API_KEY` 是否已正确设置。

## 找不到隧道

```bash
devbridge list
devbridge show <tunnelId>
```

常见原因：

- 隧道 ID 输入错误；
- 当前登录身份不属于隧道工作空间；
- 隧道已过期或删除；
- 当前环境连接到了不同集群或区域。

## 无法创建更多隧道

每个工作空间默认最多拥有 10 条有效隧道。使用：

```bash
devbridge list
```

复用已有隧道，或删除不再使用的隧道。

## Host 无法连接

依次确认：

1. `devbridge auth status` 显示凭证有效；
2. `devbridge show <tunnelId>` 能读取隧道；
3. `devbridge port list <tunnelId>` 包含需要托管的端口；
4. 本地服务确实监听该端口；
5. 当前网络允许访问 DevBridge。

检查本地端口：

```bash
curl -v http://127.0.0.1:8080
```

如果端口使用 `https`，应使用 HTTPS 检查，并确认证书和本地服务配置正确。

## Connect 无法访问服务

确认：

- Host 进程仍在运行；
- Connect 与 Host 使用同一条隧道；
- 端口仍然存在；
- 当前身份可以访问禁止匿名的端口；
- Connect 设备上的本地端口没有被占用；
- 端口协议与 Host 本地服务一致。

Connect 自动重连只处理短暂网络中断，不能恢复已删除或已过期的隧道。

## 公网 URL 返回 404 或 503

通过隧道公网 URL（`https://<tunnelId>-<port>.devbridge-s2.hwtunnel.com`）访问时，返回码反映隧道与 Host 的当前状态：

| 返回码 | 含义 | 处理 |
| ------ | ---- | ---- |
| `404`，body 为 `tunnel not found` | 隧道不存在或已过期。 | 核对隧道 ID；用 `devbridge show <tunnelId>` 确认。 |
| `404`，body 为 `port not bound` | 该端口未在隧道上配置。 | 执行 `devbridge port list <tunnelId>` 核对端口。 |
| `503`，带 `Retry-After`，body 为 `host offline` | 隧道存在，但当前没有 Host 连接。 | 等待 `Retry-After` 指示的时间后重试；重启 Host 进程。 |

::: warning 服务端边界
公网返回码由网关（中继服务）判定，不在本仓库内。目标契约是：隧道不存在或已过期返回 `404 tunnel not found`；隧道存在但 Host 离线返回 `502`/`503` 并带 `Retry-After`。该实现作为 Issue #50 的跨仓依赖项跟踪，尚未在本仓库端到端验证。网关完成前，Host 离线仍可能返回 `404 tunnel not found`，此时可用 `devbridge show <tunnelId>` 的 **Host 在线状态** 字段区分两种情况。
:::

### 新增端口仍不可访问

按顺序确认：

1. `devbridge show <tunnelId>` 显示 `Host 在线状态: 在线`；
2. `devbridge port list <tunnelId>` 中已有该端口；
3. Host 以动态端口模式运行（`devbridge host <tunnelId>` 未带 `-p`）。若启动时显式指定了端口，Host 只转发该列表，需重启生效：`devbridge host <tunnelId> -p <端口列表>`；
4. 新端口对应的本地服务在 Host 机器上正在监听。

## HTTP 与 HTTPS 行为不正确

查看端口协议：

```bash
devbridge port show <tunnelId> -p 8080
```

如果本地服务是普通 HTTP，协议应为 `http`；本地服务自身完成 TLS 时使用 `https`；只有确实需要自动识别时使用 `auto`。

当前 CLI 不支持直接修改已有端口的协议。需要修改时，删除端口并使用正确协议重新创建：

```bash
devbridge port delete <tunnelId> -p 8080
devbridge port create <tunnelId> -p 8080 --protocol http
```

## 仍然无法解决

收集以下非敏感信息：

- `devbridge version`；
- 操作系统和架构；
- 执行的命令，不包含 API Key 和令牌；
- 隧道 ID、端口和协议；
- 错误发生时间；
- 已脱敏的错误信息。

不要提交完整凭证、JWT、配置目录或带密钥的终端记录。
