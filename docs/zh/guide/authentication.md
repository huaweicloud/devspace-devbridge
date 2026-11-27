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

CLI 会打开浏览器引导你完成身份认证，并将登录凭证保存到本地受保护的凭证存储中。凭证存储优先使用
系统密钥环（keyring），密钥环不可用时降级到配置文件。

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

### 获取 API Key

你可以在 DevBridge 管理页面创建和管理 API Key：

> **API Key 管理页面**：<https://devstation.connect.huaweicloud.com/space/devbridge?tab=apikey>

在该页面创建新的 API Key 后，将其复制并通过环境变量 `HW_API_KEY` 注入 CLI 即可使用。

## 无头与服务环境

CLI 的凭证存放在两处之一，并在进程启动时决定用哪一个：

- 系统密钥环（经 D-Bus 会话总线访问的 Secret Service）；
- 配置文件 `~/.huawei/devbridge/config.yaml`。

桌面会话、`dbus-run-session` 启动的进程，以及设置了常驻 `DBUS_SESSION_BUS_ADDRESS` 的进程，会使用密钥环。判定在进程内只做一次，顺序为：

1. `DBUS_SESSION_BUS_ADDRESS` 指向真实地址（`autolaunch:` 不算）；
2. `$XDG_RUNTIME_DIR/bus` 存在且是 socket；
3. `$HOME/.dbus/session-bus` 下的文件中存在 `DBUS_SESSION_BUS_ADDRESS=` 行。

命中后两条时，CLI 会把探测到的地址导出到环境变量，让 D-Bus 客户端复用它，而不是另起一条总线。

三者都不满足时——容器、沙箱、systemd 服务，以及任何没有会话总线的主机——CLI **完全不调用密钥环**。这一点比字面上更重要：没有会话总线时，一次密钥环调用会让 D-Bus 客户端执行 `dbus-launch`，fork 出一个无人回收的
`dbus-daemon --session`；长期运行的 `devbridge host` 会因此每次读写凭据泄漏一个守护进程。跳过密钥环才能让进程数保持恒定。

该模式下，凭据的读写都走配置文件，CLI 会在进程内提示一次：

```text
未检测到可用的 D-Bus 会话总线，凭证改用本地配置文件存储：
~/.huawei/devbridge/config.yaml（目录权限 0700，文件权限 0600）。
该文件以 base64 编码保存，并非加密；如需使用系统密钥环（Secret Service），
请在桌面会话或 dbus-run-session 下运行。
```

请把它理解为有意的取舍，而不是故障：

- 凭据以 Base64 编码保存、**不做加密**，其保密性依赖于 `0700` 目录权限、`0600` 文件权限，以及主机为单用户使用；
- 共享主机上应优先通过受保护的环境变量（或密钥服务）注入 `HW_API_KEY`，避免凭据落盘；
- 如果希望在无头主机上继续使用密钥环，请给进程一条真实总线，而不是每次调用临时拉起：

```bash
dbus-run-session -- devbridge host --tunnel <tunnel-id> --host-token <token> --ports 8080
```

要确认某个进程实际选中了哪种后端，加上 `-v` 查看总线探测日志：

```bash
devbridge auth status -v
```

```text
level=DEBUG msg="d-bus session bus probe" available=false source=none
```

找到总线时 `source` 为 `env`、`xdg-runtime-dir` 或 `session-bus-file`；使用配置文件时为 `none`。Windows 与 macOS 始终使用系统自带的凭证存储（凭据管理器、Keychain），本节不适用。

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

不要通过直接删除配置目录代替正常退出，除非 CLI 已无法运行且你确认不需要保留本地状态。

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
