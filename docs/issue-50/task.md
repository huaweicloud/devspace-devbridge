# Task: Issue #50（返工 v2）

## 代码

- [x] `go-sdk/host.go`：`portSetRefresher`（ListPorts 刷新 + 1s 去抖 + 2s 超时 + diff）
- [x] `go-sdk/host.go`：移除每 channel 探测网关通知的推送依赖路径
- [x] `go-sdk/host.go`：静态端口模式忽略刷新
- [x] `go-sdk/host.go`：启动刷新失败时回退首 channel 端口通知（`--token` 模式）
- [x] `go-sdk/models.go`：`TunnelStatus.HostOnline()`
- [x] `go-sdk/internal/i18n`：端口集刷新文案（中/英）
- [x] `cli/cmd/port.go`：`create/update/delete` 降级提示（含重启命令）
- [x] `cli/cmd/connect.go`：已有隧道保持动态端口模式（不 pin 端口）
- [x] `cli/cmd/tunnel.go`：`show` 输出 Host 在线状态
- [x] `cli/internal/i18n`：提示与状态文案（中/英）

## 测试

- [x] 端口集 diff 幂等
- [x] 静态模式忽略刷新
- [x] 新增/删除端口应用到转发集合
- [x] API 失败保留端口集、超时与取消不阻塞
- [x] 并发刷新（`-race`）
- [x] 降级提示含重启命令（中/英）
- [x] `go test -race ./...`（go-sdk）、`go build/vet/test ./...`（cli）

## 文档

- [x] `docs/guide/ports.md` + `docs/zh/guide/ports.md`
- [x] `docs/guide/host.md` + `docs/zh/guide/host.md`
- [x] `docs/guide/tunnels.md` + `docs/zh/guide/tunnels.md`
- [x] `docs/reference/cli.md` + `docs/zh/reference/cli.md`
- [x] `docs/reference/troubleshooting.md` + `docs/zh/reference/troubleshooting.md`
- [x] `npm run lint:docs` 通过

## 待外部完成（阻塞项）

- [ ] 网关（Relay）侧：端口变更下发能力确认，或仅依赖本仓库 ListPorts 刷新即可满足 ≤2s；
      需在网关仓库跟踪并回链本 Issue（维护者提供仓库地址）。
- [ ] 网关（Relay）侧：公网 `404 tunnel not found` 与 `502/503 + Retry-After (host offline)` 判定。
- [ ] 原始复现场景端到端复测：运行中 `host -p 18080` → `port create -p 18081` → 记录状态码与 `time_total`。
- [ ] QA 通过 → `pending_review` → 人工确认关闭。
