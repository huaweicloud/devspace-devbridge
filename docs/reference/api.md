---
title: REST API
description: 使用 DevBridge REST API 管理开发隧道、端口和访问令牌。
---

# REST API

<p class="lead">通过 HTTPS 直接访问 Relay，使用 API Key 管理隧道和端口，并为 Host 或 Connect 连接签发临时令牌。</p>

## 服务地址与认证

公网域名为 `https://bridge.developer.myhuaweicloud.com`，API 基础地址为：

```text
https://bridge.developer.myhuaweicloud.com/open-api-inner/v1/relay-controller
```

本文路径均相对于该基础地址。每次请求携带完整 Key：

```http
X-API-Key: devbridge_<API Key 内容>
```

不加 `Bearer` 前缀，不对 Key 再做 Base64 编码。服务端通过 Key 解析 Namespace，
调用方无需传入 Namespace、账户或用户 ID。DevBridge 使用 `devbridge_` Key；
当前 Relay 也接受 `devbox_` Key，部分操作另有 scope 限制。

连接使用单向 HTTPS，无需客户端证书。当前 CLI 跳过服务端证书校验，等同于
`curl -k`，这是独立于单向 TLS 的设置。`-k` 保留传输加密，但不验证服务器身份；
能够正常校验证书时应去掉 `-k`，私有 CA 可通过 `--cacert` 指定。

```bash
export API_BASE='https://bridge.developer.myhuaweicloud.com/open-api-inner/v1/relay-controller'
export DEVBRIDGE_API_KEY='<完整 DevBridge API Key>'

curl -k -i "$API_BASE/auth/check" -H "X-API-Key: $DEVBRIDGE_API_KEY"
```

成功返回 `204 No Content`，无响应体。下文 HTTP 示例也需携带该 Header。
请求和响应使用 JSON，字段采用 camelCase；未知请求字段会被拒绝。

## 响应与错误

业务接口成功返回 `200`，直接返回对象、数组或布尔值，不包含 `result`、
`error_code`、`error_msg` 外层。只有认证检查成功返回 `204`。

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
调用方根据 HTTP 状态和 `error.code` 判断，不依赖错误文本。

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
| 503       | `50300`                   | 认证依赖暂时不可用。                   |

## 接口概览

| 方法   | 路径                                   | 用途                               |
| ------ | -------------------------------------- | ---------------------------------- |
| GET    | `/auth/check`                          | 校验凭据。                         |
| POST   | `/tunnels`                             | 创建隧道。                         |
| GET    | `/tunnels`                             | 查询当前 Namespace 的有效隧道。    |
| DELETE | `/tunnels`                             | 删除当前 Namespace 的全部隧道。    |
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
| `name`        | string  | 是   | 非空，同一 Namespace 内唯一，最长 128 字符。    |
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

`url` 是不含协议的隧道域名；未设置描述时 `description` 为 `null`。
`expirationHours` 是固定时长规格，`tunnelExpiration` 是动态过期时间。
创建接口不返回令牌，连接前需签发令牌。

## 查询隧道

`GET /tunnels` 返回当前 Namespace 未删除且未过期的隧道，可用 `?clusterId=...` 筛选：

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

无记录时返回 `[]`。`GET /tunnels/{tunnelId}` 返回与创建结果相同的字段，
有网关上报记录时增加 `status`。以下是该字段的对象内容：

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

未上报时省略 `status`；`reportedAt` 是最近上报时间，不是查询时间。
`clientConnectionCount` 是活跃 SSH channel 数。
详情的 `bandwidthUsed` 是已结算上下行总量，可能晚于网关上报的累计流量。

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

`DELETE /tunnels` 删除当前 Namespace 全部隧道，包括已过期但尚未清理的记录。

## 签发隧道令牌

每次调用签发新令牌。`scope` 是必填查询参数，取值为 `host` 或 `connect`，无需请求体：

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

`lifetime` 单位为秒，当前为 24 小时；`expiration` 为令牌过期的 Unix 秒。
有效期独立于隧道期限，令牌仍有效不代表隧道仍可用。
响应含 `Cache-Control: no-store`，不要将令牌写入日志或仓库。

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

- 没有同名隧道时创建；已有有效隧道时复用，保留现有元数据和过期时间。
- 同名隧道已过期但尚未清理时续期；未传 `expiration` 时沿用原时长规格。
- `ports` 只补充不存在的端口，已有端口策略保持不变，修改需调用端口更新接口。
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
`allowAnonymous` 控制是否允许匿名访问该端口。特殊端口 `-1` 的创建仅允许 `devbox` scope。

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

`GET /tunnels/{tunnelId}/ports` 返回上述对象的数组，无端口时返回 `[]`。
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
端口号只放在路径中，更新请求体不能传 `port`。成功直接返回更新后的端口对象。

`DELETE /tunnels/{tunnelId}/ports/{port}` 成功返回 JSON 布尔值 `true`。
不提供端口批量删除接口。

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

示例额度以实际响应为准。`activeTunnels` 只统计当前 Namespace 的有效隧道；
`maxTunnels` 是账户下所有 Namespace 共享的上限。
`quotaBytes` 和 `remainingBytes` 也是账户共享的月度额度和余额，
按北京时间每月 1 日 00:00 重置，`resetAt` 为下次重置的 Unix 秒。

## 字段约定

- `tunnelId` 是 8 位小写 Base32 字符串，`tunnelCode` 是对应的 40 位整数。
- 请求 `expiration` 和响应 `expirationHours` 单位为小时。
- `tunnelExpiration`、`created`、`reportedAt`、`resetAt` 和令牌 `expiration` 使用 Unix 秒；展示时转换为本地时区，不额外加减时区偏移。
- 流量字段单位为字节，速率字段单位为字节/秒。
