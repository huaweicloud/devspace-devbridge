---
title: 登录与凭证
description: 使用交互登录、AK/SK 或临时安全凭证登录 DevBridge。
---

# 登录与凭证

<p class="lead">在创建、托管或连接隧道前，先使用 DevBridge CLI 建立登录状态。</p>

## 交互登录

个人开发环境优先使用交互登录：

```bash
devbridge auth login
```

CLI 会引导你完成身份认证，并将登录凭证保存到本地受保护的凭证存储中。

## 使用 AK/SK

固定服务账号或自动化任务可以使用 AK/SK：

```bash
devbridge auth login \
  --access-key "$DEVBRIDGE_ACCESS_KEY" \
  --secret-key "$DEVBRIDGE_SECRET_KEY"
```

不要把真实 AK/SK 直接写进命令、脚本或流水线定义。应通过受保护的环境变量或密钥服务注入。

## 使用临时凭证

临时 AK/SK 还需要 Security Token：

```bash
devbridge auth login \
  --access-key "$DEVBRIDGE_ACCESS_KEY" \
  --secret-key "$DEVBRIDGE_SECRET_KEY" \
  --security-token "$DEVBRIDGE_SECURITY_TOKEN"
```

临时凭证适合短期任务和自动化环境，可以缩短凭证泄露后的有效窗口。

## 查看登录状态

```bash
devbridge auth status
```

状态命令用于确认当前身份、认证方式和凭证是否有效，不应输出 SK 或完整令牌。

## 退出登录

```bash
devbridge auth logout
```

退出会清除 CLI 保存的登录凭证，但不会删除已创建的隧道。

## 凭证使用原则

- 个人终端使用交互登录；
- 自动化任务优先使用临时凭证；
- 不通过聊天、日志或工单传递 SK；
- 不提交 `~/.huawei/devbridge`；
- 怀疑凭证泄露时立即退出、撤销并重新签发；
- Host 和 Connect 令牌只用于对应隧道和对应连接方向。

## 下一步

- [创建和管理隧道](./tunnels.md)
- [托管本地服务](./host.md)
