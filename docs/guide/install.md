---
title: Install the DevBridge CLI
description: Install and verify the DevBridge CLI using the official installation script.
---

# Install the DevBridge CLI

<p class="lead">The DevBridge CLI provides commands for managing tunnels, ports, Host, Connect, and credentials.</p>

## System requirements

- Bash and `curl`, or PowerShell 5.1 or later
- x86-64 or ARM64 architecture
- Access to a DevBridge installation source
- Write permission for the `~/.huawei` directory

## Install

### Bash

Run the install script from any of the following channels:

::: code-group

```bash [GitHub]
curl -fsSL https://github.com/huaweicloud/devspace-devbridge/releases/latest/download/install.sh | bash
```

```bash [GitCode]
curl -fsSL https://gitcode.com/CloudDeveloperDepartment/devbrige/releases/download/latest/install.sh | bash
```

```bash [OBS]
curl -fsSL https://tools-artifact.developer.huaweicloud.com/sharedata/devbridge/install.sh | bash
```

:::

### PowerShell

Run the install script from any of the following channels:

::: code-group

```powershell [GitHub]
irm https://github.com/huaweicloud/devspace-devbridge/releases/latest/download/install.ps1 | iex
```

```powershell [GitCode]
irm https://gitcode.com/CloudDeveloperDepartment/devbrige/releases/download/latest/install.ps1 | iex
```

```powershell [OBS]
irm https://tools-artifact.developer.huaweicloud.com/sharedata/devbridge/install.ps1 | iex
```

:::

A standard installation installs the current release and uses the default directories; you don't need to specify a version or installation source.

| Item                        | Default location      |
| --------------------------- | --------------------- |
| CLI executable              | `~/.huawei/bin`       |
| CLI configuration and state | `~/.huawei/devbridge` |

## Configure PATH

If your terminal can't find `devbridge` directly, run the following in the current session:

```bash
export PATH="$HOME/.huawei/bin:$PATH"
```

If you use PowerShell, run:

```powershell
$env:Path = "$HOME/.huawei/bin;$env:Path"
```

To make the change persist across future terminals, add the appropriate command to your shell's startup file. For Bash or Zsh, write it to `~/.bashrc` or `~/.zshrc`; for PowerShell, write it to `$PROFILE`. Alternatively, reopen the terminal to pick up the PATH change the installer has already made.

## Verify the installation

```console
devbridge version
devbridge --help
```

The first command shows the current CLI version; the second lists the supported commands.

## Directories after installation

Don't commit `~/.huawei/devbridge` to a code repository, and don't copy this directory between different users. When migrating to a new device, log in again.

For how login credentials are stored and the underlying security principles, see [Login and credentials](./authentication.md).

## Next steps

[Log in to DevBridge](./authentication.md)
