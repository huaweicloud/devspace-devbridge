---
title: Go SDK
description: Use the DevBridge Go SDK to programmatically host and access local development services.
---

# Go SDK

DevBridge is Huawei Cloud's development tunnel service. It connects Host (local service hosting) with Connect (remote connection) via a relay, enabling local services to be accessed remotely without a public inbound port.

The Go SDK wraps the core capabilities of the DevBridge CLI as a Go API.

## Features

- **Tunnel management**: create, query, update, and delete tunnels
- **Port management**: create, query, update, and delete ports
- **Token issuance**: issue Host / Connect tokens
- **Host hosting**: forward local ports to the DevBridge relay via a WebSocket + SSH tunnel
- **Connect connection**: connect to a tunnel and establish local port mappings to access remote services

## Installation

Requires Go 1.22 or above. Replace vX.Y.Z with the released SDK version:

```bash
go get github.com/huaweicloud/devspace-devbridge/go-sdk@vX.Y.Z
```

## Quick Start

### Full flow: create tunnel → Host hosting → Connect connection

```go
package main

import (
	"context"
	"log"

	"github.com/huaweicloud/devspace-devbridge/go-sdk"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := sdk.New(sdk.Config{APIKey: "your-api-key"})

	// 1. Create tunnel
	tunnel, err := client.CreateTunnel(ctx, "my-tunnel", "devdebug", nil)
	if err != nil {
		log.Fatal(err)
	}

	// 2. Add port
	allowAnon := true
	if err := client.CreatePort(ctx, tunnel.ID, 8080, "http", &allowAnon); err != nil {
		log.Fatal(err)
	}

	// 3. Host hosting (on the device running the service)
	//    Receive ready notification via OnReady, no polling needed
	hostReady := make(chan struct{})
	go func() {
		err := client.Host(ctx, sdk.HostConfig{
			TunnelID: tunnel.ID,
			Ports:    []int{8080},
			OnReady: func(ports []int) {
				log.Println("Host ready, hosted ports:", ports)
				close(hostReady)
			},
		})
		if err != nil {
			log.Println("Host exited:", err)
		}
	}()

	// 4. Connect connection (on the accessing device)
	//    Wait for Host to be ready; after connecting, http://localhost:8080 → remote service
	<-hostReady
	err = client.Connect(ctx, sdk.ConnectConfig{
		TunnelID: tunnel.ID,
		Ports:    []int{8080},
	})
	if err != nil {
		log.Fatal(err)
	}
}
```

### Host hosting only

```go
// Blocking run; stop with Ctrl+C / cancel context
err := client.Host(ctx, sdk.HostConfig{
	TunnelID: "aaaadysa",
	Ports:    []int{8080, 3000},
})
```

### Connect connection only

```go
// Blocking run; after connecting, localhost:8080 → remote
err := client.Connect(ctx, sdk.ConnectConfig{
	TunnelID: "aaaadysa",
	Ports:    []int{8080},
})
```

### Using a JWT token (skip API calls)

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

## Examples

