---
title: 创建并托管隧道
description: 安装 DevBridge CLI，登录并通过开发隧道托管本地服务，再从另一台设备连接。
---

# 创建并托管隧道

<p class="lead">在本快速入门中，你将托管本机的 `8080` 端口，然后从另一台设备连接隧道。</p>

## 准备工作

- 已安装 DevBridge CLI，并确认 `devbridge version` 可以正常执行，参阅[安装 DevBridge CLI](./install.md)；
- 已完成登录：个人环境使用交互登录，自动化环境使用 API Key，参阅[登录与凭证](./authentication.md)；
- 另一台设备可用于验证 Connect 模式（需同样安装并登录）。

::: info 适用场景
开发隧道用于开发、联调和临时共享。不要把它作为生产服务的长期入口。
:::

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
`devbridge host <tunnelId>` 托管其中配置的全部端口。详见[管理隧道](./tunnels.md)。
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
[Connect：连接远程服务](./connect.md)。

## 下一步

- [创建和管理持久隧道](./tunnels.md)
- [配置端口协议与匿名访问](./ports.md)
- [最佳实践：托管与公网访问](./best-practices/host-public-access.md)
- [查看 CLI 命令参考](../reference/cli.md)
