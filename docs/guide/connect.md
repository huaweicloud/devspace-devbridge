---
title: Connect：连接远程服务
description: 使用 Connect 模式访问 DevBridge 隧道中的远程端口。
---

# Connect：连接远程服务

<p class="lead">Connect 运行在访问设备，为隧道中的远程端口建立本地映射。</p>

## 建立连接

在已经安装并登录 DevBridge CLI 的设备上运行：

```bash
devbridge connect <tunnelId>
```

Connect 会读取隧道端口配置、申请新的 Connect 令牌，并连接当前可用的 Host。

连接建立后，通过本机相同端口访问远程服务。例如，隧道包含端口 `8080`：

```text
http://localhost:8080
```

## 连接多个端口

一条隧道可以配置多个端口。Connect 会根据隧道端口配置建立对应映射，不需要使用未经确认的
`--port` 参数筛选端口。

查看远程端口列表：

```bash
devbridge port list <tunnelId>
```

## 受保护端口

端口使用 `--deny-anonymous` 配置后，Connect 必须使用有效身份和 Connect 令牌。CLI 会自动申请令牌，通常不需要手动执行 `devbridge token`。

Host 令牌不能用于 Connect，Connect 令牌也不能用于 Host。

## 自动重连

Connect 是前台长运行命令。网络短暂中断时会自动尝试恢复连接；按 `Ctrl+C` 停止。

以下情况需要人工处理：

- 登录凭证已经失效；
- 隧道已过期或删除；
- Host 当前未运行；
- 端口已删除或协议发生变化；
- 本地端口已被其他进程占用。

## 下一步

- [管理端口访问策略](./ports.md)
- [排查连接问题](../reference/troubleshooting.md)
