---
title: OpenAPI
description: Relay Controller 集成 API 的使用边界和 OpenAPI 定义。
---

# OpenAPI

<p class="lead">普通用户应优先使用 DevBridge CLI；Relay Service、Relay Gateway 和受控集成方可以使用 Relay Controller API。</p>

## 下载定义

[下载 OpenAPI YAML](../openapi.yaml){target="_blank"}

OpenAPI 文件描述请求路径、字段、校验规则、响应和错误结构。生成客户端或对接接口时，应以该文件为准。

## 服务边界

Relay Controller 负责：

- 隧道元数据；
- 端口策略；
- Host 和 Connect 令牌签发；
- 流量计量记录；
- 隧道配额、有效期和区域边界。

Relay Controller 不转发业务流量。Host 与 Connect 的数据连接由 Relay Service 和 Relay Gateway 处理。

## 请求范围

用户范围接口需要 `X-Namespace` 请求头。隧道只能操作当前命名空间中的资源。

集群相关接口还会验证 `clusterId` 是否属于当前区域。调用方不能仅凭已知的隧道 ID 跨命名空间或跨集群访问资源。

## 主要资源

| 资源                               | 能力                            |
| ---------------------------------- | ------------------------------- |
| `/tunnels`                         | 创建、列出和批量删除隧道。      |
| `/tunnels/{tunnelId}`              | 查询、更新和删除单条隧道。      |
| `/tunnels/{tunnelId}/ports`        | 创建和列出端口策略。            |
| `/tunnels/{tunnelId}/ports/{port}` | 查询、更新和删除端口策略。      |
| `/tunnels/{tunnelId}/token`        | 签发新的 Host 或 Connect 令牌。 |
| `/clusters/{clusterId}/metering`   | 上报隧道流量。                  |

完整路径统一位于：

```text
/open-api-inner/v1/relay-controller
```

## 认证与传输

服务端启用 mTLS 时，调用方必须使用受信任客户端证书。证书用于建立服务级可信连接；命名空间和集群仍由上层身份与业务校验约束。

令牌接口每次调用都会签发新的短期 JWT，不缓存令牌。调用方应：

- 只把 Host 令牌交给 Host；
- 只把 Connect 令牌交给 Connect；
- 校验签名算法、密钥 ID、有效期、集群和作用域；
- 不记录或长期存储完整令牌。

## 错误响应

参数、资源和权限问题使用对应的 4xx HTTP 状态；未预期的服务故障使用 5xx。错误体以 `error` 字段为入口：

```json
{
  "error": {
    "code": "PARAM_INVALID",
    "message": "Request parameter is invalid",
    "target": "port"
  }
}
```

客户端应先根据 HTTP 状态判断错误类别，再根据 `error.code` 做稳定分支，不依赖完整错误文案。
