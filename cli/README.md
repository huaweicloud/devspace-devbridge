# DevBridge CLI

> 中文文档：[README.zh-CN.md](README.zh-CN.md)

A CLI tool for Huawei Cloud DevStation remote tunnel port forwarding. It implements secure remote port forwarding over WebSocket + SSH tunnels and supports cross-platform execution.

## Features

- **Tunnel management**: Create, view, update, and delete tunnels; supports setting a default tunnel; name validation disallows leading/trailing hyphens
- **Port management**: Bind ports to tunnels; supports HTTP/HTTPS/Auto protocols and anonymous access control
- **Host mode**: Expose local service ports to the remote gateway and wait for remote connections
- **Connect mode**: Connect to a tunnel on the remote gateway and create a local TCP Listener to forward ports
- **Quota query**: View account quotas and usage (tunnel count, port count, bandwidth, traffic, connection count, etc.)
- **Debugging tools**: Built-in HTTP echo service and URI ping probe for verifying tunnel connectivity
- **Multiple authentication methods**: Supports Huawei Cloud SSO browser login, API Key command-line arguments, and environment variables
- **Secure credential storage**: Prefers the system Keyring, falls back to a config file when unavailable
- **Automatic reconnection on disconnect**: Host side uses exponential backoff with full-jitter retries; Connect side auto-reconnects up to 5 times; the SSH layer supports session resumption
- **Direct token passing**: Host/Connect support passing a JWT directly, skipping API calls — suitable for offline or restricted environments
- **Bilingual (Chinese/English)**: English by default, switchable via environment variable

## Tech Stack

