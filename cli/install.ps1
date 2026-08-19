# DevBridge CLI PowerShell install script
function Install-DevBridge {
    [CmdletBinding()]
    param(
        [string]$Version = "",
        [string]$Url = "",
        [string]$Dir = "",
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
    $Script:DEFAULT_ARTIFACT_URL = "https://res-hd.hc-cdn.cn/sharedata/hdspace/devbridge"
    if ($Script:DEFAULT_ARTIFACT_URL -match '^__.*__$') {
        $Script:DEFAULT_ARTIFACT_URL = "https://obs-test-hd-space-cdn-sharedata-north7.obs.cn-north-7.ulanqab.huawei.com/space/devbridge"
    }
    $Script:DEFAULT_VERSION = "0.1.13-release"
    if ($Script:DEFAULT_VERSION -match '^__.*__$') {
        $Script:DEFAULT_VERSION = ""
    }
    $Script:ARTIFACT_DIR = if ($Dir) { $Dir } elseif ($env:ARTIFACT_DIR_FROM_ENV) { $env:ARTIFACT_DIR_FROM_ENV } else { "" }
    $Script:ARTIFACT_URL = if ($Url) { $Url } elseif ($env:ARTIFACT_URL_FROM_ENV) { $env:ARTIFACT_URL_FROM_ENV } else { "" }
    $Script:VERSION = if ($Version) { $Version } elseif ($env:APP_VERSION) { $env:APP_VERSION } else { "" }
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
    -Version VERSION     Version to install (required for remote mode)
    -Url URL             Base URL of artifact repository
    -Dir DIR             Local artifact directory (pipeline workspace)
    -Prefix DIR          Installation prefix (default: $Script:INSTALL_DIR)
    -Silent              Silent mode, skip interactive prompts
    -SkipChecksum        Skip SHA256 checksum verification
    -Help                Show this help message

Examples:
    .\install.ps1 -Url https://artifact.example.com/devbridge
    .\install.ps1 -Url https://artifact.example.com/devbridge -Version 1.0.0
    .\install.ps1 -Dir C:\path\to\artifacts -Version 1.0.0
    .\install.ps1 -Dir .\bin -Silent

Environment Variables:
    ARTIFACT_DIR_FROM_ENV  Same as -Dir
    ARTIFACT_URL_FROM_ENV  Same as -Url
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
    # Get-BinaryName - 获取远程产物二进制文件名
    # ---------------------------------------------------------------------------
    function Get-BinaryName {
        return "$($Script:APP_NAME)_$($Script:PLATFORM)_$($Script:VERSION)$($Script:EXE_SUFFIX)"
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
            if ($Script:ARTIFACT_URL -and (Test-RemoteHash $binaryPath)) {
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
    # Test-RemoteHash - 比较本地与远程 SHA256 哈希
    # ---------------------------------------------------------------------------
    function Test-RemoteHash {
        param([string]$BinaryPath)

        $hashUrl = "$($Script:ARTIFACT_URL)/$($Script:APP_NAME)_$($Script:PLATFORM)_$($Script:VERSION).sha256"

        Write-Step "Downloading hash from $hashUrl"
        $remoteHashes = Get-WebContent -Url $hashUrl -BestEffort
        if ([string]::IsNullOrWhiteSpace($remoteHashes)) {
            Write-Step "Failed to download remote hash, skipping comparison"
            return $true
        }

        $localHash = Get-FileSha256 -Path $BinaryPath

        $remoteHash = ($remoteHashes -split "`n" | Select-Object -First 1).Trim() -split '\s' | Select-Object -First 1
        if ([string]::IsNullOrWhiteSpace($remoteHash)) {
            Write-Step "Remote hash is empty, skipping comparison"
            return $true
        }

        Write-Step "Hash Local:  $localHash"
        Write-Step "Hash Remote: $remoteHash"

        return $localHash -eq $remoteHash
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
    # Find-Artifact - 在本地目录中查找最新匹配的二进制产物
    # ---------------------------------------------------------------------------
    function Find-Artifact {
        param([string]$SearchDir)

        $pattern = "$($Script:APP_NAME)_$($Script:PLATFORM)_*$($Script:EXE_SUFFIX)"

        $found = Get-ChildItem -Path $SearchDir -Filter $pattern -File -ErrorAction SilentlyContinue |
            Sort-Object Name -Descending |
            Select-Object -First 1

        if ($found) {
            return $found.FullName
        }
        return ""
    }

    # ---------------------------------------------------------------------------
    # Download-Binary - 从远程下载二进制文件及 SHA256 校验文件
    # ---------------------------------------------------------------------------
    function Download-Binary {
        param([string]$Url, [string]$OutputDir)

        $filename = Get-BinaryName

        $remoteUrl = "${Url}/${filename}"
        $localFile = Join-Path $OutputDir $filename

        Write-Step "Downloading ${remoteUrl} ..."
        Invoke-HttpGet -Url $remoteUrl -Output $localFile

        Invoke-HttpGet -Url "${Url}/$($Script:APP_NAME)_$($Script:PLATFORM)_$($Script:VERSION).sha256" `
                       -Output (Join-Path $OutputDir "$($Script:APP_NAME)_$($Script:PLATFORM)_$($Script:VERSION).sha256") `
                       -BestEffort

        return $localFile
    }

    # ---------------------------------------------------------------------------
    # Verify-Checksum - 校验二进制文件的 SHA256 哈希值
    # ---------------------------------------------------------------------------
    function Verify-Checksum {
        param([string]$BinaryFile)

        $shaFile = "${BinaryFile}.sha256"

        if ($Script:SKIP_CHECKSUM) {
            Write-Warn "Skipping checksum verification"
            return
        }

        if (-not (Test-Path $shaFile)) {
            Write-Warn "SHA256 file not found, skipping checksum verification"
            return
        }

        $localHash = Get-FileSha256 -Path $BinaryFile

        $shaContent = Get-Content -Path $shaFile -Raw
        $remoteHash = ($shaContent -split "`n" | Select-Object -First 1).Trim() -split '\s' | Select-Object -First 1
        if ([string]::IsNullOrWhiteSpace($remoteHash)) {
            Write-Warn "Remote hash is empty, skipping checksum verification"
            return
        }

        if ($localHash -eq $remoteHash) {
            return
        }

        Write-ErrorAndExit "Checksum verification failed for ${BinaryFile}"
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

        if ([string]::IsNullOrEmpty($Script:ARTIFACT_DIR) -and [string]::IsNullOrEmpty($Script:ARTIFACT_URL)) {
            $Script:ARTIFACT_URL = $Script:DEFAULT_ARTIFACT_URL
            Write-Step "No artifact source specified, using default: $($Script:ARTIFACT_URL)"
        }

        if (-not [string]::IsNullOrEmpty($Script:ARTIFACT_URL) -and [string]::IsNullOrEmpty($Script:VERSION)) {
            $Script:VERSION = $Script:DEFAULT_VERSION
            if ([string]::IsNullOrEmpty($Script:VERSION)) {
                Write-ErrorAndExit "No version specified. Use -Version <version> to specify, or run the CI-built install script which has a built-in version."
            }
        }

        if (Check-ExistingInstall) {
            Show-PostInstallNotice
            return
        }

        Prompt-CleanOldData

        $binaryFile = ""

        if (-not [string]::IsNullOrEmpty($Script:ARTIFACT_DIR)) {
            Write-Step "Looking for binary in local directory: $($Script:ARTIFACT_DIR)"
            $binaryFile = Find-Artifact $Script:ARTIFACT_DIR
            if ([string]::IsNullOrEmpty($binaryFile)) {
                Write-ErrorAndExit "No binary found for $($Script:PLATFORM) in $($Script:ARTIFACT_DIR)"
            }
            Write-Step "Found binary: $binaryFile"

        } elseif (-not [string]::IsNullOrEmpty($Script:ARTIFACT_URL)) {
            $downloadDir = New-Item -ItemType Directory -Path (Join-Path $env:TEMP "devbridge-install-$(Get-Random)") -Force

            Write-Step "Downloading from remote repository: $($Script:ARTIFACT_URL)"

            $binaryFile = Download-Binary -Url $Script:ARTIFACT_URL -OutputDir $downloadDir.FullName
            if (-not (Test-Path $binaryFile)) {
                Write-ErrorAndExit "Failed to download binary"
            }
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
