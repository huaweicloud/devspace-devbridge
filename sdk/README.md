# DevBridge Go SDK

DevBridge 是华为云的开发隧道服务，通过中继连接 Host（本地服务托管）与 Connect（远端连接），使本地服务无需公网入站端口即可被远程访问。

本 SDK 把 DevBridge CLI 的核心能力封装为 Go API，支持：

- **隧道管理**：创建、查询、更新、删除隧道
- **端口管理**：创建、查询、更新、删除端口
- **令牌签发**：签发 Host / Connect 令牌
- **Host 托管**：把本地端口通过 WebSocket + SSH 隧道转发到 DevBridge 中继
- **Connect 连接**：连接隧道，在本地建立端口映射访问远端服务

## 安装

```bash
go get huawei.com/devbridge/sdk
```

## 快速开始

### 完整流程：Host 托管 + Connect 连接

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/huaweicloud/devbridge-sdk"
)

func main() {
    ctx := context.Background()
    client, err := devbridge.NewClient(devbridge.Config{APIKey: "your-api-key"})
    if err != nil {
        log.Fatal(err)
    }

    // 1. 创建隧道
    tunnel, err := client.CreateTunnel(ctx, "my-tunnel", "开发联调", nil)
    if err != nil {
        log.Fatal(err)
    }

    // 2. 添加端口
    allowAnon := true
    client.CreatePort(ctx, tunnel.ID, 8080, "http", &allowAnon)

    // 3. Host 托管（在服务所在设备）
    go func() {
        client.Host(ctx, devbridge.HostConfig{
            TunnelID: tunnel.ID,
            Ports:    []int{8080},
        })
    }()

    time.Sleep(2 * time.Second)

    // 4. Connect 连接（在访问设备）
    //    连接后 http://localhost:8080 → 远端服务
    client.Connect(ctx, devbridge.ConnectConfig{
        TunnelID: tunnel.ID,
        Ports:    []int{8080},
    })
}
```

### 仅 Host 托管

```go
client, err := devbridge.NewClient(devbridge.Config{APIKey: "your-api-key"})
    if err != nil {
        log.Fatal(err)
    }

// 阻塞运行，Ctrl+C / cancel context 停止
client.Host(ctx, devbridge.HostConfig{
    TunnelID: "aaaadysa",
    Ports:    []int{8080, 3000},
})
```

### 仅 Connect 连接

```go
client, err := devbridge.NewClient(devbridge.Config{APIKey: "your-api-key"})
    if err != nil {
        log.Fatal(err)
    }

// 阻塞运行，连接后 localhost:8080 → 远端
client.Connect(ctx, devbridge.ConnectConfig{
    TunnelID: "aaaadysa",
    Ports:    []int{8080},
})
```

### 使用 JWT 令牌（跳过 API 调用）

```go
client, err := devbridge.NewClient(devbridge.Config{})
    if err != nil {
        log.Fatal(err)
    }

// 签发 Host 令牌
token, _ := client.IssueToken(ctx, "aaaadysa", "host")

// 用令牌启动 Host
client.Host(ctx, devbridge.HostConfig{
    TunnelID: "aaaadysa",
    JWTToken: token.Token,
})
```

## API 参考

### 客户端配置

```go
client, err := devbridge.NewClient(devbridge.Config{
    APIKey:      "your-key",            // API Key
    APIBaseURL:  "custom-url",          // 自定义 API 地址
    GatewayAddr: "addr:443",            // 自定义网关地址
    GatewayHost: "host",                // 自定义网关 SNI host
    ClusterID:   "custom-cluster",      // 自定义集群
    Logger:      logger,                // 自定义 logger
})
if err != nil {
    log.Fatal(err)
}
```

API Key 也可通过环境变量 `HW_API_KEY` 设置。

### 隧道管理

| 方法 | 说明 |
|------|------|
| `CreateTunnel(ctx, name, desc, *expiration)` | 创建隧道 |
| `ListTunnels(ctx)` | 查询隧道列表 |
| `ShowTunnel(ctx, id)` | 查询隧道详情 |
| `UpdateTunnel(ctx, id, *name, *desc, *exp)` | 更新隧道 |
| `DeleteTunnel(ctx, id)` | 删除隧道 |
| `DeleteAllTunnels(ctx)` | 删除全部隧道 |
| `IssueToken(ctx, id, scope)` | 签发令牌（scope: "host" / "connect"） |
| `GetLimits(ctx)` | 查询配额 |

### 端口管理

| 方法 | 说明 |
|------|------|
| `CreatePort(ctx, tunnelID, port, protocol, *allowAnon)` | 创建端口 |
| `ListPorts(ctx, tunnelID)` | 查询端口列表 |
| `ShowPort(ctx, tunnelID, port)` | 查询端口详情 |
| `UpdatePort(ctx, tunnelID, port, *allowAnon)` | 更新端口 |
| `DeletePort(ctx, tunnelID, port)` | 删除端口 |

### Host / Connect

| 方法 | 说明 |
|------|------|
| `Host(ctx, HostConfig)` | 托管本地端口（阻塞） |
| `Connect(ctx, ConnectConfig)` | 连接远端隧道（阻塞） |

## 架构

```
设备 A (Host)                          设备 B (Connect)
┌─────────────┐                       ┌──────────────┐
│ 本地服务     │                       │ 访问者       │
│ :8080       │                       │ localhost:8080│
└──────┬──────┘                       └──────┬───────┘
       │                                     │
  ┌────┴────┐                          ┌─────┴─────┐
  │ Host    │                          │ Connect   │
  │ (SDK)   │                          │ (SDK)     │
  └────┬────┘                          └─────┬─────┘
       │ WebSocket 出站                      │ WebSocket 出站
       └───────────┐     ┌──────────────────┘
                   ▼     ▼
              ┌─────────────────┐
              │ DevBridge 中继   │
              │ (SSH over WS)   │
              └─────────────────┘
```

Host 和 Connect 都主动连接 DevBridge 中继，因此 Host 所在设备不需要开放公网入站端口。

## 错误处理

```go
err := client.Host(ctx, cfg)
if err != nil {
    switch {
    case errors.Is(err, devbridge.ErrTunnelNotFound):
        // 隧道不存在
    case errors.Is(err, devbridge.ErrDuplicateHost):
        // 已有 Host 在运行
    case errors.Is(err, devbridge.ErrQuotaExceeded):
        // 配额超限
    default:
        // 其他错误
    }

    // 检查 API 业务错误
    if code, ok := devbridge.IsAPIError(err); ok {
        fmt.Println("错误码:", code) // 如 "HD.98320078"
    }
}
```

## 依赖

- `github.com/coder/websocket` — WebSocket 客户端
- `github.com/microsoft/dev-tunnels-ssh` — SSH 隧道协议
