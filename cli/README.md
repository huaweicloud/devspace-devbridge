# DevBridge CLI

华为云 DevStation 远程隧道端口转发 CLI 工具，基于 WebSocket + SSH 隧道实现安全的远程端口转发，支持跨平台运行。

## 功能特性

- **隧道管理**：创建、查看、更新、删除隧道，支持设置默认隧道，名称校验首尾不允许连字符
- **端口管理**：为隧道绑定端口，支持 HTTP/HTTPS/Auto 协议及匿名访问控制
- **Host 模式**：将本地服务端口暴露到远程网关，等待远端连接
- **Connect 模式**：连接到远程网关的隧道，在本地创建 TCP Listener 实现端口转发
- **配额查询**：查看账户配额与用量（隧道数、端口数、带宽、流量、连接数等）
- **调试工具**：内置 HTTP echo 服务与 URI ping 探测，便于验证隧道连通性
- **多种认证方式**：支持华为云 SSO 浏览器登录、API Key 命令行参数、环境变量
- **凭证安全存储**：优先使用系统 Keyring，不可用时降级到配置文件
- **断线自动重连**：Host 端指数退避 + 全抖动重试，Connect 端最多 5 次自动重连，SSH 层面支持 session 恢复
- **Token 直传**：Host/Connect 支持直接传入 JWT，跳过 API 调用，适用于离线或受限环境
- **中英双语**：默认英文，可通过环境变量切换

## 技术栈

