---
title: 配额查询与调试工具
description: 使用 limits 查询配额，使用 echo 和 ping 调试隧道链路。
---

# 配额查询与调试工具

<p class="lead">DevBridge CLI 提供配额查询和链路调试命令，帮助确认资源用量和排查连接问题。</p>

## 查看配额与用量

使用 `devbridge limits` 查看当前账户的配额与用量：

```bash
devbridge limits
```

输出包含：

| 项目                     | 说明                           |
| ------------------------ | ------------------------------ |
| 重置时间                 | 配额计量的重置时间点。         |
| 流量配额 / 已用流量      | 隧道转发的流量上限和当前用量。 |
| 活跃隧道数               | 当前工作空间中的有效隧道数量。 |
| 隧道 / 端口 / Host 上限  | 各资源的最大数量限制。         |
| 隧道带宽上限             | 单条隧道的带宽限制。           |
| 单端口 HTTP 请求频率上限 | 单个端口允许的 HTTP 请求频率。 |
| 单端口连接数上限         | 单个端口允许的并发连接数。     |

::: tip 何时查看配额

- 创建隧道或端口失败时，确认是否达到数量上限；
- Host 或 Connect 流量异常时，确认是否达到流量或带宽上限；
- 请求被限流时，确认是否达到请求频率上限。

:::

## 使用 echo 验证链路

`devbridge echo` 启动一个 HTTP echo 服务，返回请求的详细信息，用于验证 Host 和 Connect 链路是否通畅。

### 启动 echo 服务

```bash
# 默认随机端口，监听 127.0.0.1
devbridge echo

# 指定端口和监听地址
devbridge echo -p 8080 -i 0.0.0.0
```

| 参数 | 说明                         |
| ---- | ---------------------------- |
| `-p` | 监听端口，默认随机分配。     |
| `-i` | 监听地址，默认 `127.0.0.1`。 |

### 配合 Host 验证

1. 在终端 1 启动 echo 服务：

   ```bash
   devbridge echo -p 8080
   ```

2. 在终端 2 启动 Host 托管 `8080` 端口：

   ```bash
   devbridge host -p 8080
   ```

3. 在另一台设备通过 Connect 或浏览器访问隧道地址，如果返回 echo 服务的响应，说明链路通畅。

echo 服务返回的是请求详情（方法、路径、请求头等），与 `python3 -m http.server` 返回目录列表不同，适合确认请求是否正确到达。

## 使用 ping 探测连通性

`devbridge ping` 对指定 URI 发起 HTTP ping 探测，输出状态码和延迟，用于检查隧道地址是否可达。

### 基本用法

```bash
# 对隧道地址发起探测，默认 1000ms 间隔
devbridge ping https://<tunnelId>-8080.cn-north-4-bridge.myhuaweicloud.com

# 指定探测间隔（毫秒）
devbridge ping https://<tunnelId>-8080.cn-north-4-bridge.myhuaweicloud.com -i 500
```

也可以探测本地端口（例如 Connect 建立映射后）：

```bash
devbridge ping http://127.0.0.1:8080 -i 3000
```

输出形如：

```text
HTTP 200 OK -- 4 ms
```

| 参数 | 说明                              |
| ---- | --------------------------------- |
| `-i` | 探测间隔，单位为毫秒，默认 1000。 |

### 探测场景

| 探测目标                             | 说明                                       |
| ------------------------------------ | ------------------------------------------ |
| 隧道公网地址                         | 验证 Host 是否正常运行、隧道地址是否可达。 |
| `http://127.0.0.1:<port>`            | 验证 Connect 本地映射是否正常工作。        |
| `http://127.0.0.1:<port>`（Host 侧） | 确认本地服务正在监听。                     |

按 `Ctrl+C` 停止探测。

## 调试日志

任一命令追加 `-v` / `--verbose` 启用 Debug 级别日志，输出请求地址、重试次数和耗时等诊断信息：

```bash
devbridge -v host <tunnelId>
devbridge connect <tunnelId> --verbose
devbridge -v list
```

`--verbose` 不输出 AK、SK 或令牌明文。

## 相关内容

- [CLI 命令参考](../reference/cli.md)
- [Host：托管本地服务](./host.md)
- [Connect：连接远程服务](./connect.md)
- [问题排查](../reference/troubleshooting.md)
