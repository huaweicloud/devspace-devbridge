---
title: CLI 命令参考
description: DevBridge CLI 的认证、隧道、端口、Host 和 Connect 命令速查。
---

# CLI 命令参考

<p class="lead">本页汇总 DevBridge CLI 的主要命令。使用命令级 <code>--help</code> 查看当前安装版本的完整参数。</p>

## 全局命令

| 命令                         | 说明                 |
| ---------------------------- | -------------------- |
| `devbridge --help`           | 显示 CLI 帮助。      |
| `devbridge version`          | 显示 CLI 版本。      |
| `devbridge <command> --help` | 显示指定命令的参数。 |

### 全局参数

| 参数              | 说明                                 |
| ----------------- | ------------------------------------ |
| `--help`、`-h`    | 显示当前命令的帮助。                 |
| `--verbose`、`-v` | 输出更详细的运行日志，便于排查问题。 |

## 认证

| 命令                                | 说明                   |
| ----------------------------------- | ---------------------- |
| `devbridge auth login`              | 交互登录（打开浏览器）。 |
| `devbridge auth login --api-key <key>` | 使用 API Key 登录。    |
| `devbridge auth status`             | 查看当前登录状态。     |
| `devbridge auth logout`             | 清除本地登录凭证。     |

### 登录参数

| 参数        | 说明                                            |
| ----------- | ----------------------------------------------- |
| `--api-key` | API Key，跳过浏览器交互直接登录。               |

CLI 也会自动读取 `HW_API_KEY` 环境变量。

## 隧道

| 命令                                    | 说明                         |
| --------------------------------------- | ---------------------------- |
| `devbridge create <name>`               | 创建隧道。                   |
| `devbridge list`                        | 列出当前工作空间的有效隧道。 |
| `devbridge show <tunnelId>`             | 查看隧道详情。               |
| `devbridge update <tunnelId>`           | 更新隧道。                   |
| `devbridge delete <tunnelId>`           | 删除一条隧道。               |
| `devbridge delete-all`                  | 删除当前工作空间的全部隧道。 |
| `devbridge token <tunnelId> -s host`    | 签发新的 Host 令牌。         |
| `devbridge token <tunnelId> -s connect` | 签发新的 Connect 令牌。      |
| `devbridge set <tunnelId>`              | 设置本机默认隧道。           |
| `devbridge unset`                       | 清除本机默认隧道。           |

### 隧道参数

| 参数 | 适用命令                                | 说明                            |
| ---- | --------------------------------------- | ------------------------------- |
| `-n` | `update`                                | 更新隧道名称。                  |
| `-d` | `create`、`update`、无隧道 ID 的 `host` | 设置隧道描述。                  |
| `-e` | `create`、`update`、无隧道 ID 的 `host` | 设置有效期规格，单位为小时。    |
| `-s` | `token`                                 | 令牌范围：`host` 或 `connect`。 |

## 端口

| 命令                                                               | 说明                     |
| ------------------------------------------------------------------ | ------------------------ |
| `devbridge port create <tunnelId> -p <port> --protocol <protocol>` | 创建端口。               |
| `devbridge port list <tunnelId>`                                   | 列出隧道端口。           |
| `devbridge port show <tunnelId> -p <port>`                         | 查看端口详情。           |
| `devbridge port update <tunnelId> -p <port>`                       | 更新匿名访问策略。       |
| `devbridge port delete <tunnelId> -p <port>`                       | 删除端口。               |

### 端口参数

| 参数                      | 说明                                                     |
| ------------------------- | -------------------------------------------------------- |
| `-p`、`--port-number`     | 端口号，范围为 `1` 到 `65535`。                          |
| `--protocol`              | `http`、`https` 或 `auto`。仅创建时可用，默认为 `auto`。 |
| `--deny-anonymous`        | 创建或更新时禁止匿名访问。                               |
| `-a`、`--allow-anonymous` | 创建或更新时允许匿名访问。                               |

当前 CLI 不支持修改已有端口的协议。端口命令不支持 `-d` 描述参数，也不提供端口批量删除命令。

## Host

