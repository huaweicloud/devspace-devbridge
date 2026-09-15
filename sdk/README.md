# DevBridge Go SDK

DevBridge 是华为云的开发隧道服务，通过中继连接 Host（本地服务托管）与 Connect（远端连接），使本地服务无需公网入站端口即可被远程访问。

本 SDK 把 DevBridge CLI 的核心能力封装为 Go API，支持：

- **隧道管理**：创建、查询、更新、删除隧道
- **端口管理**：创建、查询、更新、删除端口
- **令牌签发**：签发 Host / Connect 令牌
- **Host 托管**：把本地端口通过 WebSocket + SSH 隧道转发到 DevBridge 中继
- **Connect 连接**：连接隧道，在本地建立端口映射访问远端服务

## 安装

要求 Go 1.22 及以上版本。

```bash
go get github.com/huaweicloud/devspace-devbridge/sdk@v0.1.0
```

## 快速开始

### 完整流程：创建隧道 → Host 托管 → Connect 连接

```go
package main

import (
	"context"
	"log"

	"github.com/huaweicloud/devspace-devbridge/sdk"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := sdk.New(sdk.Config{APIKey: "your-api-key"})
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
	if err := client.CreatePort(ctx, tunnel.ID, 8080, "http", &allowAnon); err != nil {
		log.Fatal(err)
	}

	// 3. Host 托管（在服务所在设备）
	//    通过 OnReady 获得就绪通知，无需轮询等待
	go func() {
		err := client.Host(ctx, sdk.HostConfig{
			TunnelID: tunnel.ID,
			Ports:    []int{8080},
			OnReady: func(ports []int) {
				log.Println("Host 就绪，托管端口:", ports)
			},
		})
		if err != nil {
			log.Println("Host 退出:", err)
		}
	}()

	// 4. Connect 连接（在访问设备）
	//    连接后 http://localhost:8080 → 远端服务
	err = client.Connect(ctx, sdk.ConnectConfig{
		TunnelID: tunnel.ID,
		Ports:    []int{8080},
	})
	if err != nil {
		log.Fatal(err)
	}
}
```

更多示例见 [example_test.go](./example_test.go)。

### 仅 Host 托管

```go
// 阻塞运行，Ctrl+C / cancel context 停止
err := client.Host(ctx, sdk.HostConfig{
	TunnelID: "aaaadysa",
	Ports:    []int{8080, 3000},
})
```

### 仅 Connect 连接

```go
// 阻塞运行，连接后 localhost:8080 → 远端
err := client.Connect(ctx, sdk.ConnectConfig{
	TunnelID: "aaaadysa",
	Ports:    []int{8080},
})
```

### 使用 JWT 令牌（跳过 API 调用）

```go
token, err := client.IssueToken(ctx, "aaaadysa", "host")
if err != nil {
	log.Fatal(err)
}

err = client.Host(ctx, sdk.HostConfig{
	TunnelID: "aaaadysa",
	JWTToken: token.Token,
})
```

## API 参考

### 客户端配置

```go
client, err := sdk.New(sdk.Config{
	APIKey:      "your-key",   // API Key，留空时读取 HW_API_KEY 环境变量
	APIBaseURL:  "custom-url", // 自定义 REST API 地址
	GatewayAddr: "addr:443",   // 自定义网关地址
	GatewayHost: "host",       // 自定义网关 SNI host
	StatusWriter: io.Discard,  // 状态输出目的地，默认 os.Stdout，io.Discard 表示静音
})
```

### 隧道管理

| 方法 | 说明 |
|------|------|
| `CreateTunnel(ctx, name, desc, *expiration)` | 创建隧道，描述仅支持中文/字母/数字（0-64 字符） |
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

`HostConfig` / `ConnectConfig` 常用字段：

| 字段 | 说明 |
|------|------|
| `Ports` | 端口列表；`Host` 为空时从网关下发，`Connect` 为空时从 Host 通过 SSH 协商 |
| `JWTToken` / `APIKey` | 二选一，跳过 SDK 内部的令牌签发 |
| `LocalIP`（仅 Connect） | 本地监听地址，默认 `127.0.0.1` |
| `OnReady` | 会话就绪回调：`Host` 返回托管端口列表，`Connect` 返回端口映射列表 |

## 错误处理

```go
err := client.Host(ctx, cfg)
if err != nil {
	switch {
	case errors.Is(err, sdk.ErrMissingAPIKey):
		// 缺少 API Key
	case errors.Is(err, sdk.ErrTunnelNotFound):
		// 隧道不存在
	case errors.Is(err, sdk.ErrDuplicateHost):
		// 已有 Host 在运行
	case errors.Is(err, sdk.ErrQuotaExceeded):
		// 配额超限
	case errors.Is(err, sdk.ErrInvalidTunnelID):
		// 隧道 ID 格式无效
	case errors.Is(err, sdk.ErrInvalidTunnelDescription):
		// 隧道描述无效（仅中文/字母/数字，0-64 字符）
	case errors.Is(err, sdk.ErrInvalidPort):
		// 端口号无效
	default:
		// 其他错误
	}

	// 检查 API 业务错误
	if code, ok := sdk.IsAPIError(err); ok {
		fmt.Println("错误码:", code) // 如 "HD.98320078"
	}
}
```

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

## 依赖

- `github.com/coder/websocket` — WebSocket 客户端
- `github.com/microsoft/dev-tunnels-ssh` — SSH 隧道协议

## 更新日志

见 [CHANGELOG.md](./CHANGELOG.md)。

## 许可

[Apache License 2.0](../LICENSE)
