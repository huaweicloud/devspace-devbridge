---
title: 本地配置与目录
description: 了解 DevBridge CLI 的安装目录、配置目录、凭证和默认隧道。
---

# 本地配置与目录

<p class="lead">DevBridge CLI 将可执行文件与用户状态分开保存，便于升级程序并保护个人凭证。</p>

## 默认目录

| 路径                  | 内容                                     |
| --------------------- | ---------------------------------------- |
| `~/.huawei/bin`       | DevBridge CLI 可执行文件。               |
| `~/.huawei/devbridge` | 登录状态、CLI 配置、默认隧道和运行状态。 |

安装脚本支持 `-p/--prefix` 参数自定义安装目录，默认为 `~/.huawei/bin`。

## PATH

在当前终端添加 CLI：

```bash
export PATH="$HOME/.huawei/bin:$PATH"
```

如果你使用 PowerShell，则执行：

```powershell
$env:Path = "$HOME/.huawei/bin;$env:Path"
```

若要持续生效，把相应命令加入当前 Shell 的启动文件。Bash 或 Zsh 可写入
`~/.bashrc` 或 `~/.zshrc`，PowerShell 可写入 `$PROFILE`。

## 登录凭证

登录成功后，CLI 将 API Key 保存到受保护的本地存储。凭证存储优先使用系统密钥环（keyring），
密钥环不可用时降级到配置文件。以下内容不应复制、提交或输出：

- API Key；
- Host 和 Connect 令牌；
- CLI 保存的登录状态。

设备迁移时，在新设备重新运行 `devbridge auth login`，不要直接复制整个配置目录。

## 环境变量

自动化环境可以先通过密钥服务注入 `HW_API_KEY`，CLI 会自动读取：

```bash
export HW_API_KEY="your-api-key"
devbridge auth login
```

也可以通过 `--api-key` 参数显式传入：

```bash
devbridge auth login --api-key "$HW_API_KEY"
```

## 默认隧道

设置默认隧道：

```bash
devbridge set <tunnelId>
```

设置后，支持默认上下文的命令可以省略隧道 ID：

```bash
devbridge port list
devbridge host -p 8080
```

清除默认隧道：

```bash
devbridge unset
```

默认隧道是本机 CLI 状态，不会修改服务端资源，也不会授权其他用户。

## 清理本地状态

正常退出登录应使用：

```bash
devbridge auth logout
```

不要通过直接删除配置目录代替正常退出，除非 CLI 已无法运行且你确认不需要保留本地状态。
