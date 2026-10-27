---
title: REST API
description: Use the DevBridge REST API to manage tunnels, ports, and access tokens.
---

# REST API

<p class="lead">Use an API key to manage tunnels and ports, get connection tokens, and query traffic balance.</p>

## Service address and authentication

API base URL:

```text
https://bridge.developer.myhuaweicloud.com/open-api-inner/v1/relay-controller
```

Paths in this document are relative to this base URL. Every request carries the full API key:

```http
X-API-Key: devbridge_<API key content>
```

Create a DevBridge key on the [API key management page](https://devstation.connect.huaweicloud.com/space/devbridge?tab=apikey)
and put the full value into `X-API-Key`.

```bash
export API_BASE='https://bridge.developer.myhuaweicloud.com/open-api-inner/v1/relay-controller'
export DEVBRIDGE_API_KEY='<full DevBridge API key>'

curl -k -i "$API_BASE/auth/check" -H "X-API-Key: $DEVBRIDGE_API_KEY"
```

A successful request returns `204 No Content`. All requests below carry `X-API-Key`;
requests and responses use JSON with camelCase fields.

The `-k` in the examples skips server certificate validation. For production calls,
keep certificate validation on, use a system-trusted CA, or pass a CA certificate via `--cacert`.

## Responses and errors

Business endpoints return `200` on success, with the response body as the corresponding object, array, or boolean.
The authentication check returns `204` on success.

Failures use the corresponding HTTP 4xx or 5xx status code:

```json
{
  "error": {
    "code": "10002",
    "message": "tunnel not found"
  }
}
```

`error.code` and `error.message` are always present; `target` is optional and indicates the error field.
Parameter-validation failures may include a `details` array, each item containing `code`, `target`, and `message`.
Use the HTTP status and `error.code` to decide how to handle the error; `message` is for display only.

| HTTP status | Common error codes        | Meaning                                                             |
| ----------- | ------------------------- | ------------------------------------------------------------------- |
| 400         | `40000`, `11001`          | Invalid parameter or port.                                          |
| 401         | `40100`                   | Key missing, invalid, or the corresponding identity is unavailable. |
| 403         | `40300`, `10005`, `12001` | Scope not allowed, no access to the tunnel, or account disabled.    |
| 404         | `10001`, `10002`, `11003` | Cluster, tunnel, or port not found.                                 |
| 409         | `10003`, `10007`, `11002` | Tunnel ID, name, or port conflict.                                  |
| 410         | `10004`                   | Tunnel has expired.                                                 |
| 429         | `10006`, `11005`, `12002` | Tunnel count, port count, or monthly traffic quota reached.         |
| 429         | `42900`                   | Requests too frequent; slow down and retry with backoff.            |
| 500         | `50000`, `30001`          | Internal processing or token generation failed.                     |
| 503         | `50300`                   | Service temporarily unavailable; retry later.                       |

## API overview

| Method | Path                                   | Purpose                                        |
| ------ | -------------------------------------- | ---------------------------------------------- |
| GET    | `/auth/check`                          | Verify credentials.                            |
| POST   | `/tunnels`                             | Create a tunnel.                               |
| GET    | `/tunnels`                             | Query the current user's valid tunnels.        |
| DELETE | `/tunnels`                             | Delete all of the current user's tunnels.      |
| GET    | `/tunnels/{tunnelId}`                  | Query details and runtime status.              |
| PUT    | `/tunnels/{tunnelId}`                  | Update a tunnel.                               |
| DELETE | `/tunnels/{tunnelId}`                  | Delete a specific tunnel.                      |
| POST   | `/tunnels/{tunnelId}/token?scope=host` | Issue a host or connect token.                 |
| POST   | `/tunnels/upsert-with-token`           | Create or reuse by name and issue both tokens. |
| POST   | `/tunnels/{tunnelId}/ports`            | Create a port.                                 |
| GET    | `/tunnels/{tunnelId}/ports`            | Query the port list.                           |
| GET    | `/tunnels/{tunnelId}/ports/{port}`     | Query a port's details.                        |
| PUT    | `/tunnels/{tunnelId}/ports/{port}`     | Update a port policy.                          |
| DELETE | `/tunnels/{tunnelId}/ports/{port}`     | Delete a port.                                 |
| GET    | `/limits`                              | Query limits and balance.                      |

## Create a tunnel

```http
POST /tunnels
Content-Type: application/json

{
  "name": "frontend-dev",
  "description": "Frontend development",
  "expiration": 24
}
```

| Request field | Type    | Required | Description                                                                             |
| ------------- | ------- | -------- | --------------------------------------------------------------------------------------- |
| `name`        | string  | Yes      | 1–128 characters, unique per user.                                                      |
| `description` | string  | No       | Up to 512 characters.                                                                   |
| `clusterId`   | string  | No       | Cluster ID for the current region; defaults to the server's default cluster if omitted. |
| `expiration`  | integer | No       | Expiration in hours, default 72, range 1–720.                                           |
| `type`        | string  | No       | `bridge` or `env`, default `bridge`.                                                    |

Success response:

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

`url` is the tunnel domain; `description` may be `null`.
`expirationHours` is the fixed duration setting, and `tunnelExpiration` is the dynamically computed expiration time.
After creation, obtain connection tokens via the token endpoint.

## Query tunnels

`GET /tunnels` returns the current user's valid tunnels and can be filtered with `?clusterId=...`:

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

A list response is either an empty array or an array of tunnels. `GET /tunnels/{tunnelId}` returns the same fields as creation,
plus the optional runtime `status`. Here is what `status` contains:

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

`reportedAt` is the time the status was updated, and `clientConnectionCount` is the number of client connections.
The `bandwidthUsed` in details is the settled total upload and download traffic, updated every minute.

## Update and delete a tunnel

```http
PUT /tunnels/aaaadysa
Content-Type: application/json

{
  "description": "Updated development environment",
  "expiration": 48
}
```

`name`, `description`, `expiration`, and `type` are all optional; omit them or pass `null` to leave them unchanged,
and pass an empty string to clear the description. Passing `expiration` recomputes the expiration time from now.
A successful update returns the JSON boolean `true` directly.

The following delete endpoints also return `true`, while removing ports and runtime status:

```text
DELETE /tunnels/{tunnelId}
DELETE /tunnels
```

`DELETE /tunnels` deletes all of the current user's tunnels, including expired ones.

## Issue a tunnel token

Each call issues a new token. `scope` is a required query parameter, either `host` or `connect`:

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

`lifetime` is the validity duration in seconds; `expiration` is the Unix time (in seconds) at which the token expires.
When establishing a connection, both the tunnel and the token must be within their validity periods. Keep tokens safe.

## Create or reuse by name and issue tokens

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

Supports the tunnel-creation fields plus an optional `ports` array of up to 10 items:

- Creates or reuses a tunnel by name; when reusing a valid tunnel, its original details and expiration are kept.
- An expired tunnel that is kept is renewed, with its duration taken from `expiration` (defaulting to the original duration).
- `ports` appends ports; existing ports keep their original policies, which can be changed via the port update endpoint.
- New Host and Connect tokens are generated on each call.

On success, returns the tunnel-creation response plus these two fields:

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

## Create and query ports

```http
POST /tunnels/aaaadysa/ports
Content-Type: application/json

{
  "port": 8080,
  "protocol": "http",
  "allowAnonymous": false
}
```

All three fields are required: the port ranges from 1 to 65535; `protocol` is `http`, `https`, or `auto`;
and `allowAnonymous` controls whether anonymous access to the port is allowed.

Success response:

```json
{
  "tunnelId": "aaaadysa",
  "tunnelCode": 123456,
  "port": 8080,
  "protocol": "http",
  "allowAnonymous": false
}
```

`GET /tunnels/{tunnelId}/ports` returns an array of ports, `[]` when empty.
`GET /tunnels/{tunnelId}/ports/{port}` returns a single port object.

## Update and delete a port

```http
PUT /tunnels/aaaadysa/ports/8080
Content-Type: application/json

{
  "allowAnonymous": false
}
```

`protocol` and `allowAnonymous` are both optional; omit them or pass `null` to leave them unchanged.
The `port` in the path specifies the target port. On success, returns the updated port object.

`DELETE /tunnels/{tunnelId}/ports/{port}` returns the JSON boolean `true` on success.

## Limits and balance

`GET /limits` returns:

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

Example quotas are for illustration only; use the actual response. `activeTunnels` is the current user's number of valid tunnels;
`maxTunnels` is the limit shared by all users under the account.
`quotaBytes` and `remainingBytes` are also account-shared monthly quota and balance, reset at 00:00 Beijing time on the 1st of each month;
`resetAt` is the Unix time (in seconds) of the next reset.

## Field conventions

- `tunnelId` is the tunnel identifier; pass it in the same-named path parameter.
- The request `expiration` and response `expirationHours` are in hours.
- `tunnelExpiration`, `created`, `reportedAt`, `resetAt`, and the token `expiration` use Unix time in seconds; convert to local time for display.
- Traffic fields are in bytes; rate fields are in bytes per second.