| 命令                                                   | 说明                           |
| ------------------------------------------------------ | ------------------------------ |
| `devbridge host <tunnelId>`                            | 托管已有隧道中配置的全部端口。 |
| `devbridge host -p <port> -d <description> -e <hours>` | 创建隧道并立即托管端口。       |
| `devbridge host <tunnelId> --token <jwt>`              | 使用已有 JWT 令牌托管，跳过 API 调用。 |
| `devbridge host <tunnelId> --api-key <key>`            | 使用 API Key 鉴权，跳过令牌签发。       |

Host 是前台长运行命令。`-d` 和 `-e` 只在 Host 同时创建隧道时描述新隧道。

### Host 参数

| 参数                  | 说明                                                         |
| --------------------- | ------------------------------------------------------------ |
| `-p`、`--ports`       | 本地端口号，可传多个。仅在创建临时隧道时使用。               |
| `-d`、`--description` | 新隧道的描述。                                               |
| `-e`、`--expiration`  | 新隧道的有效期，单位为小时。                                 |
| `-t`、`--token`       | 直接提供 JWT 令牌，跳过 API 令牌签发和端口查询。             |
| `-k`、`--api-key`     | 使用 API Key 鉴权，跳过 TunnelToken，通过 X-API-Key 认证。   |

## Connect

| 命令                                                | 说明                               |
| --------------------------------------------------- | ---------------------------------- |
| `devbridge connect <tunnelId>`                      | 连接隧道并建立本地端口映射。       |
| `devbridge connect <tunnelId> --token <jwt>`        | 使用已有 JWT 令牌连接，跳过 API 调用。 |
| `devbridge connect <tunnelId> --api-key <key>`      | 使用 API Key 鉴权，跳过令牌签发。       |

### Connect 参数

| 参数              | 说明                                                         |
| ----------------- | ------------------------------------------------------------ |
| `-t`、`--token`   | 直接提供 JWT 令牌，跳过 API 令牌签发和端口查询。             |
| `-k`、`--api-key` | 使用 API Key 鉴权，跳过 TunnelToken，通过 X-API-Key 认证。   |

## 配额查询

| 命令               | 说明                     |
| ------------------ | ------------------------ |
| `devbridge limits` | 查看账户配额与当前用量。 |

输出包含：

- 重置时间；
- 流量配额与已用流量；
- 活跃隧道数；
- 隧道、端口、Host 数量上限；
- 隧道带宽上限；
- 单端口 HTTP 请求频率上限；
- 单端口连接数上限。

## 调试工具

### echo

启动一个 HTTP echo 服务，用于验证隧道链路是否通畅。

| 命令                                | 说明                                             |
| ----------------------------------- | ------------------------------------------------ |
| `devbridge echo`                    | 启动 echo 服务，默认随机端口，监听 `127.0.0.1`。 |
| `devbridge echo -p 8080`            | 指定监听端口。                                   |
| `devbridge echo -p 8080 -i 0.0.0.0` | 指定监听端口和监听地址。                         |

### ping

对指定 URI 发起 HTTP ping 探测，用于检查隧道地址的连通性和延迟。

| 命令                          | 说明                                |
| ----------------------------- | ----------------------------------- |
| `devbridge ping <uri>`        | 对 URI 发起探测，默认 1000ms 间隔。 |
| `devbridge ping <uri> -i 500` | 指定探测间隔，单位为毫秒。          |

示例：

```bash
devbridge ping https://<tunnelId>-8080.cn-north-4-bridge.myhuaweicloud.com
devbridge ping http://127.0.0.1:8080 -i 3000
```

输出形如：

```text
HTTP 200 OK -- 4 ms
```

### 调试日志

任一命令追加 `-v` / `--verbose` 启用 Debug 级别日志：

```bash
devbridge -v host <tunnelId>
devbridge connect <tunnelId> --verbose
```

## 补全

| 命令                                                 | 说明                                        |
| ---------------------------------------------------- | ------------------------------------------- |
| `devbridge completion [bash\|zsh\|fish\|powershell]` | 生成 Shell 自动补全脚本（Cobra 内置命令）。 |

## 相关内容

- [管理隧道](../guide/tunnels.md)
- [管理端口](../guide/ports.md)
- [Host：托管本地服务](../guide/host.md)
- [Connect：连接远程服务](../guide/connect.md)
