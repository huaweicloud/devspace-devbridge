---
title: 登录与凭证
description: 使用交互登录或 API Key 登录 DevBridge。
---

# 登录与凭证

<p class="lead">在创建、托管或连接隧道前，先使用 DevBridge CLI 建立登录状态。</p>

## 交互登录

个人开发环境优先使用交互登录：

```bash
devbridge auth login
```

CLI 会打开浏览器引导你完成身份认证，并将登录凭证保存到本地受保护的凭证存储中。

## 使用 API Key

固定服务账号或自动化任务可以使用 API Key 直接登录：

```bash
devbridge auth login --api-key "$HW_API_KEY"
```

也可以通过环境变量注入，CLI 会自动读取：

```bash
export HW_API_KEY="your-api-key"
devbridge auth login
```

不要把真实 API Key 直接写进命令、脚本或流水线定义。应通过受保护的环境变量或密钥服务注入。

## 查看登录状态

```bash
devbridge auth status
```

状态命令用于确认当前身份和凭证是否有效，不应输出完整 API Key。

## 退出登录

```bash
devbridge auth logout
```

退出会清除 CLI 保存的登录凭证，但不会删除已创建的隧道。

## 凭证使用原则

- 个人终端使用交互登录；
- 自动化任务使用 API Key，通过环境变量注入；
- 不通过聊天、日志或工单传递 API Key；
- 不提交 `~/.huawei/devbridge`；
- 怀疑凭证泄露时立即退出、撤销并重新签发；
- Host 和 Connect 令牌只用于对应隧道和对应连接方向。

## 下一步

- [创建和管理隧道](./tunnels.md)
- [托管本地服务](./host.md)
