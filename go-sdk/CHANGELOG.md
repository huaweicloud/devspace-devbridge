# Changelog

本文件记录 DevBridge Go SDK 的用户可见变更，版本遵循语义化版本（SemVer）。

## 2026-11-23

首个公开发布版本（v0.1.0-release）。

- 隧道管理：创建、查询、更新、删除
- 端口管理：创建、查询、更新、删除
- 令牌签发：Host / Connect 令牌
- Host 托管：通过 WebSocket + SSH 隧道转发本地端口，支持 `OnReady` 就绪回调与断线自动重连
- Connect 连接：连接远端隧道并建立本地端口映射，支持 `OnReady` 就绪回调与断线自动重连
- 客户端配置：API Key、自定义网关地址、自定义 HTTP 客户端、WebSocket 握手请求头、状态输出控制
- 错误处理：提供 `ErrMissingAPIKey`、`ErrTunnelNotFound`、`ErrDuplicateHost`、`ErrQuotaExceeded` 等 sentinel error，支持 `IsAPIError` 提取业务错误码
- 国际化支持：通过 `DEVBRIDGE_LANG` 环境变量切换语言
