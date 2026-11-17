# Spec: Issue #45 — delete-all 删除全部隧道后未清理默认隧道，且无确认

> 来源：Issue #45 Spec Document 评论（Issue Agent 产出，Developability: ready）
> 实现分支：`feat/issue-45`

## 1. 问题背景

`devbridge tunnel delete-all` 存在两个缺陷：

1. **成功删除全部隧道后未清除默认隧道**：配置文件（`~/.huawei/devbridge/config.yaml`）中的 `default-tunnel-id` 仍指向已删除的隧道，后续不带 ID 的命令（`tunnel show` / `port list` / `host` / `connect`）会请求一个已不存在的隧道而报错。
2. **无确认机制**：delete-all 会删除账号下**全部**隧道，误触代价大，但命令执行前没有任何交互确认或强制 flag。

## 2. 目标与非目标

### 目标
- delete-all 成功后调用 `config.DeleteDefaultTunnel()` 清除 `default-tunnel-id` 并提示用户。
- 新增 `--yes`/`-y` flag：跳过交互确认（供 CI/管道使用）。
- TTY 场景下出现 `y/N` 确认；非 TTY 且未传 `--yes` 时报错退出并提示加 `--yes`。
- 确认逻辑抽成可复用函数（后续 `tunnel delete` 可接入，本次不接入）。
- i18n 新增确认/拒绝文案（中/英）。
- 回归测试覆盖三条验收路径。

### 非目标
- `tunnel delete` 接入确认逻辑（仅保证函数可复用）。
- 服务端 / SDK 改动。
- i18n 新文案以外的 UI 变更。

## 3. 方案设计

### 3.1 确认门（执行前）

```
delete-all 执行 → confirmOrAbort()
  ├─ --yes/-y flag 已设置          → 直接放行
  ├─ stdin 是 TTY                  → 打印 "Delete all tunnels? [y/N]"，读一行
  │    ├─ y/Y                      → 放行
  │    └─ N/回车/其他              → 返回"已取消"错误，中止
  └─ 非 TTY 且无 --yes             → 返回错误："非交互环境需要 --yes"
```

### 3.2 默认隧道清理（删除成功后）

对齐 `deleteCmd` 现有行为（cli/cmd/tunnel.go:173-176）：

```
DeleteAllTunnels 成功
  → config.DeleteDefaultTunnel()   // 幂等：key 不存在视为已删除
  → 打印 DefaultTunnelCleared 提示
  → 打印 TunnelDeletedAll 提示
DeleteAllTunnels 失败
  → 原样返回错误，不清默认隧道
```

### 3.3 关键设计决策

| 决策 | 选项 | 选择 | 理由 |
|------|------|------|------|
| 非 TTY 无 --yes 行为 | 静默跳过 / 报错退出 | 报错退出 | 防管道误删，Issue 明确要求 |
| 清理时机 | 删除前 / 成功后 | 成功后 | 删除失败不清默认隧道 |
| 确认函数位置 | 内联 / 抽取复用 | 抽取 | Issue 建议复用到 tunnel delete |
| TTY 检测 | `os.Stdin.Stat()` mode 判断 | 采用 | 纯标准库，无新依赖，测试可重定向模拟 |

## 4. 影响模块

| 模块 | 改动 |
|------|------|
| `cli/cmd/tunnel.go` | deleteAllCmd：确认门 + 成功后清理默认隧道 |
| `cli/cmd/confirm.go`（新增） | 可复用 `confirmOrAbort`（--yes 检查 + TTY 检测 + y/N 交互） |
| `cli/internal/i18n/messages.go` + `translations.go` | 新增 `Tunnel.FlagYes`、`Tunnel.ConfirmDeleteAll`、`Tunnel.DeleteAborted`、`Tunnel.ConfirmNonTTY` |
| `cli/cmd/tunnel_test.go`（新增） | 三条回归路径测试 |

## 5. 测试计划（TDD）

| # | 场景 | 断言 |
|---|------|------|
| 1 | delete-all 成功（--yes）且有默认隧道 | 默认隧道被清除 + 输出清除提示 + 删除成功提示 |
| 2 | 非 TTY 且无 --yes | 报错退出、提示 `--yes`、不调用删除 API、默认隧道保留 |
| 3 | DeleteAllTunnels 失败 | 返回错误、默认隧道**未**被清除 |
| 4 | TTY 输入 y | 放行（注入 fake confirm reader） |
| 5 | TTY 输入 N/回车 | 中止、不调用删除 API |

时间字段断言规范：本任务不涉及时序字段，无需特殊处理。

## 6. 风险与缓解

| 风险 | 缓解 |
|------|------|
| TTY 检测在测试环境不可用 | 测试用 `os.Stdin` 重定向模拟非 TTY；confirm 函数支持注入 reader 与 TTY 判定 |
| 破坏现有 delete-all CI 脚本 | 行为变更符合 Issue 要求，PR 描述注明 breaking change |
