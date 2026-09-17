# Changelog

本文件记录 DevBridge Go SDK 的用户可见变更，版本遵循语义化版本（SemVer）。

## Unreleased

### Changed（Breaking）

- 包结构重构：
  - 包名 `devbridge` → `sdk`（与模块路径末段一致）
  - 业务对象 `Client` 更名 `Devbridge`，构造函数 `NewClient` → `New`
  - HTTP 通信实现下沉至 `internal/httpclient`（外部不可导入）
- 迁移方式：将 `devbridge.` 前缀替换为 `sdk.`，`NewClient(...)` 替换为 `New(...)`

## v0.1.0（2026-09-14）

首个公开发布版本。

### Added

- 隧道管理：创建、查询、更新、删除
- 端口管理：创建、查询、更新、删除
- 令牌签发：Host / Connect
- Host 托管与 Connect 连接：自动重连、`OnReady` 就绪回调
- 状态输出可配置：`Config.StatusWriter`（默认 os.Stdout，可静音）
- 单元测试覆盖配置校验逻辑

### Changed

- 模块路径：`github.com/huaweicloud/devspace-devbridge/go-sdk`
- 隧道描述校验：仅允许中文、字母、数字，长度 0-64
- README 全面更新：配置说明、错误处理、示例与许可信息

### Removed

- 移除 `sdk/cmd` 调试服务与联调脚本（验证职责由单元测试与 CI 接管）
