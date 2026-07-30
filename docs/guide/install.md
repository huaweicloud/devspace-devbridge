---
title: 安装 DevBridge CLI
description: 使用官方安装脚本安装并验证 DevBridge CLI。
---

# 安装 DevBridge CLI

<p class="lead">DevBridge CLI 提供隧道、端口、Host、Connect 和凭证管理命令。</p>

## 系统要求

- Bash 或 PowerShell；
- `curl`（适用于 Bash 安装方式）；
- `irm`（适用于 PowerShell 安装方式）；
- 可以访问 DevBridge 安装源；
- 对 `~/.huawei` 目录具有写权限。

## 安装

运行：

```bash一键安装
curl -fsSL https://res-hd.hc-cdn.cn/sharedata/hdspace/devbridge/install.sh | bash
安装后执行
source ~/.bashrc
```

如果你使用 PowerShell，可以运行：

```powershell
irm https://res-hd.hc-cdn.cn/sharedata/hdspace/devbridge/install .ps1 | iex
安装后执行
$env:Path = [System.Environment]::GetEnvironmentVariable('Path','User')
```

安装脚本使用固定目录，不提供版本、安装目录、下载源或静默安装选项。

| 内容           | 默认位置              |
| -------------- | --------------------- |
| CLI 可执行文件 | `~/.huawei/bin`       |
| CLI 配置与状态 | `~/.huawei/devbridge` |

## 配置 PATH

如果终端无法直接找到 `devbridge`，在当前会话执行：

```bash
export PATH="$HOME/.huawei/bin:$PATH"
```

如果你使用 PowerShell，则执行：

```powershell
$env:Path = "$HOME/.huawei/bin;$env:Path"
```

若要对后续终端持续生效，把相应命令加入对应 shell 的启动文件；Bash/Zsh 可写入 `~/.bashrc` 或 `~/.zshrc`，PowerShell 可写入 `$PROFILE`，然后重新打开终端。

## 验证安装

```bash
devbridge version
devbridge --help
```

第一条命令显示当前 CLI 版本；第二条命令显示当前版本支持的命令。

## 安装后的目录

不要把 `~/.huawei/devbridge` 提交到代码仓库，也不要在不同用户之间复制该目录。设备迁移时，应在新设备重新登录。

有关目录内容和默认隧道状态，请参阅[本地配置与目录](../reference/configuration.md)。

## 下一步

[登录 DevBridge](./authentication.md)
