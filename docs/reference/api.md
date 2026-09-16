---
title: REST API
description: 使用 DevBridge REST API 管理开发隧道、端口和访问令牌。
---

# REST API

<p class="lead">使用 API Key 管理隧道和端口，获取连接令牌，查询流量余额。</p>

## 服务地址与认证

API 基础地址：

```text
https://bridge.developer.myhuaweicloud.com/open-api-inner/v1/relay-controller
```

本文路径均相对于该基础地址。每次请求携带完整 Key：

```http
X-API-Key: devbridge_<API Key 内容>
```

在 [API Key 管理页面](https://devstation.connect.huaweicloud.com/space/devbridge/apikey)
创建 DevBridge Key，将完整值填入 `X-API-Key`。

```bash
export API_BASE='https://bridge.developer.myhuaweicloud.com/open-api-inner/v1/relay-controller'
export DEVBRIDGE_API_KEY='<完整 DevBridge API Key>'

curl -k -i "$API_BASE/auth/check" -H "X-API-Key: $DEVBRIDGE_API_KEY"
```

成功返回 `204 No Content`。下文请求均携带 `X-API-Key`，
请求和响应使用 JSON，字段采用 camelCase。

示例中的 `-k` 用于跳过服务器证书校验。正式调用建议启用证书校验，
使用系统信任的 CA，或通过 `--cacert` 指定 CA 证书。

## 响应与错误

业务接口成功返回 `200`，响应体为对应的对象、数组或布尔值。
认证检查成功返回 `204`。

失败使用对应的 HTTP 4xx 或 5xx 状态码：

```json
{
  "error": {
    "code": "10002",
    "message": "tunnel not found"
  }
}
```

`error.code` 和 `error.message` 必有；`target` 可选，表示错误字段。
参数校验失败可包含 `details` 数组，每项含 `code`、`target`、`message`。
使用 HTTP 状态和 `error.code` 判断错误，`message` 用于展示说明。

| HTTP 状态 | 常见错误码                | 含义                                   |
| --------- | ------------------------- | -------------------------------------- |
| 400       | `40000`、`11001`          | 参数或端口无效。                       |
| 401       | `40100`                   | Key 缺失、无效或对应身份不可用。       |
| 403       | `40300`、`10005`、`12001` | scope 不允许、无权访问隧道或账户禁用。 |
| 404       | `10001`、`10002`、`11003` | 集群、隧道或端口不存在。               |
| 409       | `10003`、`10007`、`11002` | 隧道 ID、名称或端口冲突。              |
| 410       | `10004`                   | 隧道已过期。                           |
| 429       | `10006`、`11005`、`12002` | 隧道数、端口数或月度流量达到额度。     |
| 429       | `42900`                   | 请求过于频繁，需降低频率并退避重试。   |
| 500       | `50000`、`30001`          | 内部处理或令牌生成失败。               |
| 503       | `50300`                   | 服务暂时不可用，稍后重试。             |

## 接口概览

| 方法   | 路径                                   | 用途                               |
| ------ | -------------------------------------- | ---------------------------------- |
| GET    | `/auth/check`                          | 校验凭据。                         |
| POST   | `/tunnels`                             | 创建隧道。                         |
| GET    | `/tunnels`                             | 查询当前用户的有效隧道。           |
| DELETE | `/tunnels`                             | 删除当前用户的全部隧道。           |
| GET    | `/tunnels/{tunnelId}`                  | 查询详情和运行状态。               |
| PUT    | `/tunnels/{tunnelId}`                  | 更新隧道。                         |
| DELETE | `/tunnels/{tunnelId}`                  | 删除指定隧道。                     |
| POST   | `/tunnels/{tunnelId}/token?scope=host` | 签发 host 或 connect 令牌。        |
| POST   | `/tunnels/upsert-with-token`           | 按名称创建或复用，并签发两个令牌。 |
| POST   | `/tunnels/{tunnelId}/ports`            | 创建端口。                         |
| GET    | `/tunnels/{tunnelId}/ports`            | 查询端口列表。                     |
| GET    | `/tunnels/{tunnelId}/ports/{port}`     | 查询端口详情。                     |
| PUT    | `/tunnels/{tunnelId}/ports/{port}`     | 更新端口策略。                     |
| DELETE | `/tunnels/{tunnelId}/ports/{port}`     | 删除端口。                         |
| GET    | `/limits`                              | 查询限制和余额。                   |

## 创建隧道

```http
POST /tunnels
Content-Type: application/json

{
  "name": "frontend-dev",
  "description": "Frontend development",
  "expiration": 24
}
```

| 请求字段      | 类型    | 必填 | 说明                                            |
| ------------- | ------- | ---- | ----------------------------------------------- |
| `name`        | string  | 是   | 长度 1–128 字符，同一用户的隧道名称唯一。       |
| `description` | string  | 否   | 最长 512 字符。                                 |
| `clusterId`   | string  | 否   | 当前区域的集群 ID；省略使用服务端默认集群。     |
| `expiration`  | integer | 否   | 不活跃时长规格，单位小时，默认 72，范围 1–720。 |
| `type`        | string  | 否   | `bridge` 或 `env`，默认 `bridge`。              |

成功响应：

```json
{
  "name": "frontend-dev",
  "tunnelId": "aaaadysa",
  "tunnelCode": 123456,
  "clusterId": "cn-north-4-bridge",
  "description": "Frontend development",
  "bandwidthUsed": 0,
  "expirationHours": 24,
  "tunnelExpiration": 1789603200,
  "created": 1789516800,
  "url": "aaaadysa.cn-north-4-bridge.myhuaweicloud.com",
  "type": "bridge"
}
```

`url` 是隧道域名；`description` 可为 `null`。
`expirationHours` 是固定时长规格，`tunnelExpiration` 是动态过期时间。
创建后通过令牌接口获取连接令牌。

## 查询隧道

`GET /tunnels` 返回当前用户的有效隧道，可用 `?clusterId=...` 筛选：

```json
[
  {
    "tunnelId": "aaaadysa",
    "tunnelCode": 123456,
    "clusterId": "cn-north-4-bridge",
    "name": "frontend-dev",
    "description": "Frontend development",
    "expirationHours": 24,
    "tunnelExpiration": 1789603200,
    "created": 1789516800,
    "url": "aaaadysa.cn-north-4-bridge.myhuaweicloud.com",
    "portCount": 1
  }
]
```

列表响应为空数组或隧道数组。`GET /tunnels/{tunnelId}` 返回与创建结果相同的字段，
以及可选的运行状态 `status`。以下是 `status` 的内容：

```json
{
  "hostConnectionCount": 1,
  "clientConnectionCount": 2,
  "uploadBytesPerSecond": 1024,
  "downloadBytesPerSecond": 2048,
  "totalUploadBytes": 1048576,
  "totalDownloadBytes": 2097152,
  "reportedAt": 1789516860
}
```

`reportedAt` 是状态更新时间，`clientConnectionCount` 是客户端连接数。
详情的 `bandwidthUsed` 是已结算的上下行总流量，按分钟更新。

## 更新和删除隧道

```http
PUT /tunnels/aaaadysa
Content-Type: application/json

{
  "description": "Updated development environment",
  "expiration": 48
}
```

`name`、`description`、`expiration`、`type` 均可选，省略或传 `null` 保持不变；
清空描述可传空字符串。传入 `expiration` 会从当前时间重新计算过期时间。
更新成功直接返回 JSON 布尔值 `true`。

以下删除接口也返回 `true`，同时移除端口和运行状态：

```text
DELETE /tunnels/{tunnelId}
DELETE /tunnels
```

`DELETE /tunnels` 删除当前用户的全部隧道，包括过期隧道。

## 签发隧道令牌

每次调用签发新令牌。`scope` 是必填查询参数，取值为 `host` 或 `connect`：

```bash
curl -k -X POST "$API_BASE/tunnels/aaaadysa/token?scope=host" -H "X-API-Key: $DEVBRIDGE_API_KEY"
```

```json
{
  "tunnelId": "aaaadysa",
  "scope": "host",
  "lifetime": 86400,
  "expiration": 1789603200,
  "token": "<JWT>"
}
```

`lifetime` 是有效时长，单位为秒；`expiration` 是令牌过期的 Unix 秒。
建立连接时，隧道和令牌均须处于有效期内。请妥善保管令牌。

## 按名称创建或复用并签发令牌

```http
POST /tunnels/upsert-with-token
Content-Type: application/json

{
  "name": "frontend-dev",
  "expiration": 24,
  "ports": [
    {"port": 8080, "protocol": "http", "allowAnonymous": false}
  ]
}
```

支持创建隧道的字段，以及可选的 `ports` 数组，每次最多 10 项：

- 按名称创建或复用隧道；复用有效隧道时保留原有信息和过期时间。
- 保留中的过期隧道会续期，时长取 `expiration`，默认沿用原时长规格。
- `ports` 用于追加端口；已有端口沿用原策略，可通过端口更新接口修改。
- 每次生成新的 Host 和 Connect 令牌。

成功直接返回创建隧道响应的字段，并增加以下两个字段：

```json
{
  "hostToken": {
    "lifetime": 86400,
    "expiration": 1789603200,
    "token": "<Host JWT>"
  },
  "connectToken": {
    "lifetime": 86400,
    "expiration": 1789603200,
    "token": "<Connect JWT>"
  }
}
```

## 创建和查询端口

```http
POST /tunnels/aaaadysa/ports
Content-Type: application/json

{
  "port": 8080,
  "protocol": "http",
  "allowAnonymous": false
}
```

三个字段均必填：普通端口范围为 1–65535；`protocol` 为 `http`、`https` 或 `auto`；
`allowAnonymous` 控制是否允许匿名访问该端口。

成功响应：

```json
{
  "tunnelId": "aaaadysa",
  "tunnelCode": 123456,
  "port": 8080,
  "protocol": "http",
  "allowAnonymous": false
}
```

`GET /tunnels/{tunnelId}/ports` 返回端口数组，空列表为 `[]`。
`GET /tunnels/{tunnelId}/ports/{port}` 返回单个端口对象。

## 更新和删除端口

```http
PUT /tunnels/aaaadysa/ports/8080
Content-Type: application/json

{
  "allowAnonymous": false
}
```

`protocol` 和 `allowAnonymous` 均可选，省略或传 `null` 保持不变。
路径中的 `port` 指定目标端口。成功返回更新后的端口对象。

`DELETE /tunnels/{tunnelId}/ports/{port}` 成功返回 JSON 布尔值 `true`。

## 限制与余额

`GET /limits` 返回：

```json
{
  "resetAt": 1790784000,
  "quotaBytes": 53687091200,
  "remainingBytes": 53683945472,
  "activeTunnels": 1,
  "maxTunnels": 10,
  "maxPortsPerTunnel": 10,
  "maxHostsPerTunnel": 1,
  "maxTunnelBandwidthBytesPerSecond": 5242880,
  "maxHttpRequestsPerMinutePerPort": 500,
  "maxConnectionsPerPort": 100
}
```

示例额度以实际响应为准。`activeTunnels` 是当前用户的有效隧道数；
`maxTunnels` 是账户下所有用户共享的上限。
`quotaBytes` 和 `remainingBytes` 也是账户共享的月度额度和余额，
按北京时间每月 1 日 00:00 重置，`resetAt` 为下次重置的 Unix 秒。

## 字段约定

- `tunnelId` 是隧道标识，访问接口时填入路径的同名参数。
- 请求 `expiration` 和响应 `expirationHours` 单位为小时。
- `tunnelExpiration`、`created`、`reportedAt`、`resetAt` 和令牌 `expiration` 使用 Unix 秒，展示时转换为本地时区。
- 流量字段单位为字节，速率字段单位为字节/秒。
