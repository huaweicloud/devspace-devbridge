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
devbridge --version
```

如果仍然失败，重新运行[官方安装命令](../guide/install.md)并检查安装输出。

## 安装脚本下载失败

检查：

1. 当前网络是否允许访问安装地址；
2. DNS 和 HTTPS 代理是否正常；
3. `curl` 是否可用；
4. 安装地址是否完整且没有被换行或截断。

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

使用临时 AK/SK 时，确认 AK、SK 和 Security Token 来自同一组凭证且都未过期。

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

## HTTP 与 HTTPS 行为不正确

查看端口协议：

```bash
devbridge port show <tunnelId> -p 8080
```

如果本地服务是普通 HTTP，协议应为 `http`；本地服务自身完成 TLS 时使用 `https`；只有确实需要自动识别时使用 `auto`。

修改协议：

```bash
devbridge port update <tunnelId> -p 8080 --protocol http
```

## 仍然无法解决

收集以下非敏感信息：

- `devbridge --version`；
- 操作系统和架构；
- 执行的命令，不包含 AK、SK 和令牌；
- 隧道 ID、端口和协议；
- 错误发生时间；
- 已脱敏的错误信息。

不要提交完整凭证、JWT、配置目录或带密钥的终端记录。
