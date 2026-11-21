# Plan: Issue #50

## 目标

在不依赖「网关主动推送端口变更」这一未验证前提的情况下，让运行中的 Host 感知端口增删，
并保证降级路径有明确提示；同时补齐 `show` 在线状态与公网错误码语义文档。

## 实现步骤

1. **端口集运行期刷新（go-sdk）**
   - 新增 `portSetRefresher`：`ListPorts` 取全量端口 → 与缓存 diff → 应用到本次 relay channel。
   - 去抖 1s（`portRefreshMinInterval`）、单次 2s 超时（`portRefreshTimeout`），避免高频请求与阻塞。
   - 启动时 `Refresh(force=true)`；失败回退既有首 channel 端口通知握手（`--token` 模式）。
   - 静态端口模式（`HostConfig.Ports` 非空）直接返回固定端口，不调用 API。
   - API 失败 fail-soft，保留最近端口集；`OnPortsChanged` 仅在 diff 非空时回调。
   - 移除「每个 relay channel 探测网关通知」的旧实现：该路径依赖无证据的推送协议，
     且每个数据 channel 最多阻塞 2s。

2. **CLI 提示与状态（cli）**
   - `port create/update/delete` 成功后统一 `printPortChangeHint`：动态模式约 1s 生效；
     静态模式输出重启命令 `devbridge host <tunnelId> -p <ports>`。
   - `devbridge show` 增加 **Host 在线状态**（`TunnelStatus.HostOnline()`）。
   - 已有隧道解析端口时继续保持「不 pin 端口」，让 Host 走动态刷新。

3. **文档（中/英）**
   - `guide/ports.md`：端口变更生效机制（动态/静态对照表）。
   - `guide/host.md`：端口刷新 + Host 在线状态 + `limits` 账户级边界。
   - `guide/tunnels.md` / `reference/cli.md`：`show` 在线状态字段说明。
   - `reference/troubleshooting.md`：公网 404/503 语义表、服务端边界（跨仓依赖 #50）、新增端口不可达排查。

## 提交拆分（4 组）

1. `feat(sdk): #50 Host 端口集运行期刷新（ListPorts 按需 + 1s 去抖，不依赖网关推送）`
2. `feat(cli): #50 端口变更生效性提示（中/英）+ show 输出 Host 在线状态`
3. `docs: #50 端口生效机制、Host 在线状态与公网状态码语义（中/英）`
4. `docs: #50 任务文档 v2（含验证矩阵与跨仓依赖）`

## 验证计划

| 范围 | 命令 | 结果 |
| ---- | ---- | ---- |
| go-sdk | `go build ./... && go vet ./... && go test -race ./...` | 通过 |
| cli | `go build ./... && go vet ./... && go test ./...` | 通过 |
| docs | `npm run lint:docs` | 通过 |
| 端到端 | 真实隧道执行 `port create` + `curl -w '%{http_code} %{time_total}'` | 本环境无凭据/网关不可达，未执行 |

## 测试覆盖

- `TestPortSetRefresherAppliesRuntimeChanges`：`port create` 后新增端口进入转发集合。
- `TestPortSetRefresherIsIdempotent`：重复刷新不重复回调（diff 幂等）。
- `TestPortSetRefresherRemovesPorts`：`port delete` 后端口移出集合。
- `TestPortSetRefresherStaticModeIgnoresAPIChanges`：静态模式不调用 API、不回调。
- `TestPortSetRefresherDebouncesRapidChannelAccepts`：1s 去抖生效（固定时钟）。
- `TestPortSetRefresherKeepsSetOnAPIFailure` / `TestPortSetRefresherRespectsTimeout` /
  `TestPortSetRefresherCancelledHostContext`：失败与取消时保留端口集、不泄漏阻塞。
- `TestPortNotifierConcurrentUpdates`：并发刷新（`-race`）。
- `TestPortChangeHintShowsRestartCommand`：降级提示必须包含重启命令（中/英）。
