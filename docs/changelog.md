# 更新日志

本文件记录 DevBridge 文档站点的变更内容。

## [Unreleased]

### 新增

- 新增快速入门页面 `docs/index.md`，覆盖安装、登录、托管本地服务、从另一台设备连接的完整流程。
- 新增指南目录 `docs/guide/`，按用户任务组织以下主题：
  - `overview.md`：介绍开发隧道的工作方式、核心概念、主要能力、隔离与访问、有效期和使用边界。
  - `install.md`：介绍系统要求、Bash 与 PowerShell 安装方式、PATH 配置和验证安装。
  - `authentication.md`：介绍交互登录、AK/SK、临时凭证、查看登录状态、退出登录和凭证使用原则。
  - `tunnels.md`：介绍隧道的创建、列表、详情、更新、默认隧道、令牌签发和删除。
  - `ports.md`：介绍端口的创建、协议选择、匿名访问控制、列表、详情、更新和删除。
  - `host.md`：介绍 Host 模式下托管已有隧道、创建并立即托管、自动重连、停止 Host 和安全建议。
  - `connect.md`：介绍 Connect 模式下建立连接、连接多个端口、受保护端口和自动重连。
- 新增参考目录 `docs/reference/`，提供以下参考内容：
  - `cli.md`：汇总认证、隧道、端口、Host 和 Connect 命令及其参数。
  - `configuration.md`：说明默认目录、PATH、登录凭证、环境变量、默认隧道和本地状态清理。
  - `api.md`：定义 REST API 的服务地址、响应格式、错误码、隧道与端口接口契约。
  - `troubleshooting.md`：覆盖命令找不到、安装失败、登录失败、隧道找不到、配额上限、Host 与 Connect 异常、HTTP/HTTPS 行为等排查路径。
- 新增 `docs/404.md`，提供页面不存在的提示与返回入口。

## [1.0.0] - 初始版本

### 新增

- 建立 DevBridge 文档站点基础结构：
  - `docs/index.md` 作为快速入门。
  - `docs/guide/` 存放按用户任务组织的指南。
  - `docs/reference/` 存放 CLI、配置、API 和问题排查参考。
  - `docs/.vitepress/config.mjs` 配置导航、侧栏、搜索和站点信息。
  - `docs/.vitepress/theme/` 存放小范围主题定制。

### 文档内容要点

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

#### REST API

- 服务基址为 `https://hdspace-partner.cn-north-4.myhuaweicloud.com/open-api-public/v1/relay`。
- 响应外层包含 `error_code`、`error_msg` 和 `result`，成功码为 `0000`。
- 提供隧道与端口的完整 CRUD 接口，以及隧道令牌签发接口。
- 定义 `HD.98310001`、`HD.98300005`、`HD.98320078` 等错误码及处理建议。

#### 配置与目录

- 可执行文件与用户状态分开保存于 `~/.huawei/bin` 和 `~/.huawei/devbridge`。
- 凭证保存在受保护的本地存储，不应复制、提交或输出。
- 设备迁移时应在新设备重新登录，不直接复制配置目录。

#### 问题排查

- 覆盖命令找不到、安装脚本下载失败、登录失败、隧道找不到、配额上限、Host 无法连接、Connect 无法访问、HTTP/HTTPS 行为不正确等场景。
- 提供非敏感信息收集清单用于问题上报。