| Category           | Technology                                                                |
| ------------------ | ------------------------------------------------------------------------- |
| Language           | Go 1.26                                                                   |
| CLI framework      | [cobra](https://github.com/spf13/cobra)                                   |
| WebSocket          | [coder/websocket](https://github.com/coder/websocket)                     |
| SSH tunnel         | [microsoft/dev-tunnels-ssh](https://github.com/microsoft/dev-tunnels-ssh) |
| Credential storage | [go-keyring](https://github.com/zalando/go-keyring)                       |
| Config file        | [yaml.v3](https://gopkg.in/yaml.v3)                                       |

## Quick Start

### Installation

**One-line install (Linux / macOS):**

```bash
# Install the latest version from GitHub Release
curl -fsSL https://github.com/huaweicloud/devspace-devbridge/releases/latest/download/install.sh | bash

# Install a specific version from GitHub Release
curl -fsSL https://github.com/huaweicloud/devspace-devbridge/releases/download/v1.0.0/install.sh | bash

# Install the latest version from GitCode Release
curl -fsSL https://gitcode.com/CloudDeveloperDepartment/devbrige/releases/download/latest/install.sh | bash

# Install a specific version from GitCode Release
curl -fsSL https://gitcode.com/CloudDeveloperDepartment/devbrige/releases/download/v1.0.0/install.sh | bash

# Install the latest version from OBS
curl -fsSL https://tools-artifact.developer.huaweicloud.com/sharedata/devbridge/install.sh | bash
```

**One-line install (Windows PowerShell):**

```powershell
# Install the latest version from GitHub Release
irm https://github.com/huaweicloud/devspace-devbridge/releases/latest/download/install.ps1 | iex

# Install a specific version from GitHub Release
irm https://github.com/huaweicloud/devspace-devbridge/releases/download/v1.0.0/install.ps1 | iex

# Install the latest version from GitCode Release
irm https://gitcode.com/CloudDeveloperDepartment/devbrige/releases/download/latest/install.ps1 | iex

# Install the latest version from OBS
irm https://tools-artifact.developer.huaweicloud.com/sharedata/devbridge/install.ps1 | iex
```

The install script automatically detects the platform, downloads the corresponding binary, verifies the SHA256 checksum, installs it to `~/.huawei/bin/`, and configures PATH.

**Install options (optional):**

```bash
# Specify install directory (default ~/.huawei/bin)
curl -fsSL <release-url>/install.sh | bash -s -- -p /opt/devbridge

# Silent install (CI/CD scenarios, skip interactive prompts)
curl -fsSL <release-url>/install.sh | bash -s -- -s

# Skip checksum verification
curl -fsSL <release-url>/install.sh | bash -s -- --skip-checksum
```

Install directory: `~/.huawei/bin/`, config directory: `~/.huawei/devbridge/`

### Login

```bash
# Huawei Cloud SSO browser login (default)
devbridge auth login

# Login with API Key
devbridge auth login --api-key YOUR_API_KEY

# View login status
devbridge auth status

# Logout
devbridge auth logout
```

You can also provide credentials via environment variables:

```bash
export HW_API_KEY=your_api_key
```

### Language Switching

The CLI outputs English by default. To use Chinese, set the environment variable `DEVBRIDGE_LANG=zh`:

```bash
# Temporary (current terminal)
export DEVBRIDGE_LANG=zh

# Windows PowerShell temporary
$env:DEVBRIDGE_LANG = "zh"

# Windows permanent (user-level environment variable)
[Environment]::SetEnvironmentVariable("DEVBRIDGE_LANG", "zh", "User")
```

### Tunnel Management

```bash
# Create a tunnel
devbridge create my-tunnel -d "dev-env-tunnel" -e 24

# List all tunnels
devbridge list

# View tunnel details (uses default tunnel when ID is omitted)
devbridge show <tunnel-id>

# Update a tunnel
devbridge update <tunnel-id> -n new-name -d "new-desc" -e 48

# Delete a tunnel
devbridge delete <tunnel-id>

# Delete all tunnels
devbridge delete-all

# Issue a tunnel token
devbridge token <tunnel-id> -s host
devbridge token <tunnel-id> -s connect

# Set/clear default tunnel
devbridge set <tunnel-id>
devbridge unset
```

### Port Management

```bash
# Add a port (default protocol auto)
devbridge port create <tunnel-id> -p 8080

# Specify protocol and allow anonymous access
devbridge port create <tunnel-id> -p 3000 --protocol http -a

# Specify protocol and deny anonymous access
devbridge port create <tunnel-id> -p 3000 --protocol https --deny-anonymous

# List ports
devbridge port list <tunnel-id>

# View port details
devbridge port show <tunnel-id> -p 8080

# Update port anonymous policy
devbridge port update <tunnel-id> -p 8080 -a
devbridge port update <tunnel-id> -p 8080 --deny-anonymous

# Delete a port
devbridge port delete <tunnel-id> -p 8080
```

### Remote Port Forwarding

**Host side (expose local services):**

```bash
# Specify tunnel ID: reads the ports bound to this tunnel from the API; the -p argument is ignored
devbridge host <tunnel-id>

# No tunnel ID specified: -p is required, creates a new tunnel and binds the ports automatically
devbridge host -p 8080 -d "dev-env" -e 8

# Multi-port forwarding (creates a tunnel automatically)
devbridge host -p 8080 -p 3000

# Pass a JWT token directly: skips API calls, ports are provided by the gateway
devbridge host <tunnel-id> -t <jwt-token>
```

**Connect side (connect to a remote tunnel):**

```bash
# Default mode: fetches token and port list via the API
devbridge connect <tunnel-id>

# Pass a JWT token directly: skips API calls, ports are negotiated and provided by the Host side via SSH
devbridge connect <tunnel-id> -t <jwt-token>
```

### Quota Query

```bash
# View account quotas and current usage
devbridge limits
```

Output includes: reset time, traffic quota/used, active tunnel count, tunnel/port/Host count limits, tunnel bandwidth limit, per-port HTTP request rate limit, and per-port connection count limit.

### Debugging Tools

```bash
# Start the HTTP echo service (default random port, listens on 127.0.0.1)
devbridge echo

# Specify port and listen address
devbridge echo -p 8080 -i 0.0.0.0

# Ping a URI (default 1000ms interval)
devbridge ping https://<tunnel-id>-8080.cn-north-4-bridge.myhuaweicloud.com

# Specify probe interval (milliseconds)
devbridge ping https://example.com -i 500
```

### Debug Logging

Append `-v/--verbose` to any command to enable Debug-level logging:

```bash
devbridge -v host <tunnel-id>
devbridge connect <tunnel-id> --verbose
```

## Build

### Local Build

```bash
# Development environment
make build-dev

# Test environment
make build-test

# Production environment
make build-prod

# Cross-platform build (6 platforms + SHA256 checksum)
make build-all

# Custom parameters
make build VERSION=1.0.0 SERVER_DOMAIN=https://xxx LOGIN_URL=https://xxx GATEWAY_ADDR=host:port CLUSTER_DOMAIN=xxx
```

### Compile-time Parameter Injection

Configuration is injected at compile time via `-ldflags`:

| Parameter                                                  | Description                                                     |
| ---------------------------------------------------------- | --------------------------------------------------------------- |
| `huawei.com/devbridge/cmd.version`                         | Version number                                                  |
| `huawei.com/devbridge/internal/auth.LoginURL`              | SSO login page URL                                              |
| `huawei.com/devbridge/internal/config.DefaultServerDomain` | API service domain                                              |
| `huawei.com/devbridge/internal/config.ServerAddr`          | Gateway address (host:port)                                     |
| `huawei.com/devbridge/internal/config.ServerHost`          | Cluster domain (used to construct `{tunnelId}.{clusterId}` SNI) |

### CI/CD Build

The project uses GitHub Actions (`.github/workflows/build-cli.yml`); after manual triggering, it supports parallel builds for 6 platforms:

- Linux amd64 / arm64
- Windows amd64 / arm64
- Darwin amd64 / arm64

Build flow: Go compilation → version injection → SHA256 checksum file generation → bake one-line install scripts → publish GitHub Release → upload artifacts to GitCode Release

Artifacts are published to the Release as individual files (not a zip package) and can be downloaded directly via `curl`:

| Artifact                                       | Count                      |
| ---------------------------------------------- | -------------------------- |
| `devbridge_{OS}_{Arch}_{Version}[.exe]`        | 6 platform binaries        |
| `devbridge_{OS}_{Arch}_{Version}[.exe].sha256` | 6 checksum files           |
| `install.sh` / `install.ps1`                   | 2 one-line install scripts |

## Command Reference

```
devbridge [-v|--verbose]                      # Global debug log flag
├── version                                  # Show version
├── auth                                     # Authentication command group
│   ├── login                                # Login
│   ├── logout                               # Logout
│   └── status                               # Login status
├── host [tunnelId]                          # Start the listener side (expose local ports)
├── connect [tunnelId]                       # Start the sender side (connect to a remote tunnel)
├── list                                     # List all tunnels
├── create [name]                            # Create a tunnel
├── show [tunnel-id]                         # View tunnel details
├── update [tunnel-id]                       # Update a tunnel
├── delete [tunnel-id]                       # Delete a tunnel
├── delete-all                               # Delete all tunnels
├── token [tunnel-id]                        # Issue a tunnel token
├── set [tunnel-id]                          # Set default tunnel
├── unset                                    # Clear default tunnel
├── limits                                   # View quotas and usage
├── echo [http]                              # Start the HTTP echo service (for debugging)
├── ping <uri>                               # Ping a URI
└── port                                     # Port management command group
    ├── create [tunnel-id]                   # Add a port
    ├── list [tunnel-id]                     # List ports
    ├── show [tunnel-id]                     # Port details
    ├── delete [tunnel-id]                   # Delete a port
    └── update [tunnel-id]                   # Update a port
```

## Configuration

Config file path: `~/.huawei/devbridge/config.yaml` (directory permissions `0700`, file permissions `0600`)

Stored contents:

| Config item         | Description                    |
| ------------------- | ------------------------------ |
| `credentials`       | Credential info (api_key)      |
| `user-info`         | User info (user_name, user_id) |
| `default-tunnel-id` | Default tunnel ID              |

## Architecture

```
┌─────────────┐    WebSocket + TLS 1.3    ┌──────────────┐    WebSocket + TLS 1.3    ┌─────────────┐
│  Host       │ ◄──────────────────────► │   Gateway    │ ◄──────────────────────► │ Connect     │
│  (Listen)   │    SSH data channel       │  (Relay)     │    SSH data channel       │  (Send)     │
└──────┬──────┘                          └──────────────┘                          └──────┬──────┘
       │                                                                                 │
   Local service ports                                                            Local TCP listener
   (8080, 3000...)                                                          (forward to Host ports)
```

- **Control channel**: WebSocket connection, carries rendezvous signaling
- **Data channel**: SSH over WebSocket, implements port forwarding (Host side `-R` mode)
- **Authentication**: JWT token passed via the `Sec-WebSocket-Protocol` header
- **Tunnel identity**: Passed via the TLS SNI subdomain (`{tunnelId}.{clusterId}.myhuaweicloud.com`)

## Project Structure

```
├── cmd/                    # CLI command layer
│   ├── cli/main.go         # Program entry point
│   ├── root.go             # Root command + version command (includes --verbose global flag)
│   ├── auth.go             # Authentication commands
│   ├── connect.go          # Host/Connect commands
│   ├── tunnel.go           # Tunnel management commands
│   ├── port.go             # Port management commands
│   ├── limits.go           # Quota query command
│   ├── echo.go             # HTTP echo + URI ping debugging commands
│   └── print.go            # Table output helpers
├── internal/               # Internal business logic
│   ├── api/                # REST API client (tunnel/port/quota CRUD + API Key auth + request signing)
│   ├── auth/               # Auth module (SSO login / API Key read & validation / credential persistence)
│   ├── config/             # Config management (YAML read/write)
│   ├── connect/            # Tunnel connection core (WebSocket + SSH tunnel + port forwarding)
│   ├── i18n/               # Internationalization (Chinese/English bilingual + display-width alignment)
│   ├── logging/            # Structured logging (slog Handler + dynamic level)
│   └── netutil/            # Network utilities (URI ping + system proxy detection)
├── scripts/                # Build and release scripts
│   ├── bake-install.sh     # Bake install scripts (inject version number + Release download URL, GoReleaser before.hook)
│   ├── post-release.sh     # Post-release processing (collect artifacts + call GitCode upload)
│   └── upload-gitcode-release.sh  # GitCode Release upload script (includes latest rolling)
├── Makefile                # Local build shortcut commands (build-dev/build-test/build-prod)
├── install.sh              # One-line install script - Bash (remote download + SHA256 checksum + cross-platform)
├── install.ps1             # One-line install script - PowerShell
├── go.mod                  # Go module definition
└── go.sum                  # Go dependency verification
```

## License

[Apache License 2.0](../LICENSE)
