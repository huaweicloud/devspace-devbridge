# Spec: Issue #50 — 运行中隧道新增端口热加载；Host 离线返回可重试状态码（v2）

> 来源：[Issue #50](https://github.com/huaweicloud/devspace-devbridge/issues/50) 与 Spec v2 评论
> （[#issuecomment-6680901379](https://github.com/huaweicloud/devspace-devbridge/issues/50#issuecomment-6680901379)），
> 本文件为开发基线归档。v2 推翻 v1 的两处结论，见下文「与 v1 的差异」。

## Objective

- **缺陷一**：运行中的 Host 执行 `devbridge port create` 新增端口后，新端口不可用，必须重启 Host 才生效，且 CLI 无任何提示。
- **缺陷二**：Host 进程离线（隧道仍存在）时，公网 URL 返回 `404 tunnel not found`，与「隧道不存在/已过期」无法区分。
- **成功标准**：
  1. 原始复现场景下新端口 ≤2s 返回 200，无需重启 Host、无需重连；或网关能力不足时，CLI 明确提示需重启 Host，禁止静默成功。
  2. Host 离线时公网 URL 返回 502/503 + `Retry-After`，body 可区分 `host offline`；404 仅表示隧道不存在/已过期（网关侧交付）。

## 与 v1 的差异

1. 端口热加载必须以**原始复现场景**验收，不能只以「覆盖后续 relay channel」代替。
2. 「网关在端口变更后主动下发全量端口通知」是 PR 旧实现的唯一前提，**无协议证据**，不得默认成立。
   本仓库改为 Host 侧主动刷新端口集（`ListPorts`），该接口是仓库内已验证的 REST 接口，不依赖推送。
3. 问题二的公网状态码属网关（Relay）侧，**docs-only 不计为修复**。本仓库只交付 `show` 在线状态、
   错误码语义与排障、SDK 判定辅助，并在 Issue 记录跨仓依赖。
4. 关闭流程：QA 通过 → `pending_review` → 人工确认后关闭，禁止自动 finalize。

## 现状基线（rebase 后 master）

- `go-sdk/host.go` 的 accept loop 只在首个 relay channel 解析端口通知，`pn.received` 置位后不再解析；
  后续端口变化不会进入转发集合。
- `cli/cmd/connect.go` 对已有隧道调用 `ListPorts` 取得端口，但旧实现把它作为静态列表传入 `HostConfig.Ports`。
- 仓库仅含 CLI / go-sdk / docs，无 Relay/网关服务端代码。

## 方案

### 证据门结果（2026-11-06）

本执行环境无法访问网关（`devbridge-s2.hwtunnel.com` DNS 不可解析、无可用隧道凭据），
因此**未取得网关端口通知协议证据**（无新 relay channel/RelayPayload 样本、无下发时机与频率）。
按 Spec v2 要求：**不采用依赖网关推送的实现路径**，改用方案 A2，并同时落地降级 B。

### A2（已实现）：Host 侧按需刷新端口集

- `portSetRefresher`：每次接受 relay channel 时调用 `ListPorts`，1s 去抖（`portRefreshMinInterval`），
  单次刷新 2s 超时（`portRefreshTimeout`），与缓存端口集 diff 后应用到本次 channel 的 `ForwardFromRemotePort`。
- 启动时先刷新一次；失败则回退到既有「首 channel 端口通知」握手（`--token` 模式唯一端口来源）。
- API 失败 fail-soft：保留最近一次端口集，不中断 Host 会话。
- 静态端口模式（`HostConfig.Ports` 非空）完全不刷新、不调用 API，语义仍是「需重启」。
- `HostConfig.OnPortsChanged(added, removed, current)` 在 diff 非空时回调。

### B（已实现）：降级提示

`port create` / `port update` / `port delete` 成功后统一输出中/英提示：动态端口模式约 1s 自动同步；
静态模式需重启，并给出可复制命令 `devbridge host <tunnelId> -p <ports>`。

### 问题二（本仓库边界）

- `devbridge show` 输出 **Host 在线状态**（`status.hostConnectionCount > 0`，SDK `TunnelStatus.HostOnline()` 助手）。
- docs 中/英给出现行行为与目标语义对照、服务端边界说明、跨仓依赖跟踪。
- 公网 404/502/503+`Retry-After` 判定须由网关实现，未端到端验证前不关闭 Issue。

## Acceptance Criteria

- [ ] **端到端**：运行中 `devbridge host <tunnel-id> -p 18080` 下执行 `port create -p 18081`，≤2s 返回 200；
      附命令、时间戳、状态码、`time_total`。（本环境无法执行，见验证矩阵）
- [x] **证据或降级二选一**：无网关推送证据 → 采用 A2（ListPorts 刷新）+ 降级提示，且不声称依赖网关热加载。
- [x] 降级提示中/英齐全、docs 同步、测试覆盖（`TestPortChangeHintShowsRestartCommand`）。
- [x] `devbridge show` 输出 Host 在线状态（`TestTunnelStatusHostOnline`）。
- [x] 单测：端口集 diff 幂等、静态模式忽略刷新、增删端口、API 失败保留端口集、并发 `-race`。
- [ ] 问题二：网关侧实现 404/502/503+`Retry-After` 并在 Issue 记录依赖链接（跨仓阻塞项）。
- [ ] PR 基于当前 master 重建并通过 CI（本地 build/vet/test 已通过）。
- [ ] 关闭流程：QA 通过 → `pending_review` → 人工确认后关闭。

## 验证矩阵

| 验收项 | 状态 | 依据 |
| ------ | ---- | ---- |
| 端口集刷新逻辑（diff/幂等/静态忽略/失败降级/并发） | 已验证 | `go-sdk`: `go test -race ./...` |
| CLI 降级提示（中/英）与 `show` 在线状态 | 已验证 | `cli`: `go build/vet/test ./...` + i18n 测试 |
| 文档中/英一致性与格式 | 已验证 | `npm run lint:docs`（markdownlint-cli2） |
| 原始复现场景 ≤2s 200 | 未验证 | 本环境无隧道凭据、网关不可达，需维护者按 PR 中的命令复测 |
| 网关端口变更推送协议证据 | 无法在本仓库验证 | 仓库无 Relay 代码；需网关仓库抓包/日志 |
| 公网 404 vs 502/503+`Retry-After` | 无法在本仓库验证 | 网关（Relay）侧实现，跨仓依赖 |

## Risks

| Risk | Impact | Mitigation |
| ---- | ------ | ---------- |
| `ListPorts` 有缓存/延迟导致 ≤2s 不达标 | Med | 1s 去抖 + 2s 超时；端到端复测记录 `time_total`，不达标则依赖降级提示 |
| 网关在 channel 分配阶段就拒绝新端口 | High | 本仓库无法解决，需网关侧配合；降级提示保证用户知情 |
| API 抖动导致刷新失败 | Low | fail-soft 保留上次端口集，仅 debug 日志 |
| 静态模式被误认为自动生效 | Med | 提示文案与 docs 明确区分两种模式，测试断言重启命令存在 |
