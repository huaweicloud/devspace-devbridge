# Plan: Issue #45 — delete-all 清理默认隧道 + 确认机制

> 分支：`feat/issue-45`（平台预置 worktree，直接使用）
> 顺序：RED → GREEN → 原子提交 → 全量回归

## 实施步骤

| 步骤 | 内容 | 产出 |
|------|------|------|
| 1 | i18n：新增 4 个文案 key（FlagYes / ConfirmDeleteAll / DeleteAborted / ConfirmNonTTY） | `messages.go` + `translations.go` |
| 2 | 确认门：新增 `cli/cmd/confirm.go`（`confirmOrAbort(cmd, prompt)`：--yes 放行；TTY y/N；非 TTY 报错） | `confirm.go` |
| 3 | RED：先写 `cli/cmd/tunnel_test.go` 三条回归路径 + 确认函数单测 | `tunnel_test.go` |
| 4 | GREEN：deleteAllCmd 接入确认门 + 成功后 `config.DeleteDefaultTunnel()` 并提示 | `tunnel.go` |
| 5 | 提交：拆 2 个原子 commit（① 确认函数+i18n；② delete-all 接入+测试） | git |
| 6 | 验证：`go build ./... && go vet ./... && go test ./...` 全绿 | 本地 |
| 7 | 五轴自审查（正确性/安全/性能/可维护性/测试）→ 修复问题 | — |
| 8 | 建 PR（base=master）→ Issue 评论 → A2A 通知 QA | PR |

## 测试矩阵（对齐 spec §5）

1. delete-all 成功（--yes）→ 默认隧道清除 + 提示
2. 非 TTY 无 --yes → 报错、不删、默认隧道保留
3. 删除 API 失败 → 默认隧道不清除
4. TTY 回答 y → 放行
5. TTY 回答 N → 中止、不删

## 原子提交规划

- `feat: #45 tunnel delete-all 增加 --yes/TTY 确认门`（confirm.go + i18n + 确认相关测试）
- `fix: #45 delete-all 成功后清除默认隧道`（tunnel.go deleteAllCmd 清理 + 回归测试）

## 验证命令

```bash
cd cli && go build ./... && go vet ./... && go test ./...
```