| 类别 | 技术 |
|------|------|
| 语言 | Go 1.26 |
| CLI 框架 | [cobra](https://github.com/spf13/cobra) |
| WebSocket | [coder/websocket](https://github.com/coder/websocket) |
| SSH 隧道 | [microsoft/dev-tunnels-ssh](https://github.com/microsoft/dev-tunnels-ssh) |
| 凭证存储 | [go-keyring](https://github.com/zalando/go-keyring) |
| 配置文件 | [yaml.v3](https://gopkg.in/yaml.v3) |

## 快速开始

### 安装

**一键安装（Linux / macOS）：**

```bash
# 从 GitHub Release 安装最新版本
curl -fsSL https://github.com/huaweicloud/devspace-devbridge/releases/latest/download/install.sh | bash

# 从 GitHub Release 安装指定版本
curl -fsSL https://github.com/huaweicloud/devspace-devbridge/releases/download/v1.0.0/install.sh | bash

# 从 GitCode Release 安装最新版本
curl -fsSL https://gitcode.com/CloudDeveloperDepartment/devbrige/releases/download/latest/install.sh | bash

# 从 GitCode Release 安装指定版本
curl -fsSL https://gitcode.com/CloudDeveloperDepartment/devbrige/releases/download/v1.0.0/install.sh | bash
```

**一键安装（Windows PowerShell）：**

```powershell
# 从 GitHub Release 安装最新版本
irm https://github.com/huaweicloud/devspace-devbridge/releases/latest/download/install.ps1 | iex

# 从 GitHub Release 安装指定版本
irm https://github.com/huaweicloud/devspace-devbridge/releases/download/v1.0.0/install.ps1 | iex

# 从 GitCode Release 安装最新版本
irm https://gitcode.com/CloudDeveloperDepartment/devbrige/releases/download/latest/install.ps1 | iex
```

安装脚本会自动检测平台、下载对应二进制、校验 SHA256、安装到 `~/.huawei/bin/` 并配置 PATH。

**安装选项（可选）：**

```bash
# 指定安装目录（默认 ~/.huawei/bin）
curl -fsSL <release-url>/install.sh | bash -s -- -p /opt/devbridge

# 静默安装（CI/CD 场景，跳过交互提示）
curl -fsSL <release-url>/install.sh | bash -s -- -s

# 跳过校验和验证
curl -fsSL <release-url>/install.sh | bash -s -- --skip-checksum
```

安装目录：`~/.huawei/bin/`，配置目录：`~/.huawei/devbridge/`

### 登录

```bash
# 华为云 SSO 浏览器登录（默认）
devbridge auth login

# 使用 API Key 登录
devbridge auth login --api-key YOUR_API_KEY

# 查看登录状态
devbridge auth status

# 注销
devbridge auth logout
```

也可以通过环境变量传入凭证：

```bash
export HW_API_KEY=your_api_key
```

### 语言切换

CLI 默认输出英文，如需中文可设置环境变量 `DEVBRIDGE_LANG=zh`：

```bash
# 临时生效（当前终端）
export DEVBRIDGE_LANG=zh

# Windows PowerShell 临时生效
$env:DEVBRIDGE_LANG = "zh"

# Windows 永久生效（用户级环境变量）
[Environment]::SetEnvironmentVariable("DEVBRIDGE_LANG", "zh", "User")
```

### 隧道管理

```bash
# 创建隧道
devbridge create my-tunnel -d "开发环境隧道" -e 24

# 列出所有隧道
devbridge list

# 查看隧道详情（未指定 ID 时使用默认隧道）
devbridge show <tunnel-id>

# 更新隧道
devbridge update <tunnel-id> -n new-name -d "新描述" -e 48

# 删除隧道
devbridge delete <tunnel-id>

# 删除所有隧道
devbridge delete-all

# 颁发隧道令牌
devbridge token <tunnel-id> -s host
devbridge token <tunnel-id> -s connect

# 设置/清除默认隧道
devbridge set <tunnel-id>
devbridge unset
```

### 端口管理

```bash
# 添加端口（默认协议 auto）
devbridge port create <tunnel-id> -p 8080

# 指定协议并允许匿名访问
devbridge port create <tunnel-id> -p 3000 --protocol http -a

# 指定协议并禁止匿名访问
devbridge port create <tunnel-id> -p 3000 --protocol https --deny-anonymous

# 列出端口
devbridge port list <tunnel-id>

# 查看端口详情
devbridge port show <tunnel-id> -p 8080

# 更新端口匿名策略
devbridge port update <tunnel-id> -p 8080 -a
devbridge port update <tunnel-id> -p 8080 --deny-anonymous

# 删除端口
devbridge port delete <tunnel-id> -p 8080
```

### 远程端口转发

**Host 端（暴露本地服务）：**

```bash
# 指定隧道 ID：从 API 读取该隧道已绑定的端口，-p 参数会被忽略
devbridge host <tunnel-id>

# 未指定隧道 ID：必须带 -p，自动创建新隧道并绑定端口
devbridge host -p 8080 -d "开发环境" -e 8

# 多端口转发（自动创建隧道）
devbridge host -p 8080 -p 3000

# 直接传入 JWT token：跳过 API 调用，端口由网关下发
devbridge host <tunnel-id> -t <jwt-token>
```

**Connect 端（连接远程隧道）：**

```bash
# 默认模式：通过 API 获取 token 和端口列表
devbridge connect <tunnel-id>

# 直接传入 JWT token：跳过 API 调用，端口由 Host 端通过 SSH 协商下发
devbridge connect <tunnel-id> -t <jwt-token>
```

### 配额查询

```bash
# 查看账户配额与当前用量
devbridge limits
```

输出包含：重置时间、流量配额/已用、活跃隧道数、隧道/端口/Host 数上限、隧道带宽上限、单端口 HTTP 请求频率上限、单端口连接数上限。

### 调试工具

```bash
# 启动 HTTP echo 服务（默认随机端口，监听 127.0.0.1）
devbridge echo

# 指定端口和监听地址
devbridge echo -p 8080 -i 0.0.0.0

# 对 URI 发起 ping 探测（默认 1000ms 间隔）
devbridge ping https://<tunnel-id>-8080.cn-north-4-bridge.myhuaweicloud.com

# 指定探测间隔（毫秒）
devbridge ping https://example.com -i 500
```

### 调试日志

任一命令追加 `-v/--verbose` 启用 Debug 级别日志：

```bash
devbridge -v host <tunnel-id>
devbridge connect <tunnel-id> --verbose
```

## 构建

### 本地构建

```bash
# 开发环境
make build-dev

# 测试环境
make build-test

# 生产环境
make build-prod

# 跨平台构建（6 平台 + SHA256 校验）
make build-all

# 自定义参数
make build VERSION=1.0.0 SERVER_DOMAIN=https://xxx LOGIN_URL=https://xxx GATEWAY_ADDR=host:port CLUSTER_DOMAIN=xxx
```

### 编译时注入参数

通过 `-ldflags` 在编译时注入配置：

| 参数 | 说明 |
|------|------|
| `huawei.com/devbridge/cmd.version` | 版本号 |
| `huawei.com/devbridge/internal/auth.LoginURL` | SSO 登录页面 URL |
| `huawei.com/devbridge/internal/config.DefaultServerDomain` | API 服务域名 |
| `huawei.com/devbridge/internal/connect.ServerAddr` | 网关地址（host:port） |
| `huawei.com/devbridge/internal/connect.ServerHost` | 集群域名（用于拼接 `{tunnelId}.{clusterId}` SNI） |

### CI/CD 构建

项目使用 GitHub Actions（`.github/workflows/build-cli.yml`），手动触发后支持 6 个平台并行构建：

- Linux amd64 / arm64
- Windows amd64 / arm64
- Darwin amd64 / arm64

构建流程：Go 编译 → 版本号注入 → SHA256 校验文件生成 → 烤制一键安装脚本 → 发布 GitHub Release → 镜像上传到 GitCode Release

产物以散装单文件形式发布到 Release（非 zip 包），可直接 `curl` 下载：

| 产物 | 数量 |
|------|------|
| `devbridge_{OS}_{Arch}_{Version}[.exe]` | 6 个平台二进制 |
| `devbridge_{OS}_{Arch}_{Version}[.exe].sha256` | 6 个校验文件 |
| `install.sh` / `install.ps1` | 2 个一键安装脚本 |

## 命令参考

```
devbridge [-v|--verbose]                      # 全局调试日志标志
├── version                                  # 显示版本
├── auth                                     # 认证命令组
│   ├── login                                # 登录
│   ├── logout                               # 注销
│   └── status                               # 登录状态
├── host [tunnelId]                          # 启动监听端（暴露本地端口）
├── connect [tunnelId]                       # 启动发送端（连接远程隧道）
├── list                                     # 列出所有隧道
├── create [name]                            # 创建隧道
├── show [tunnel-id]                         # 查看隧道详情
├── update [tunnel-id]                       # 更新隧道
├── delete [tunnel-id]                       # 删除隧道
├── delete-all                               # 删除所有隧道
├── token [tunnel-id]                        # 颁发隧道令牌
├── set [tunnel-id]                          # 设置默认隧道
├── unset                                    # 清除默认隧道
├── limits                                   # 查看配额与用量
├── echo [http]                              # 启动 HTTP echo 服务（调试用）
├── ping <uri>                               # 对 URI 发起 ping 探测
└── port                                     # 端口管理命令组
    ├── create [tunnel-id]                   # 添加端口
    ├── list [tunnel-id]                     # 列出端口
    ├── show [tunnel-id]                     # 端口详情
    ├── delete [tunnel-id]                   # 删除端口
    └── update [tunnel-id]                   # 更新端口
```

## 配置

配置文件路径：`~/.huawei/devbridge/config.yaml`（目录权限 `0700`，文件权限 `0600`）

存储内容：

| 配置项 | 说明 |
|--------|------|
| `credentials` | 凭证信息（api_key） |
| `user-info` | 用户信息（user_name, user_id） |
| `default-tunnel-id` | 默认隧道 ID |

## 架构

```
┌─────────────┐    WebSocket + TLS 1.3    ┌──────────────┐    WebSocket + TLS 1.3    ┌─────────────┐
│  Host 端    │ ◄──────────────────────► │   网关       │ ◄──────────────────────► │ Connect 端  │
│  (Listen)   │    SSH 数据通道           │  (Relay)     │    SSH 数据通道           │  (Send)     │
└──────┬──────┘                          └──────────────┘                          └──────┬──────┘
       │                                                                                 │
   本地服务端口                                                                    本地 TCP Listener
   (8080, 3000...)                                                               (转发到 Host 端口)
```

- **控制通道**：WebSocket 连接，传递 rendezvous 信令
- **数据通道**：SSH over WebSocket，实现端口转发（Host 端 `-R` 模式）
- **认证**：JWT 令牌通过 `Sec-WebSocket-Protocol` 头传递
- **隧道标识**：通过 TLS SNI 子域名传递（`{tunnelId}.{clusterId}.myhuaweicloud.com`）

## 项目结构

```
├── cmd/                    # CLI 命令层
│   ├── cli/main.go         # 程序入口
│   ├── root.go             # 根命令 + 版本命令（含 --verbose 全局标志）
│   ├── auth.go             # 认证命令
│   ├── connect.go          # Host/Connect 命令
│   ├── tunnel.go           # 隧道管理命令
│   ├── port.go             # 端口管理命令
│   ├── limits.go           # 配额查询命令
│   ├── echo.go             # HTTP echo + URI ping 调试命令
│   └── print.go            # 表格输出辅助
├── internal/               # 内部业务逻辑
│   ├── api/                # REST API 客户端（隧道/端口/配额 CRUD + API Key 认证 + 请求签名）
│   ├── auth/               # 认证模块（SSO 登录/API Key 读取校验/凭证持久化）
│   ├── config/             # 配置管理（YAML 读写）
│   ├── connect/            # 隧道连接核心（WebSocket + SSH 隧道 + 端口转发）
│   ├── i18n/               # 国际化（中/英双语 + 显示宽度对齐）
│   ├── logging/            # 结构化日志（slog Handler + 动态级别）
│   └── netutil/            # 网络工具（URI ping + 系统代理探测）
├── scripts/                # 构建与发布脚本
│   ├── bake-install.sh     # 烤制安装脚本（注入版本号 + Release 下载地址，GoReleaser before.hook）
│   ├── post-release.sh     # Release 后处理（收集产物 + 调用 GitCode 上传）
│   └── upload-gitcode-release.sh  # GitCode Release 上传脚本（含 latest 滚动）
├── Makefile                # 本地构建快捷命令（build-dev/build-test/build-prod）
├── install.sh              # 一键安装脚本 - Bash（远程下载 + SHA256 校验 + 跨平台）
├── install.ps1             # 一键安装脚本 - PowerShell
├── go.mod                  # Go 模块定义
└── go.sum                  # Go 依赖校验
```

## 许可证

[Apache License 2.0](../LICENSE)
