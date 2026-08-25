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

```bash
curl -fsSL https://res-hd.hc-cdn.cn/sharedata/hdspace/devbridge/install.sh | bash
```

### PowerShell

```powershell
irm https://res-hd.hc-cdn.cn/sharedata/hdspace/devbridge/install.ps1 | iex
```

标准安装会安装当前发布版本并使用默认目录，不需要指定版本或下载源。

| 内容           | 默认位置              |
| -------------- | --------------------- |
| CLI 可执行文件 | `~/.huawei/bin`       |
| CLI 配置与状态 | `~/.huawei/devbridge` |

## 安装脚本参数

标准安装无需任何参数。如需自定义版本、下载源或安装目录，通过参数传入。

### Bash

| 参数              | 说明                                      |
| ----------------- | ----------------------------------------- |
| `-v, --version`   | 指定安装版本，默认使用脚本内置版本。      |
| `-u, --url`       | 制品下载基础 URL，默认使用内置 CDN 地址。 |
| `-p, --prefix`    | 安装目录，默认 `~/.huawei/bin`。          |
| `-s, --silent`    | 静默模式，跳过交互提示。                  |
| `--skip-checksum` | 跳过 SHA256 校验。                        |
| `-h, --help`      | 显示帮助。                                |

```bash
curl -fsSL <url>/install.sh | bash -s -- -v 1.0.0
curl -fsSL <url>/install.sh | bash -s -- -p /custom/path -s
```

### PowerShell

| 参数            | 说明                                      |
| --------------- | ----------------------------------------- |
| `-Version`      | 指定安装版本，默认使用脚本内置版本。      |
| `-Url`          | 制品下载基础 URL，默认使用内置 CDN 地址。 |
| `-Prefix`       | 安装目录，默认 `~/.huawei/bin`。          |
| `-Silent`       | 静默模式，跳过交互提示。                  |
| `-SkipChecksum` | 跳过 SHA256 校验。                        |

PowerShell 管道方式不支持传参，需先保存脚本再执行：

```powershell
irm <url>/install.ps1 -OutFile install.ps1
.\install.ps1 -Version 1.0.0
```

### 环境变量

| 变量                    | 等价参数     | 说明                               |
| ----------------------- | ------------ | ---------------------------------- |
| `APP_VERSION`           | `--version`  | 指定安装版本。                     |
| `ARTIFACT_URL_FROM_ENV` | `--url`      | 制品下载基础 URL。                 |

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
