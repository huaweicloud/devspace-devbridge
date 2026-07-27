---
title: REST API
description: 使用 DevBridge REST API 管理开发隧道、端口和访问令牌。
---

# REST API

<p class="lead">通过 HTTPS 接口创建和管理开发隧道，并为 Host 或 Connect 连接签发临时令牌。</p>

## 服务地址

华北-北京四区域的 API 基础地址为：

```text
https://hdspace-partner.cn-north-4.myhuaweicloud.com/open-api-public/v1/relay
```

本文中的路径均相对于该地址。例如，创建隧道的完整地址是：

```text
POST https://hdspace-partner.cn-north-4.myhuaweicloud.com/open-api-public/v1/relay/tunnels
```

请求体和响应体均使用 `application/json`，字段名采用下划线命名。
本页面定义公共接口契约；未列出的字段不需要传入，也不保证在响应中返回。

## 响应格式

所有接口使用相同的响应外层结构：

```json
{
  "error_code": "0000",
  "error_msg": "",
  "result": {}
}
```

| 字段         | 类型   | 说明                                  |
| ------------ | ------ | ------------------------------------- |
| `error_code` | string | `0000` 表示成功，其他值表示请求失败。 |
| `error_msg`  | string | 错误说明；请求成功时为空字符串。      |
| `result`     | any    | 接口结果。无业务返回内容时为空对象。  |

请求失败时应同时检查 HTTP 状态码和 `error_code`：

```json
{
  "error_code": "HD.98320078",
  "error_msg": "Tunnel was not found.",
  "result": null
}
```

成功码为 `0000`，失败码格式为 `HD.########`。调用方先根据 HTTP 状态判断错误类别，
再使用错误码处理具体情况：

| 错误码        | 含义                     | 建议处理                      |
| ------------- | ------------------------ | ----------------------------- |
| `HD.98310001` | 请求参数无效。           | 根据 `error_msg` 修正请求。   |
| `HD.98300005` | 身份认证失败。           | 重新获取或更新调用凭证。      |
| `HD.98300013` | 请求过于频繁。           | 降低请求频率并进行退避重试。  |
| `HD.98300014` | 服务繁忙。               | 稍后进行退避重试。            |
| `HD.98300019` | 请求过于频繁。           | 降低请求频率并进行退避重试。  |
| `HD.98320074` | 只有创建者可以查看隧道。 | 使用隧道创建者的身份访问。    |
| `HD.98320075` | 只有创建者可以删除隧道。 | 使用隧道创建者的身份删除。    |
| `HD.98320076` | 隧道名称已经被使用。     | 更换名称后重新提交。          |
| `HD.98320077` | 隧道数量达到配额上限。   | 删除不再使用的隧道后重试。    |
| `HD.98320078` | 隧道不存在。             | 刷新隧道列表，不再使用旧 ID。 |

`HD.98300013` 和 `HD.98300019` 对调用方采用相同的退避处理。不要依赖
`error_msg` 文本进行程序判断，该字段主要用于日志和问题定位。

## 接口概览

### 隧道

| 方法     | 路径                  | 用途           |
| -------- | --------------------- | -------------- |
| `POST`   | `/tunnels`            | 创建隧道。     |
| `GET`    | `/tunnels`            | 查询隧道列表。 |
| `DELETE` | `/tunnels`            | 删除全部隧道。 |
| `GET`    | `/tunnels/{id}`       | 查询隧道详情。 |
| `PUT`    | `/tunnels/{id}`       | 更新隧道。     |
| `DELETE` | `/tunnels/{id}`       | 删除指定隧道。 |
| `POST`   | `/tunnels/{id}/token` | 签发隧道令牌。 |

### 端口

| 方法     | 路径                         | 用途           |
| -------- | ---------------------------- | -------------- |
| `POST`   | `/tunnels/{id}/ports`        | 创建端口。     |
| `GET`    | `/tunnels/{id}/ports`        | 查询端口列表。 |
| `GET`    | `/tunnels/{id}/ports/{port}` | 查询端口详情。 |
| `PUT`    | `/tunnels/{id}/ports/{port}` | 更新端口。     |
| `DELETE` | `/tunnels/{id}/ports/{port}` | 删除端口。     |

## 创建隧道

```http
POST /tunnels
Content-Type: application/json

{
  "name": "frontend-dev",
  "description": "Frontend integration environment",
  "expiration": 24
}
```

| 请求字段      | 类型    | 必填 | 说明                                        |
| ------------- | ------- | ---- | ------------------------------------------- |
| `name`        | string  | 是   | 隧道名称，最长 128 个字符。                 |
| `description` | string  | 否   | 隧道描述，最长 512 个字符。                 |
| `expiration`  | integer | 否   | 有效期规格，单位为小时，默认 72，最大 720。 |

