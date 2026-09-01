# 更新日志

本文件记录 DevBridge 文档站点的变更内容。

## 2026-08-31

### 端侧变更

#### 变更

##### 认证

- 登录方式从 AK/SK 迁移到 API Key。
- `devbridge auth login` 新增 `--api-key` 参数，支持直接传入 API Key。
- 移除 `--access-key`、`--secret-key` 和 `--security-token` 参数。
- 新增 `HW_API_KEY` 环境变量支持，CLI 自动读取。
- 浏览器登录回调适配 `{error_code, error_msg, result}` 包装格式，失败错误格式与 API 客户端一致。
- 登录失败时附上 API Key 管理页 URL，提示用户删除后重试。
- 凭证存储改为 Keyring 优先、配置文件降级，移除过期逻辑。

##### Host / Connect

- `host` 和 `connect` 命令新增 `--token` / `-t` 参数，支持直接传入 JWT 令牌，跳过 API 令牌签发和端口查询。
- `host` 和 `connect` 命令新增 `--api-key` / `-k` 参数，支持使用 API Key 鉴权，跳过 TunnelToken 签发。
- `host` 和 `connect` 支持通过 `set` 设置的默认隧道，未指定隧道 ID 时自动使用。
- `connect --token` 模式要求显式指定隧道 ID，不走默认隧道。

##### 隧道详情

- `devbridge show` 输出新增 Host 连接数、Connect 连接数、上传流量和下载流量统计。

#### 移除

- 移除 `list` 和 `port list` 命令的 `-j` / JSON 输出参数。
- 移除 `--huaweicloud` flag 和 `loginType` 参数。

### 服务端变更

#### 新增

##### API Key 管理

- 新增 API Key 创建、查看和删除功能。
- API Key 按 DevBridge、DevBox 使用场景区分，每个场景最多可创建 20 个。
- API Key 完整值仅在创建时展示，列表显示脱敏值和最近使用时间；删除后立即失效。

### 构建与发布

#### 变更

##### CI 流水线

- 新增 `build-cli.yml` 工作流，支持手动触发构建和 PR 自动构建验证。
- PR 触发时仅编译验证（`goreleaser build --snapshot`），不发布 Release。
- 版本号包含 `release` 时才发布到 GitHub Release 和 GitCode Release。

##### GoReleaser

- 构建工具从手写跨平台脚本迁移到 GoReleaser，统一管理 6 平台交叉编译、版本号注入、SHA256 校验和 tar.gz 打包。
- 烤制安装脚本（`bake-install.sh`）作为 GoReleaser `before.hook`，自动注入版本号和 Release 下载地址。
- 移除已废弃的 `scripts/build.sh` 手写构建脚本。

##### GitCode Release

- 构建产物镜像上传到 GitCode Release，支持 `latest` 滚动标签。
- 上传采用两步式 API（upload_url + PUT），增加超时和重试机制。

##### 安装脚本

- 精简为纯远程模式，支持多源自动检测（GitHub / GitCode / OBS）。
- 各渠道独立烤制下载地址，不做跨源 fallback。

## 2026-08-11

### 端侧变更

#### 新增

##### 配额查询

- 新增 `devbridge limits` 命令，查看账户配额与当前用量。
- 输出包含：重置时间、流量配额与已用流量、活跃隧道数、隧道/端口/Host 数量上限、隧道带宽上限、单端口 HTTP 请求频率上限、单端口连接数上限。

##### 调试工具

- 新增 `devbridge echo` 命令，启动 HTTP echo 服务用于验证隧道链路。
  - 默认随机端口，监听 `127.0.0.1`。
  - 支持 `-p` 指定端口、`-i` 指定监听地址。
- 新增 `devbridge ping` 命令，对 URI 发起 HTTP ping 探测。
  - 默认 1000ms 间隔，支持 `-i` 指定间隔（毫秒）。
  - 输出 HTTP 状态码与延迟，例如 `HTTP 200 OK -- 4 ms`。

##### 调试日志

- 全局新增 `-v` / `--verbose` 参数，启用 Debug 级别日志。
- 适用于全部命令，例如 `devbridge -v host <tunnelId>`、`devbridge connect <tunnelId> --verbose`。

## 2026-07-31

### 端侧变更

#### 新增

#### 安装

- 支持 Bash 和 PowerShell 5.1 及以上版本安装。
- CLI 默认安装到 `~/.huawei/bin`，配置保存在 `~/.huawei/devbridge`。
- 支持 x86-64 和 ARM64 架构。

#### 认证

- 支持交互登录 `devbridge auth login`。
- 支持 AK/SK 登录，通过 `--access-key` 和 `--secret-key` 传入。
- 支持临时凭证登录，额外通过 `--security-token` 传入。
- 提供 `devbridge auth status` 查看登录状态，`devbridge auth logout` 退出登录。

#### 隧道

- 隧道 ID 为 8 位小写 Base32 字符串。
- 创建隧道时可指定名称、描述和有效期（小时），默认 72 小时，最大 720 小时。
- 一个工作空间默认最多拥有 10 条有效隧道。
- 支持 `create`、`list`、`show`、`update`、`delete`、`delete-all` 命令。
- 支持通过 `set` 和 `unset` 管理本机默认隧道。
- 支持通过 `token` 命令签发 Host 或 Connect 令牌，令牌具有固定短期有效期。

#### 端口

- 端口范围为 `1` 到 `65535`，同一条隧道内不能重复。
- 协议支持 `http`、`https` 和 `auto`，默认为 `auto`。
- 支持通过 `--allow-anonymous` 和 `--deny-anonymous` 控制匿名访问策略。
- 支持 `port create`、`port list`、`port show`、`port update`、`port delete` 命令。
- 当前 CLI 不支持直接修改已有端口的协议，需要先删除再创建。

#### Host

- 支持托管已有隧道的端口，或创建临时隧道并立即托管。
- 支持同时托管多个本地端口。
- 网络短暂中断时自动重连。
- 停止 Host 后持久隧道和端口配置仍然保留。

#### Connect

- 通过 `devbridge connect <tunnelId>` 建立本地端口映射。
- 自动读取隧道端口配置并建立对应映射。
- 受保护端口需要有效身份和 Connect 令牌，CLI 自动申请。
- 网络短暂中断时自动重连。

#### 配置与目录

- 可执行文件与用户状态分开保存于 `~/.huawei/bin` 和 `~/.huawei/devbridge`。
- 凭证保存在受保护的本地存储，不应复制、提交或输出。
- 设备迁移时应在新设备重新登录，不直接复制配置目录。

#### 问题排查

- 覆盖命令找不到、安装脚本下载失败、登录失败、隧道找不到、配额上限、Host 无法连接、Connect 无法访问、HTTP/HTTPS 行为不正确等场景。
- 提供非敏感信息收集清单用于问题上报。

### 服务端变更

#### REST API

- 服务基址为 `https://hdspace-partner.cn-north-4.myhuaweicloud.com/open-api-public/v1/relay`。
- 响应外层包含 `error_code`、`error_msg` 和 `result`，成功码为 `0000`。
- 提供隧道与端口的完整 CRUD 接口，以及隧道令牌签发接口。
- 定义 `HD.98310001`、`HD.98300005`、`HD.98320078` 等错误码及处理建议。
