---
title: 安装 DevBridge CLI
description: 使用官方安装脚本安装并验证 DevBridge CLI。
---

# 安装 DevBridge CLI

<p class="lead">DevBridge CLI 提供隧道、端口、Host、Connect 和凭证管理命令。</p>

## 系统要求

- Bash 和 `curl`，或者 PowerShell 5.1 及以上版本；
- x86-64 或 ARM64 架构；
- 可以访问 DevBridge 安装源；
- 对 `~/.huawei` 目录具有写权限。

## 安装

### Bash

任选一个渠道运行安装脚本：

```bash
# GitHub 渠道
curl -fsSL https://github.com/huaweicloud/devspace-devbridge/releases/latest/download/install.sh | bash

# GitCode 渠道
curl -fsSL https://gitcode.com/CloudDeveloperDepartment/devbrige/releases/download/latest/install.sh | bash

# OBS 渠道
curl -fsSL https://tools-artifact.developer.huaweicloud.com/sharedata/devbridge/install.sh | bash
```

### PowerShell

任选一个渠道运行安装脚本：

```powershell
# GitHub 渠道
irm https://github.com/huaweicloud/devspace-devbridge/releases/latest/download/install.ps1 | iex

# GitCode 渠道
irm https://gitcode.com/CloudDeveloperDepartment/devbrige/releases/download/latest/install.ps1 | iex

# OBS 渠道
irm https://tools-artifact.developer.huaweicloud.com/sharedata/devbridge/install.ps1 | iex
```

标准安装会安装当前发布版本并使用默认目录，不需要指定版本或下载源。

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

若要对后续终端持续生效，把相应命令加入 Shell 启动文件。Bash 或 Zsh 可写入
`~/.bashrc` 或 `~/.zshrc`，PowerShell 可写入 `$PROFILE`。也可以重新打开终端，
使用安装程序已经写入的用户 `PATH`。

## 验证安装

```console
devbridge version
devbridge --help
```

第一条命令显示当前 CLI 版本；第二条命令显示当前版本支持的命令。

## 安装后的目录

不要把 `~/.huawei/devbridge` 提交到代码仓库，也不要在不同用户之间复制该目录。设备迁移时，应在新设备重新登录。

有关目录内容和默认隧道状态，请参阅[本地配置与目录](../reference/configuration.md)。

## 下一步

[登录 DevBridge](./authentication.md)