成功响应：

```json
{
  "error_code": "0000",
  "error_msg": "",
  "result": {
    "id": "aaaadysa",
    "name": "frontend-dev",
    "description": "Frontend integration environment",
    "expiration_hours": 24,
    "tunnel_expiration": 1785129600
  }
}
```

`expiration_hours` 是创建或更新时设置的固定小时数；
`tunnel_expiration` 是当前过期时间，使用 Unix 秒。

## 查询隧道

`GET /tunnels` 返回当前身份可访问的隧道列表：

```json
{
  "error_code": "0000",
  "error_msg": "",
  "result": [
    {
      "id": "aaaadysa",
      "name": "frontend-dev",
      "description": "Frontend integration environment",
      "expiration_hours": 24,
      "tunnel_expiration": 1785129600,
      "port_count": 2
    }
  ]
}
```

`GET /tunnels/{id}` 返回单条隧道，字段与创建结果一致。

## 更新和删除隧道

更新时只需提交需要修改的字段：

```http
PUT /tunnels/aaaadysa
Content-Type: application/json

{
  "description": "Updated integration environment",
  "expiration": 48
}
```

`name`、`description` 和 `expiration` 均为可选字段。未提交的字段保持不变。

以下删除接口成功时 `result` 为空对象：

```text
DELETE /tunnels/{id}
DELETE /tunnels
```

`DELETE /tunnels` 会删除当前身份下的全部隧道，请谨慎调用。

## 签发隧道令牌

每次调用都会签发一个新令牌。`scope` 必须是 `host` 或 `connect`：

```http
POST /tunnels/aaaadysa/token
Content-Type: application/json

{
  "scope": "host"
}
```

成功响应：

```json
{
  "error_code": "0000",
  "error_msg": "",
  "result": {
    "tunnel_id": "aaaadysa",
    "scope": "host",
    "lifetime": 3600,
    "expiration": 1785046800,
    "token": "<token>"
  }
}
```

| 字段         | 类型    | 说明                                   |
| ------------ | ------- | -------------------------------------- |
| `tunnel_id`  | string  | 隧道 ID。                              |
| `scope`      | string  | 令牌用途：`host` 或 `connect`。        |
| `lifetime`   | integer | 令牌有效时长，单位为秒。               |
| `expiration` | integer | 令牌过期时间，使用 Unix 秒。           |
| `token`      | string  | Host 或 Connect 建立连接时使用的令牌。 |

令牌属于敏感信息，不要写入日志、代码仓库或长期配置文件。

## 创建端口

```http
POST /tunnels/aaaadysa/ports
Content-Type: application/json

{
  "port": 8080,
  "protocol": "http",
  "allow_anonymous": false
}
```

| 请求字段          | 类型    | 必填 | 说明                                 |
| ----------------- | ------- | ---- | ------------------------------------ |
| `port`            | integer | 是   | 端口号，范围为 1 到 65535。          |
| `protocol`        | string  | 是   | `http`、`https` 或 `auto`。          |
| `allow_anonymous` | boolean | 是   | 是否允许未登录用户通过公网地址访问。 |

成功响应：

```json
{
  "error_code": "0000",
  "error_msg": "",
  "result": {
    "tunnel_id": "aaaadysa",
    "tunnel_code": 123456,
    "port": 8080,
    "protocol": "http",
    "allow_anonymous": false
  }
}
```

## 查询端口

`GET /tunnels/{id}/ports` 返回端口数组：

```json
{
  "error_code": "0000",
  "error_msg": "",
  "result": [
    {
      "tunnel_id": "aaaadysa",
      "tunnel_code": 123456,
      "port": 8080,
      "protocol": "http",
      "allow_anonymous": false
    }
  ]
}
```

`GET /tunnels/{id}/ports/{port}` 返回一条端口记录，字段与创建结果一致。

## 更新和删除端口

`protocol` 和 `allow_anonymous` 均为可选字段，未提交的字段保持不变：

```http
PUT /tunnels/aaaadysa/ports/8080
Content-Type: application/json

{
  "protocol": "https",
  "allow_anonymous": false
}
```

更新成功后，`result` 返回更新后的端口记录。

删除端口：

```text
DELETE /tunnels/aaaadysa/ports/8080
```

删除响应的 `result` 为布尔值，`true` 表示端口已删除。

## 字段约定

- 隧道 ID 是 8 位小写 Base32 字符串。
- `tunnel_code` 是对应的 40 位整数。
- `expiration` 请求参数和 `expiration_hours` 响应字段以小时为单位。
- `tunnel_expiration` 和令牌 `expiration` 使用 Unix 秒。
- 时间字段均按 UTC 时间解释，展示时由客户端转换为本地时区。
