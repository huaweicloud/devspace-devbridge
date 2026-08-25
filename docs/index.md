---
title: 创建并托管开发隧道
description: 安装 DevBridge CLI，登录并通过开发隧道托管本地服务。
---

# 创建并托管开发隧道

<p class="lead">使用 DevBridge 将本地开发服务安全地开放给远程设备，并控制端口的访问方式。</p>

在本快速入门中，你将安装并登录 DevBridge CLI，托管本机的 `8080` 端口，然后从另一台设备连接隧道。

## 准备工作

开始前，请确保：

- 当前设备可以运行 Bash 和 `curl`，或者 PowerShell 5.1 及以上版本；
- 你拥有可用的 DevBridge 身份或 API Key；
- 另一台设备可用于验证 Connect 模式。

::: info 适用场景
开发隧道用于开发、联调和临时共享。不要把它作为生产服务的长期入口。
:::

## 安装

使用 Bash 运行官方安装脚本：

```bash
curl -fsSL https://res-hd.hc-cdn.cn/sharedata/hdspace/devbridge/install.sh | bash
```

使用 PowerShell 时运行：

```powershell
irm https://res-hd.hc-cdn.cn/sharedata/hdspace/devbridge/install.ps1 | iex
```

安装完成后验证 CLI：

```bash
devbridge version
```

如果终端找不到命令，或者需要了解系统要求、PATH 和安装目录，请参阅
[安装 DevBridge CLI](./guide/install.md)。

## 登录

个人开发环境建议使用交互登录：

```bash
devbridge auth login
```

检查当前登录状态：

```bash
devbridge auth status
```

自动化环境可以使用 API Key。请参阅[登录与凭证](./guide/authentication.md)。

## 托管本地服务

### 1. 启动测试服务

在终端 1 启动一个本地 HTTP 服务：

```bash
python3 -m http.server 8080
```

### 2. 启动 Host

在终端 2 创建一条有效期为 8 小时的隧道，并托管本地端口：

```bash
devbridge host -p 8080 -e 8
```

Host 成功后会输出隧道 ID 和访问地址。隧道地址采用以下格式：

```text
https://<tunnelId>.<clusterId>.myhuaweicloud.com
```

保持 Host 进程运行。网络短暂中断时，CLI 会自动尝试恢复连接；按 `Ctrl+C` 停止本次托管。

::: tip 重复使用隧道
需要保留固定地址或配置多个端口时，先创建持久隧道，再使用
`devbridge host <tunnelId>` 托管其中配置的全部端口。详见[管理隧道](./guide/tunnels.md)。
:::

## 从另一台设备连接

在另一台已安装并登录 DevBridge CLI 的设备上运行：

```bash
devbridge connect <tunnelId>
```

Connect 会读取隧道端口配置并建立本地映射。连接建立后，通过本机对应端口访问 Host 设备上的服务：

```text
http://localhost:8080
```

保持 Connect 进程运行；按 `Ctrl+C` 停止连接。有关多端口和重连行为，请参阅
[Connect：连接远程服务](./guide/connect.md)。

## 下一步

- [了解开发隧道的组成和工作方式](./guide/overview.md)
- [创建和管理持久隧道](./guide/tunnels.md)
- [配置端口协议与匿名访问](./guide/ports.md)
- [查看 CLI 命令参考](./reference/cli.md)
