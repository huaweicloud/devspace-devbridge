---
title: 更新日志
description: DevBridge 命令行工具与服务的变更日志。
---

# 更新日志

## 2026-09-21

### 改进

- 新增 `config` 命令，用于查看与自定义网关连接配置。
- 默认网关地址与 SNI 域名切换至 `devbridge-s2.hwtunnel.com`。
- 支持全端口托管（端口 `-1`），需使用 DevBox 场景的 API Key。
- 同一隧道同一时间仅允许一个 Host 连接，并对重复连接返回更明确的错误提示。

## 2026-08-31

### 改进

- 认证方式由 AK/SK 迁移至 API Key，并新增 `--api-key` 参数与 `HW_API_KEY` 环境变量。
- 登录凭证优先存储于操作系统钥匙串（Keyring）。
- `host` 与 `connect` 支持 API Key 鉴权与默认隧道。
- `connect --token` 模式要求显式指定隧道 ID。
- `devbridge show` 新增 Host/Connect 连接数与上传、下载流量统计。
- 新增 API Key 管理页面，支持在线创建、查看与删除 API Key，并按 DevBridge、DevBox 使用场景区分。

### 移除

- 移除 AK/SK 登录相关参数（`--access-key`、`--secret-key`、`--security-token`）。
- 移除 `list` 与 `port list` 命令的 JSON 输出参数。
- 移除 `--huaweicloud` 与 `loginType` 参数。

## 2026-08-11

### 改进

- 新增 `limits` 命令，用于查看账户配额与当前使用情况。
- 新增 `echo` 与 `ping` 调试工具，用于验证隧道链路。
- 全局新增 `-v` / `--verbose` 参数，用于启用调试级别日志。

## 2026-07-31

### 改进

- 支持通过 Bash 与 PowerShell 安装，覆盖 x86-64 与 ARM64 架构。
- 支持交互登录、AK/SK 登录与临时凭证登录，并提供登录状态查询与退出。
- 隧道支持创建、列表、查看、更新、删除、令牌签发以及默认隧道管理。
- 端口支持增删改查、协议选择与匿名访问控制。
- Host 支持托管多个本地端口、创建临时隧道以及断线自动重连。
- Connect 支持本地端口映射与断线自动重连。
- 提供隧道与端口的 REST API 及相应错误码。