For more complete runnable examples, see [example_test.go](https://github.com/huaweicloud/devspace-devbridge/blob/master/go-sdk/example_test.go), covering:

- Full flow: create tunnel → add port → Host hosting → Connect connection
- Host hosting with an existing tunnel
- Using a JWT token (skip API calls)
- Connect connection and access to remote services
- Tunnel management

## API Reference

For complete type and method documentation, see [pkg.go.dev](https://pkg.go.dev/github.com/huaweicloud/devspace-devbridge/go-sdk).

### Client configuration

```go
client := sdk.New(sdk.Config{
	APIKey:      "your-key",   // API Key; reads HW_API_KEY env var when empty
	APIBaseURL:  "custom-url", // Custom REST API address
	GatewayAddr: "addr:443",   // WebSocket gateway address (host:port)
	GatewayHost: "host",       // WebSocket gateway SNI host
	OutputWriter: io.Discard,  // Status output; defaults to os.Stdout, io.Discard silences it

	// Extra headers merged into the host/connect WebSocket handshake.
	WSHeaders: http.Header{"User-Agent": []string{"my-app/1.0"}},

	// Custom HTTP client for REST API requests; nil uses the SDK default.
	HTTPClient: http.DefaultClient,
})
```

### Tunnel management

| Method                                       | Description                                                                                   |
| -------------------------------------------- | --------------------------------------------------------------------------------------------- |
| CreateTunnel(ctx, name, desc, *expiration)   | Create a tunnel; the description supports only Chinese characters/letters/digits (0-64 chars) |
| ListTunnels(ctx)                             | List tunnels                                                                                  |
| ShowTunnel(ctx, id)                          | Show tunnel details                                                                           |
| UpdateTunnel(ctx, id, \*name, \*desc, \*exp) | Update a tunnel                                                                               |
| DeleteTunnel(ctx, id)                        | Delete a tunnel                                                                               |
| DeleteAllTunnels(ctx)                        | Delete all tunnels                                                                            |
| IssueToken(ctx, id, scope)                   | Issue a token (scope: "host" / "connect")                                                     |
| GetLimits(ctx)                               | Query quotas                                                                                  |

### Port management

| Method                                                | Description       |
| ----------------------------------------------------- | ----------------- |
| CreatePort(ctx, tunnelID, port, protocol, *allowAnon) | Create a port     |
| ListPorts(ctx, tunnelID)                              | List ports        |
| ShowPort(ctx, tunnelID, port)                         | Show port details |
| UpdatePort(ctx, tunnelID, port, *allowAnon)           | Update a port     |
| DeletePort(ctx, tunnelID, port)                       | Delete a port     |

### Host / Connect

| Method                      | Description                           |
| --------------------------- | ------------------------------------- |
| Host(ctx, HostConfig)       | Host local ports (blocking)           |
| Connect(ctx, ConnectConfig) | Connect to a remote tunnel (blocking) |

Common fields of HostConfig / ConnectConfig:

| Field                  | Description                                                                                                      |
| ---------------------- | ---------------------------------------------------------------------------------------------------------------- |
| Ports                  | Port list; when empty for Host, delivered from the gateway; when empty for Connect, negotiated with Host via SSH |
| JWTToken / APIKey      | Choose one; skips the SDK's internal token issuance                                                              |
| LocalIP (Connect only) | Local listen address, defaults to 127.0.0.1                                                                      |
| OnReady                | Session-ready callback: Host returns the hosted port list, Connect returns the port mapping list                 |

## Error handling

```go
err := client.Host(ctx, cfg)
if err != nil {
	switch {
	case errors.Is(err, sdk.ErrMissingAPIKey):
		// Missing API Key
	case errors.Is(err, sdk.ErrTunnelNotFound):
		// Tunnel not found
	case errors.Is(err, sdk.ErrDuplicateHost):
		// A Host is already running
	case errors.Is(err, sdk.ErrQuotaExceeded):
		// Quota exceeded
	case errors.Is(err, sdk.ErrInvalidTunnelID):
		// Invalid tunnel ID format
	case errors.Is(err, sdk.ErrInvalidTunnelName):
		// Invalid tunnel name (Chinese chars/letters/digits/hyphens, 1-64 chars)
	case errors.Is(err, sdk.ErrInvalidTunnelDescription):
		// Invalid tunnel description (Chinese chars/letters/digits only, 0-64 chars)
	case errors.Is(err, sdk.ErrInvalidPort):
		// Invalid port number
	case errors.Is(err, sdk.ErrInvalidProtocol):
		// Invalid protocol (http / https / auto)
	case errors.Is(err, sdk.ErrInvalidScope):
		// Invalid token scope (host / connect)
	default:
		// Other error
	}

	// Check API business errors
	if code, ok := sdk.IsAPIError(err); ok {
		fmt.Println("Error code:", code) // e.g. "HD.98320078"
	}
}
```

## Architecture

```text
Device A (Host)                       Device B (Connect)
┌───────────────┐                     ┌────────────────┐
│ Local service │                     │ Visitor        │
│ :8080         │                     │ localhost:8080 │
└───────┬───────┘                     └───────┬────────┘
        │                                     │
   ┌────┴────┐                           ┌─────┴─────┐
   │ Host    │                           │ Connect   │
   │ (SDK)   │                           │ (SDK)     │
   └────┬────┘                           └─────┴─────┘
        │ WebSocket outbound                 │ WebSocket outbound
        └───────────┐     ┌───────────────────┘
                    ▼     ▼
              ┌─────────────────┐
              │ DevBridge relay │
              │ (SSH over WS)   │
              └─────────────────┘
```

Both Host and Connect actively connect to the DevBridge relay, so the device running Host does not need to open a public inbound port.

## Dependencies

- github.com/coder/websocket — WebSocket client
- github.com/microsoft/dev-tunnels-ssh — SSH tunnel protocol

## Related links

- [SDK source on GitHub](https://github.com/huaweicloud/devspace-devbridge/tree/master/go-sdk)
- [SDK changelog](https://github.com/huaweicloud/devspace-devbridge/blob/master/go-sdk/CHANGELOG.md)
