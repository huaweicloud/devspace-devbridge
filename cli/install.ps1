# DevBridge CLI PowerShell install script
function Install-DevBridge {
    [CmdletBinding()]
    param(
        [string]$Version = "",
        [string]$Url = "",
        [string]$Prefix = "",
        [switch]$Silent,
        [switch]$SkipChecksum,
        [switch]$Help
    )

    $ErrorActionPreference = "Continue"
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

    # ---------------------------------------------------------------------------
    # 全局配置
    # ---------------------------------------------------------------------------
    $Script:APP_NAME = "devbridge"
    $Script:APP_DISPLAY_NAME = "DevBridge"
    $Script:INSTALL_DIR = if ($Prefix) { $Prefix } else { Join-Path $HOME ".huawei\bin" }
    $Script:CONFIG_DIR = Join-Path $HOME ".huawei\devbridge"
    $Script:DEFAULT_ARTIFACT_URL = "https://tools-artifact.developer.huaweicloud.com/sharedata/devbridge"
    if ($Script:DEFAULT_ARTIFACT_URL -match '^__.*__$') {
        $Script:DEFAULT_ARTIFACT_URL = "https://obs-test-hd-space-cdn-sharedata-north7.obs.cn-north-7.ulanqab.huawei.com/space/devbridge"
    }
    $Script:DEFAULT_VERSION = "0.1.13-release"
    if ($Script:DEFAULT_VERSION -match '^__.*__$') {
        $Script:DEFAULT_VERSION = ""
    }
    $Script:ARTIFACT_URL = if ($Url) { $Url } elseif ($env:ARTIFACT_URL_FROM_ENV) { $env:ARTIFACT_URL_FROM_ENV } else { "" }
    $Script:VERSION = if ($Version) { $Version } elseif ($env:APP_VERSION) { $env:APP_VERSION } else { $Script:DEFAULT_VERSION }
    $Script:SKIP_CHECKSUM = $SkipChecksum
    $Script:SILENT_MODE = $Silent
    $Script:PLATFORM = ""
    $Script:GOOS = ""
    $Script:ARCH = ""
    $Script:EXE_SUFFIX = ""

    # ---------------------------------------------------------------------------
    # 日志函数
    # ---------------------------------------------------------------------------
    function Write-Info { param([string]$Msg) Write-Host "[INFO]  $Msg" -ForegroundColor Green }
    function Write-Warn { param([string]$Msg) Write-Host "[WARN]  $Msg" -ForegroundColor Yellow }
    function Write-Step { param([string]$Msg) Write-Host $Msg -ForegroundColor Gray }
    function Write-ErrorAndExit {
        param([string]$Msg)
        Write-Host "[ERROR] $Msg" -ForegroundColor Red
        throw $Msg
    }

    # Get-FileSha256 - 计算文件 SHA256 哈希（小写）
    function Get-FileSha256 {
        param([string]$Path)
        return (Get-FileHash -Path $Path -Algorithm SHA256).Hash.ToLower()
    }

    # Get-WebContent - 获取远程 URL 文本内容
    function Get-WebContent {
        param([string]$Url, [switch]$BestEffort)
        try {
            $resp = Invoke-WebRequest -Uri $Url -UseBasicParsing -ErrorAction Stop
            if ($resp.Content -is [byte[]]) {
                return [System.Text.Encoding]::UTF8.GetString($resp.Content)
            }
            return $resp.Content
        } catch {
            if (-not $BestEffort) {
                Write-ErrorAndExit "Failed to fetch from ${Url}: $($_.Exception.Message)"
            }
            return ""
        }
    }

    # ---------------------------------------------------------------------------
    # 欢迎横幅
    # ---------------------------------------------------------------------------
    function Show-Welcome {
        Write-Host ""
        Write-Host "+==================================================+" -ForegroundColor Blue
        Write-Host "|         DevBridge Installation Wizard           |" -ForegroundColor Blue
        Write-Host "+==================================================+" -ForegroundColor Blue
        Write-Host "  Welcome to $Script:APP_DISPLAY_NAME One-Click Installation Script" -ForegroundColor DarkYellow
        Write-Host ""
    }

    # ---------------------------------------------------------------------------
    # usage
    # ---------------------------------------------------------------------------
    function Show-Usage {
        @"
$Script:APP_DISPLAY_NAME Installer (PowerShell)

Usage: install.ps1 [options]

Options:
    -Version VERSION     Version to install (default: $($Script:DEFAULT_VERSION))
    -Url URL             Base URL of artifact repository (default: $($Script:DEFAULT_ARTIFACT_URL))
    -Prefix DIR          Installation prefix (default: $($Script:INSTALL_DIR))
    -Silent              Silent mode, skip interactive prompts
    -SkipChecksum        Skip SHA256 checksum verification
    -Help                Show this help message

Examples:
    # GitHub one-click:
    irm https://github.com/huaweicloud/devspace-devbridge/releases/latest/download/install.ps1 | iex
    # GitCode one-click:
    irm https://gitcode.com/CloudDeveloperDepartment/devbrige/releases/download/latest/install.ps1 | iex
    # OBS/CDN one-click:
    irm https://tools-artifact.developer.huaweicloud.com/sharedata/devbridge/install.ps1 | iex
    # Explicit version / mirror:
    .\install.ps1 -Version 1.0.0
    .\install.ps1 -Url https://gitcode.com/CloudDeveloperDepartment/devbrige/releases/download/<version> -Version 1.0.0

Note:
    Without -Url, the script downloads from the baked-in DEFAULT_ARTIFACT_URL
    (GitHub / GitCode / OBS, depending on where the script was obtained).

Environment Variables:
    ARTIFACT_URL_FROM_ENV  Same as -Url (skips mirror probing)
    APP_VERSION            Same as -Version
"@
        return
    }

    # ---------------------------------------------------------------------------
    # Detect-Platform - 检测操作系统和 CPU 架构
    # ---------------------------------------------------------------------------
    function Detect-Platform {
        $os = ""
        $arch = ""

        if ($IsLinux -or ($env:OS -eq "Linux")) {
            $os = "Linux"
        } elseif ($IsMacOS -or ($env:OS -eq "Darwin")) {
            $os = "Darwin"
        } elseif ($IsWindows -or ($env:OS -eq "Windows_NT")) {
            $os = "Windows"
        } else {
            Write-ErrorAndExit "Unsupported OS: $([System.Environment]::OSVersion.Platform)"
        }

        $cpu = [System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture
        if ($cpu -eq [System.Runtime.InteropServices.Architecture]::X64) {
            $arch = "amd64"
        } elseif ($cpu -eq [System.Runtime.InteropServices.Architecture]::Arm64) {
            $arch = "arm64"
        } elseif ($env:PROCESSOR_ARCHITECTURE -eq "AMD64") {
            $arch = "amd64"
        } elseif ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
            $arch = "arm64"
        } else {
            Write-ErrorAndExit "Unsupported architecture: $cpu"
        }

        $Script:PLATFORM = "${os}_${arch}"
        $Script:GOOS = $os.ToLower()
        $Script:ARCH = $arch
        $Script:EXE_SUFFIX = if ($Script:GOOS -eq "windows") { ".exe" } else { "" }
    }

    # ---------------------------------------------------------------------------
    # Check-Platform - 校验平台组合是否支持
    # ---------------------------------------------------------------------------
    function Check-Platform {
        $supported = @("Linux_amd64", "Linux_arm64", "Darwin_amd64", "Darwin_arm64", "Windows_amd64", "Windows_arm64")

        if ($supported -notcontains $Script:PLATFORM) {
            Write-Host "Installation failed." -ForegroundColor Red
            Write-Host "Cause: Unsupported platform: $($Script:PLATFORM)" -ForegroundColor Yellow
            Write-Host "Supported: $($supported -join ' ')" -ForegroundColor Gray
            throw "Unsupported platform: $($Script:PLATFORM)"
        }

        Write-Step "Detected platform: $($Script:PLATFORM)"
    }

    # ---------------------------------------------------------------------------
    # Get-BinaryName - 获取远程产物 tar.gz 包名
    # ---------------------------------------------------------------------------
    function Get-BinaryName {
        return "$($Script:APP_NAME)_$($Script:GOOS)_$($Script:ARCH)_$($Script:VERSION)$($Script:EXE_SUFFIX).tar.gz"
    }

    # ---------------------------------------------------------------------------
    # Get-BinaryNameInside - 获取 tar 包内的二进制文件名（不含 .tar.gz 后缀）
    # ---------------------------------------------------------------------------
    function Get-BinaryNameInside {
        return "$($Script:APP_NAME)_$($Script:GOOS)_$($Script:ARCH)_$($Script:VERSION)$($Script:EXE_SUFFIX)"
    }

    # ---------------------------------------------------------------------------
    # Get-BinaryPath - 获取已安装二进制文件的路径
    # ---------------------------------------------------------------------------
    function Get-BinaryPath {
        return Join-Path $Script:INSTALL_DIR "$($Script:APP_NAME)$($Script:EXE_SUFFIX)"
    }

    # ---------------------------------------------------------------------------
    # Invoke-HttpGet - 统一下载函数
    # ---------------------------------------------------------------------------
    function Invoke-HttpGet {
        param([string]$Url, [string]$Output, [switch]$BestEffort)

        try {
            Invoke-WebRequest -Uri $Url -OutFile $Output -UseBasicParsing -ErrorAction Stop
        } catch {
            if (-not $BestEffort) {
                Write-ErrorAndExit "Failed to download from ${Url}: $_"
            }
        }
    }

    # ---------------------------------------------------------------------------
    # Check-ExistingInstall - 检查是否已安装
    # ---------------------------------------------------------------------------
    function Check-ExistingInstall {
        $binaryPath = Get-BinaryPath

        if (-not (Test-Path $binaryPath)) {
            return $false
        }

        if ($Script:SILENT_MODE) {
            if (Test-RemoteHash $binaryPath) {
                Write-Info "$Script:APP_DISPLAY_NAME is already up to date."
                return $true
            }
            return $false
        }

        Write-Host "The $Script:APP_DISPLAY_NAME has been installed." -ForegroundColor DarkYellow
        Write-Host "  $binaryPath"
        $response = Read-Host "Do you want to overwrite it? [y/N]"
        if ($response -match '^[yY]$') {
            return $false
        }
        return $true
    }

    # ---------------------------------------------------------------------------
    # Test-RemoteHash - 比较本地已安装版本与目标版本
    #
    # GoReleaser 产物: checksums.txt 在 Release 根目录，tar.gz 内只有二进制。
    # 改为直接比较已安装二进制的 version 输出与目标版本号。
    # ---------------------------------------------------------------------------
    function Test-RemoteHash {
        param([string]$BinaryPath)

        $installedVersion = & $BinaryPath version 2>$null
        if (-not $installedVersion) { $installedVersion = "unknown" }

        Write-Step "Installed version: $installedVersion"
        Write-Step "Target version:    $($Script:VERSION)"

        return $installedVersion -eq $Script:VERSION
    }

    # ---------------------------------------------------------------------------
    # Prompt-CleanOldData - 提示清理旧配置数据
    # ---------------------------------------------------------------------------
    function Prompt-CleanOldData {
        if ($Script:SILENT_MODE) { return }
        if (-not (Test-Path $Script:CONFIG_DIR)) { return }

        Write-Host "The $Script:APP_DISPLAY_NAME has old config data." -ForegroundColor DarkYellow
        Write-Host "  $($Script:CONFIG_DIR)"
        $response = Read-Host "Do you want to clean old config data? [y/N]"
        if ($response -match '^[yY]$') {
            try {
                Remove-Item -Path $Script:CONFIG_DIR -Recurse -Force -ErrorAction Stop
                Write-Host "✓ Clean old config data completed." -ForegroundColor Green
            } catch {
                Write-Host "✗ Clean old config data failed." -ForegroundColor Red
            }
        }
    }

    # ---------------------------------------------------------------------------
    # Resolve-TarCommand - 查找可用的 tar 命令
    #
    # Windows 10 1803+ 内置 tar.exe 在 C:\Windows\System32，但某些企业管控环境
    # 会把 System32 从 PATH 中移除，导致裸 tar 调用失败。
    # 本函数按优先级查找：PATH -> System32 -> Git 安装目录。
    # ---------------------------------------------------------------------------
    function Resolve-TarCommand {
        # 1. tar 已在 PATH 中
        $cmd = Get-Command tar -ErrorAction SilentlyContinue
        if ($cmd) { return $cmd.Source }

        # 2. Windows 内置 tar.exe（System32 / SysWOW64）
        $sysPaths = @(
            Join-Path $env:SystemRoot "System32\tar.exe"
            Join-Path $env:SystemRoot "SysWOW64\tar.exe"
        )
        foreach ($p in $sysPaths) {
            if (Test-Path $p) { return $p }
        }

        # 3. Git for Windows 自带的 tar.exe
        $gitPaths = @(
            "C:\Program Files\Git\usr\bin\tar.exe",
            "C:\Program Files (x86)\Git\usr\bin\tar.exe"
        )
        foreach ($p in $gitPaths) {
            if (Test-Path $p) { return $p }
        }

        return $null
    }


    # ---------------------------------------------------------------------------
    # Download-Binary - 从远程下载 tar.gz 包并解压
    #
    # 下载源：显式 -Url 优先，否则用 DEFAULT_ARTIFACT_URL（各渠道烤制时写入自己的地址）
    # 返回解压后的二进制文件路径（checksums.txt 也在同目录下，供 Verify-Checksum 使用）
    # ---------------------------------------------------------------------------
    function Download-Binary {
        param([string]$Url, [string]$OutputDir)

        $tarballName = Get-BinaryName
        $localTarball = Join-Path $OutputDir $tarballName

        $mirror = if ($Url) { $Url } else { $Script:DEFAULT_ARTIFACT_URL }
        $remoteUrl = "${mirror}/${tarballName}"

        Write-Step "Downloading from ${remoteUrl} ..."
        Invoke-HttpGet -Url $remoteUrl -Output $localTarball

        # 下载 checksums.txt（GoReleaser 生成的校验汇总文件，besteffort: 旧 Release 可能没有）
        $checksumsUrl = "${mirror}/checksums.txt"
        Write-Step "Downloading ${checksumsUrl} ..."
        $checksumsPath = Join-Path $OutputDir "checksums.txt"
        Invoke-HttpGet -Url $checksumsUrl -Output $checksumsPath -BestEffort

        # 解压 tar.gz（Windows 10+ 内置 tar）
        Write-Step "Extracting ${tarballName} ..."
        $tarCmd = Resolve-TarCommand
        if (-not $tarCmd) {
            Write-ErrorAndExit @"
tar command not found. Cannot extract ${tarballName}.
Possible fixes:
  1. Ensure C:\Windows\System32 is in your PATH (Windows 10 1803+ has built-in tar.exe)
  2. Install Git for Windows: winget install Git.Git
  3. Re-run this installer after adding tar to PATH
"@
        }

        & $tarCmd xzf "$localTarball" -C "$OutputDir"

        # 返回解压后的二进制路径
        $binaryNameInside = Get-BinaryNameInside
        return Join-Path $OutputDir $binaryNameInside
    }

    # ---------------------------------------------------------------------------
    # Verify-Checksum - 校验二进制文件的 SHA256 哈希值
    # ---------------------------------------------------------------------------
    function Verify-Checksum {
        param([string]$BinaryFile)

        # GoReleaser 产物: checksums.txt 在 Release 根目录，校验对象是 tar.gz 而非裸二进制
        $tarballFile = "${BinaryFile}.tar.gz"
        $checksumsFile = Join-Path (Split-Path $BinaryFile) "checksums.txt"

        if ($Script:SKIP_CHECKSUM) {
            Write-Warn "Skipping checksum verification"
            return
        }

        if (-not (Test-Path $checksumsFile)) {
            Write-Warn "checksums.txt not found, skipping checksum verification"
            return
        }

        if (-not (Test-Path $tarballFile)) {
            Write-Warn "Archive $tarballFile not found, skipping checksum verification"
            return
        }

        $localHash = Get-FileSha256 -Path $tarballFile

        # checksums.txt 格式: <hash>  <filename>，查找 tar.gz 对应的 hash
        $tarballBasename = Split-Path $tarballFile -Leaf
        $remoteHash = ""
        foreach ($line in (Get-Content $checksumsFile)) {
            if ($line -match "\s+$($tarballBasename)$") {
                $remoteHash = ($line -split "\s+")[0]
                break
            }
        }
        if ([string]::IsNullOrWhiteSpace($remoteHash)) {
            Write-Warn "Remote hash is empty, skipping checksum verification"
            return
        }

        if ($localHash -eq $remoteHash) {
            return
        }

        Write-ErrorAndExit "Checksum verification failed for ${tarballFile}"
    }

    # ---------------------------------------------------------------------------
    # Install-Binary - 安装二进制文件到 INSTALL_DIR
    # ---------------------------------------------------------------------------
    function Install-Binary {
        param([string]$BinaryFile)

        if (-not (Test-Path $Script:INSTALL_DIR)) {
            New-Item -ItemType Directory -Path $Script:INSTALL_DIR -Force | Out-Null
        }

        $installPath = Get-BinaryPath
        Copy-Item -Path $BinaryFile -Destination $installPath -Force

        Write-Host "✓ Binary installed to: $installPath" -ForegroundColor Green
        Write-Host "✓ Installation $Script:APP_DISPLAY_NAME completed." -ForegroundColor Green
    }

    # ---------------------------------------------------------------------------
    # Add-ToPath - 将安装目录添加到 PATH
    # ---------------------------------------------------------------------------
    function Add-ToPath {
        $binPath = $Script:INSTALL_DIR

        $currentPath = [System.Environment]::GetEnvironmentVariable("Path", "User")
        if ($currentPath -split ';' -contains $binPath) {
            Write-Host "✓ PATH already contains $binPath" -ForegroundColor Green
            return
        }

        [System.Environment]::SetEnvironmentVariable("Path", "${binPath};${currentPath}", "User")
        Write-Host "✓ Successfully added $binPath to user PATH" -ForegroundColor Green
    }

    # ---------------------------------------------------------------------------
    # Verify-Installation - 验证安装结果
    # ---------------------------------------------------------------------------
    function Verify-Installation {
        $binaryPath = Get-BinaryPath

        if (Test-Path $binaryPath) {
            $installedVersion = & $binaryPath version 2>$null
            if (-not $installedVersion) { $installedVersion = "unknown" }
            Write-Host "✓ Installation successful! $Script:APP_DISPLAY_NAME version: $installedVersion" -ForegroundColor Green
        } else {
            Write-Warn "$Script:APP_DISPLAY_NAME binary not found at $binaryPath"
        }
    }

    # ---------------------------------------------------------------------------
    # Show-PostInstallNotice - 安装后提示
    # ---------------------------------------------------------------------------
    function Show-PostInstallNotice {
        Write-Host ""
        Write-Host "+==================================================+" -ForegroundColor Blue
        Write-Host "|                      NOTICE                      |" -ForegroundColor Blue
        Write-Host "+==================================================+" -ForegroundColor Blue
        Write-Host "To apply changes immediately:"
        Write-Host "  Restart your terminal" -ForegroundColor DarkYellow
        Write-Host "  Or run: `$env:Path = [System.Environment]::GetEnvironmentVariable('Path','User')" -ForegroundColor DarkYellow
    }

    # ---------------------------------------------------------------------------
    # main - 主入口
    # ---------------------------------------------------------------------------
    function Main {
        if ($Help) { Show-Usage }

        Show-Welcome
        Detect-Platform
        Check-Platform

        if ([string]::IsNullOrEmpty($Script:VERSION)) {
            Write-ErrorAndExit "No version specified. Use -Version <version>, or run the CI-built install script which has a built-in version."
        }

        Write-Step "Artifact URL: $($Script:ARTIFACT_URL)"
        Write-Step "Version: $($Script:VERSION)"

        if (Check-ExistingInstall) {
            Show-PostInstallNotice
            return
        }

        Prompt-CleanOldData

        $downloadDir = New-Item -ItemType Directory -Path (Join-Path $env:TEMP "devbridge-install-$(Get-Random)") -Force

        Write-Step "Downloading from remote repository: $($Script:ARTIFACT_URL)"

        $binaryFile = Download-Binary -Url $Script:ARTIFACT_URL -OutputDir $downloadDir.FullName
        if (-not (Test-Path $binaryFile)) {
            Write-ErrorAndExit "Failed to download binary"
        }

        Verify-Checksum -BinaryFile $binaryFile
        Install-Binary -BinaryFile $binaryFile
        Add-ToPath
        Verify-Installation
        Show-PostInstallNotice
    }

    Main
}

Install-DevBridge @args
