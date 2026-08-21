#!/usr/bin/env bash
# =============================================================================
# install_bash.sh - DevBridge CLI 安装脚本 (Bash)
#
# 支持两种安装模式：
#   1. 本地模式：从流水线产物目录安装 (-d)
#   2. 远程模式：从制品仓库 URL 下载安装 (-u)
#
# 远程产物为解压后的二进制文件，命名规范：
#   Linux/Darwin: devbridge_{OS}_{Arch}_{Version}
#   Windows:      devbridge_{OS}_{Arch}_{Version}.exe
#
# 支持平台：Linux/Darwin/Windows x amd64/arm64
# =============================================================================

if [ -z "$BASH_VERSION" ]; then
    echo "Error: This script must be run with bash" >&2
    exit 1
fi

if [ "${BASH_VERSINFO[0]}" -lt 3 ]; then
    echo "Error: This script requires bash 3.0 or higher" >&2
    exit 1
fi

set -uo pipefail

on_error() {
    local exit_code=$?
    local line_no=$1
    echo -e "\033[0;31m[ERROR]\033[0m Script exited unexpectedly at line ${line_no} (exit code: ${exit_code})" >&2
    exit "${exit_code}"
}
trap 'on_error ${LINENO}' ERR

# ---------------------------------------------------------------------------
# 终端输出颜色
# ---------------------------------------------------------------------------
MUTED='\033[0;2m'
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
ORANGE='\033[0;33m'
BLUE='\033[1;34m'
NC='\033[0m'

# ---------------------------------------------------------------------------
# 全局配置
# ---------------------------------------------------------------------------
APP_NAME=devbridge
APP_DISPLAY_NAME="DevBridge"
INSTALL_DIR="$HOME/.huawei/bin"
CONFIG_DIR="$HOME/.huawei/devbridge"
DEFAULT_ARTIFACT_URL="https://res-hd.hc-cdn.cn/sharedata/hdspace/devbridge"
if [[ "${DEFAULT_ARTIFACT_URL}" == __*__ ]]; then
    DEFAULT_ARTIFACT_URL="https://obs-test-hd-space-cdn-sharedata-north7.obs.cn-north-7.ulanqab.huawei.com/space/devbridge"
fi
DEFAULT_VERSION="0.1.13-release"
if [[ "${DEFAULT_VERSION}" == __*__ ]]; then
    DEFAULT_VERSION=""
fi
ARTIFACT_DIR=""
ARTIFACT_URL=""
VERSION=""
SKIP_CHECKSUM=false
SILENT_MODE=false
CURL_CMD="curl"
_cleanup_tmp_dirs=()

# ---------------------------------------------------------------------------
# 日志函数
# ---------------------------------------------------------------------------
info()    { echo -e "${GREEN}[INFO]${NC}  $*" >&2; }
warn()    { echo -e "${YELLOW}[WARN]${NC}  $*" >&2; }
error()   { echo -e "${RED}[ERROR]${NC} $*" >&2; exit 1; }
step()    { echo -e "${MUTED}$*${NC}" >&2; }
verbose()  { echo -e "${BLUE}[STEP]${NC} $*" >&2; }

# compute_sha256 - 计算文件 SHA256 哈希，无可用工具时返回空
compute_sha256() {
    if command -v sha256sum &>/dev/null; then
        sha256sum "$1" | awk '{print $1}'
    elif command -v shasum &>/dev/null; then
        shasum -a 256 "$1" | awk '{print $1}'
    fi
}

# get_rc_file - 获取当前 shell 的 rc 文件路径
get_rc_file() {
    if [[ -n "${ZSH_VERSION:-}" ]]; then
        echo "$HOME/.zshrc"
    elif [[ -n "${BASH_VERSION:-}" ]]; then
        echo "$HOME/.bashrc"
    else
        echo "$HOME/.profile"
    fi
}

# ---------------------------------------------------------------------------
# 欢迎横幅
# ---------------------------------------------------------------------------
welcome() {
    echo
    echo -e "${BLUE}+==================================================+${NC}"
    echo -e "${BLUE}|         DevBridge Installation Wizard           |${NC}"
    echo -e "${BLUE}+==================================================+${NC}"
    echo -e "${ORANGE}  Welcome to ${APP_DISPLAY_NAME} One-Click Installation Script${NC}"
    echo
}

# ---------------------------------------------------------------------------
# usage
# ---------------------------------------------------------------------------
usage() {
    cat <<EOF
${APP_DISPLAY_NAME} Installer

Usage: install_bash.sh [options]

Options:
    -v, --version VERSION       Version/timestamp to install (required for remote mode)
    -u, --url URL               Base URL of artifact repository
    -d, --dir DIR               Local artifact directory (pipeline workspace)
    -p, --prefix DIR            Installation prefix (default: ${INSTALL_DIR})
    -s, --silent                Silent mode, skip interactive prompts
    --skip-checksum             Skip SHA256 checksum verification
    --ssl-no-revoke             (TLS) (Schannel) Disable certificate revocation checks
    -h, --help                  Show this help message

Examples:
    curl -fsSL https://your-domain.com/install.sh | bash -s -- -v 20260716144213
    bash install_bash.sh -d /path/to/artifacts -v 1.0.0
    bash install_bash.sh -u https://artifact.example.com/devbridge -v 1.0.0
    bash install_bash.sh -d ./bin -s

Environment Variables:
    ARTIFACT_DIR_FROM_ENV  Same as --dir
    ARTIFACT_URL_FROM_ENV  Same as --url
    APP_VERSION            Same as --version
EOF
    exit 0
}

# ---------------------------------------------------------------------------
# parse_args
# ---------------------------------------------------------------------------
parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -v|--version)       VERSION="$2"; shift 2 ;;
            -u|--url)           ARTIFACT_URL="$2"; shift 2 ;;
            -d|--dir)           ARTIFACT_DIR="$2"; shift 2 ;;
            -p|--prefix)        INSTALL_DIR="$2"; shift 2 ;;
            -s|--silent)        SILENT_MODE=true; shift ;;
            --skip-checksum)    SKIP_CHECKSUM=true; shift ;;
            --ssl-no-revoke)    CURL_CMD="curl --ssl-no-revoke"; shift ;;
            -h|--help)          usage ;;
            *)                  error "Unknown option: $1" ;;
        esac
    done

    ARTIFACT_DIR="${ARTIFACT_DIR:-${ARTIFACT_DIR_FROM_ENV:-}}"
    ARTIFACT_URL="${ARTIFACT_URL:-${ARTIFACT_URL_FROM_ENV:-}}"
    VERSION="${VERSION:-${APP_VERSION:-}}"
}

# ---------------------------------------------------------------------------
# detect_platform - 检测操作系统和 CPU 架构
# ---------------------------------------------------------------------------
detect_platform() {
    local os arch

    case "$(uname -s)" in
        Linux)  os=Linux ;;
        Darwin) os=Darwin ;;
        MINGW*|MSYS*|CYGWIN*) os=Windows ;;
        *)      error "Unsupported OS: $(uname -s)" ;;
    esac

    case "$(uname -m)" in
        x86_64|amd64)  arch=amd64 ;;
        aarch64|arm64) arch=arm64 ;;
        *)             error "Unsupported architecture: $(uname -m)" ;;
    esac

    PLATFORM="${os}_${arch}"
    GOOS=$(echo "${os}" | tr '[:upper:]' '[:lower:]')
    ARCH="${arch}"
    EXE_SUFFIX=""
    [[ "${GOOS}" == "windows" ]] && EXE_SUFFIX=".exe"
}

# ---------------------------------------------------------------------------
# check_platform - 校验平台组合是否支持
# ---------------------------------------------------------------------------
check_platform() {
    local supported="Linux_amd64 Linux_arm64 Darwin_amd64 Darwin_arm64 Windows_amd64 Windows_arm64"

    if ! echo "${supported}" | grep -qw "${PLATFORM}"; then
        echo -e "${RED}Installation failed.${NC}"
        echo -e "${YELLOW}Cause: Unsupported platform: ${PLATFORM}${NC}"
        echo -e "${MUTED}Supported: ${supported}${NC}"
        exit 1
    fi

    verbose "Detected platform: ${PLATFORM}"
}

# ---------------------------------------------------------------------------
# get_binary_name - 获取远程产物二进制文件名
# ---------------------------------------------------------------------------
get_binary_name() {
    echo "${APP_NAME}_${GOOS}_${ARCH}_${VERSION}${EXE_SUFFIX}"
}

# ---------------------------------------------------------------------------
# get_binary_path - 获取已安装二进制文件的路径
# ---------------------------------------------------------------------------
get_binary_path() {
    echo "${INSTALL_DIR}/${APP_NAME}${EXE_SUFFIX}"
}

# ---------------------------------------------------------------------------
# http_get - 统一下载函数（curl 优先，wget 回退）
# ---------------------------------------------------------------------------
http_get() {
    local url="$1" output="$2" besteffort="${3:-}"

    if command -v curl &>/dev/null; then
        if [[ "${besteffort}" == "besteffort" ]]; then
            $CURL_CMD -fsSL -o "${output}" "${url}" 2>/dev/null || true
        else
            if ! $CURL_CMD -fsSL -o "${output}" "${url}" 2>&1; then
                error "Failed to download: ${url}"
            fi
        fi
    elif command -v wget &>/dev/null; then
        if [[ "${besteffort}" == "besteffort" ]]; then
            wget -q -O "${output}" "${url}" 2>/dev/null || true
        else
            if ! wget -q -O "${output}" "${url}"; then
                error "Failed to download: ${url}"
            fi
        fi
    else
        error "curl or wget is required for downloading"
    fi
}

# ---------------------------------------------------------------------------
# check_existing_install - 检查是否已安装
# ---------------------------------------------------------------------------
check_existing_install() {
    local binary_path
    binary_path=$(get_binary_path)

    if [[ ! -f "${binary_path}" ]]; then
        return 1
    fi

    if [[ "${SILENT_MODE}" == true ]]; then
        if [[ -n "${ARTIFACT_URL}" ]] && check_remote_hash "${binary_path}"; then
            info "${APP_DISPLAY_NAME} is already up to date."
            return 0
        fi
        return 1
    fi

    echo -e "${ORANGE}The ${APP_DISPLAY_NAME} has been installed.${NC}"
    echo "  ${binary_path}"
    echo -ne "${ORANGE}Do you want to overwrite it? [y/N]${NC}"
    read -r response </dev/tty
    if [[ ${response} =~ ^[yY]$ ]]; then
        return 1
    fi
    return 0
}

# ---------------------------------------------------------------------------
# check_remote_hash - 比较本地与远程 SHA256 哈希
# ---------------------------------------------------------------------------
check_remote_hash() {
    local binary_path="$1"
    local hash_url="${ARTIFACT_URL}/${APP_NAME}_${GOOS}_${ARCH}_${VERSION}.sha256"

    step "Downloading hash from ${hash_url}"
    local tmp_file
    tmp_file=$(mktemp)
    _cleanup_tmp_dirs+=("$(dirname "${tmp_file}")")

    http_get "${hash_url}" "${tmp_file}" "besteffort"
    if [[ ! -s "${tmp_file}" ]]; then
        step "Failed to download remote hash, skipping comparison"
        return 0
    fi

    local local_hash
    local_hash=$(compute_sha256 "${binary_path}")
    if [[ -z "${local_hash}" ]]; then
        step "No sha256 tool available, skipping comparison"
        return 0
    fi

    local remote_hash
    remote_hash=$(grep "${APP_NAME}" "${tmp_file}" | head -1 | awk '{print $1}')
    [[ -z "${remote_hash}" ]] && { step "Remote hash is empty, skipping comparison"; return 0; }

    step "Hash Local:  ${local_hash}"
    step "Hash Remote: ${remote_hash}"

    [[ "${local_hash}" == "${remote_hash}" ]]
}

# ---------------------------------------------------------------------------
# prompt_clean_old_data - 提示清理旧配置数据
# ---------------------------------------------------------------------------
prompt_clean_old_data() {
    [[ "${SILENT_MODE}" == true ]] && return 0
    [[ ! -d "${CONFIG_DIR}" ]] && return 0

    echo -e "${ORANGE}The ${APP_DISPLAY_NAME} has old config data.${NC}"
    echo "  ${CONFIG_DIR}"
    echo -ne "${ORANGE}Do you want to clean old config data? [y/N]${NC}"
    read -r response </dev/tty
    if [[ ${response} =~ ^[yY]$ ]]; then
        if rm -rf "${CONFIG_DIR}"; then
            echo -e "${GREEN}✓ Clean old config data completed.${NC}"
        else
            echo -e "${RED}✗ Clean old config data failed.${NC}"
        fi
    fi
}

# ---------------------------------------------------------------------------
# find_artifact - 在本地目录中查找最新匹配的二进制产物
# ---------------------------------------------------------------------------
find_artifact() {
    local search_dir="$1"
    local pattern="${APP_NAME}_${GOOS}_${ARCH}_*${EXE_SUFFIX}"

    local found=""
    for f in "${search_dir}"/${pattern}; do
        if [[ -f "$f" ]] && [[ "$f" != *.sha256 ]]; then
            if [[ -z "$found" ]] || [[ "$f" > "$found" ]]; then
                found="$f"
            fi
        fi
    done

    echo "${found}"
}

# ---------------------------------------------------------------------------
# download_binary - 从远程下载二进制文件及 SHA256 校验文件
# ---------------------------------------------------------------------------
download_binary() {
    local url="$1" output_dir="$2"
    local filename
    filename=$(get_binary_name)
    local remote_url="${url}/${filename}"
    local local_file="${output_dir}/${filename}"

    verbose "Downloading ${remote_url} ..."
    http_get "${remote_url}" "${local_file}"

    http_get "${url}/${APP_NAME}_${GOOS}_${ARCH}_${VERSION}.sha256" \
             "${output_dir}/${APP_NAME}_${GOOS}_${ARCH}_${VERSION}.sha256" \
             "besteffort"

    echo "${local_file}"
}

# ---------------------------------------------------------------------------
# verify_checksum - 校验二进制文件的 SHA256 哈希值
# ---------------------------------------------------------------------------
verify_checksum() {
    local binary_file="$1"
    local sha_file="${binary_file}.sha256"

    if [[ "${SKIP_CHECKSUM}" == true ]]; then
        warn "Skipping checksum verification"
        return 0
    fi

    if [[ ! -f "${sha_file}" ]]; then
        warn "SHA256 file not found, skipping checksum verification"
        return 0
    fi

    local local_hash
    local_hash=$(compute_sha256 "${binary_file}")
    if [[ -z "${local_hash}" ]]; then
        warn "No sha256 tool available, skipping checksum verification"
        return 0
    fi

    local remote_hash
    remote_hash=$(awk '{print $1}' "${sha_file}" | head -1)
    [[ -z "${remote_hash}" ]] && { warn "Remote hash is empty, skipping checksum verification"; return 0; }

    if [[ "${local_hash}" == "${remote_hash}" ]]; then
        return 0
    fi

    error "Checksum verification failed for ${binary_file}"
}

# ---------------------------------------------------------------------------
# install_binary - 安装二进制文件到 INSTALL_DIR
# ---------------------------------------------------------------------------
install_binary() {
    local binary_file="$1"

    mkdir -p "${INSTALL_DIR}"

    local install_path
    install_path=$(get_binary_path)
    mv "${binary_file}" "${install_path}"
    chmod 750 "${install_path}"

    echo -e "${GREEN}✓ Binary installed to: ${install_path}${NC}"
    echo -e "${GREEN}✓ Installation ${APP_DISPLAY_NAME} completed.${NC}"
}

# ---------------------------------------------------------------------------
# add_to_path - 将安装目录添加到 PATH
# ---------------------------------------------------------------------------
add_to_path() {
    local bin_path="${INSTALL_DIR}"
    local path_export="export PATH=\"${bin_path}:\$PATH\""
    local rc_file
    rc_file=$(get_rc_file)

    [[ ! -f "${rc_file}" ]] && touch "${rc_file}"

    if grep -Fxq "${path_export}" "${rc_file}" 2>/dev/null; then
        echo -e "${GREEN}✓ ${rc_file} already contains ${bin_path} in PATH${NC}"
        return 0
    fi

    echo "" >> "${rc_file}"
    echo "# Huawei ${APP_DISPLAY_NAME} bin directory to PATH" >> "${rc_file}"
    echo "${path_export}" >> "${rc_file}"

    if echo "${PATH}" | grep -q "${bin_path}" 2>/dev/null; then
        echo -e "${GREEN}✓ PATH already contains ${bin_path}${NC}"
        return 0
    fi

    echo -e "${GREEN}✓ Successfully added ${bin_path} to PATH in ${rc_file}${NC}"
}

# ---------------------------------------------------------------------------
# verify_installation - 验证安装结果
# ---------------------------------------------------------------------------
verify_installation() {
    local binary_path
    binary_path=$(get_binary_path)

    if [[ -f "${binary_path}" ]]; then
        local installed_version
        installed_version=$("${binary_path}" version 2>/dev/null || echo "unknown")
        echo -e "${GREEN}✓ Installation successful! ${APP_DISPLAY_NAME} version: ${installed_version}${NC}"
    else
        warn "${APP_DISPLAY_NAME} binary not found at ${binary_path}"
    fi
}

# ---------------------------------------------------------------------------
# show_post_install_notice - 安装后提示
# ---------------------------------------------------------------------------
show_post_install_notice() {
    local rc_file
    rc_file=$(get_rc_file)

    echo
    echo -e "${BLUE}+==================================================+${NC}"
    echo -e "${BLUE}|                      NOTICE                      |${NC}"
    echo -e "${BLUE}+==================================================+${NC}"
    echo -e "To apply changes immediately:"
    echo -e "  ${ORANGE}source ${rc_file}${NC}"
    echo -e "  Or restart your terminal"
}

# ---------------------------------------------------------------------------
# main - 主入口
# ---------------------------------------------------------------------------
main() {
    parse_args "$@"
    welcome
    detect_platform
    check_platform

    if [[ -z "${ARTIFACT_DIR}" ]] && [[ -z "${ARTIFACT_URL}" ]]; then
        ARTIFACT_URL="${DEFAULT_ARTIFACT_URL}"
        verbose "No artifact source specified, using default: ${ARTIFACT_URL}"
    fi

    if [[ -n "${ARTIFACT_URL}" ]] && [[ -z "${VERSION}" ]]; then
        VERSION="${DEFAULT_VERSION}"
        if [[ -z "${VERSION}" ]]; then
            error "No version specified. Use -v <version> to specify, or run the CI-built install script which has a built-in version."
        fi
    fi

    if check_existing_install; then
        show_post_install_notice
        exit 0
    fi

    prompt_clean_old_data

    local binary_file=""

    if [[ -n "${ARTIFACT_DIR}" ]]; then
        step "Looking for binary in local directory: ${ARTIFACT_DIR}"
        binary_file=$(find_artifact "${ARTIFACT_DIR}")
        [[ -z "${binary_file}" ]] && error "No binary found for ${PLATFORM} in ${ARTIFACT_DIR}"
        step "Found binary: ${binary_file}"

    elif [[ -n "${ARTIFACT_URL}" ]]; then
        local download_dir
        download_dir=$(mktemp -d)
        _cleanup_tmp_dirs+=("${download_dir}")
        trap 'for d in "${_cleanup_tmp_dirs[@]}"; do rm -rf "$d"; done' EXIT

        verbose "Downloading from remote repository: ${ARTIFACT_URL}"

        binary_file=$(download_binary "${ARTIFACT_URL}" "${download_dir}")
        [[ ! -f "${binary_file}" ]] && error "Failed to download binary"
    fi

    verify_checksum "${binary_file}"
    install_binary "${binary_file}"
    add_to_path
    verify_installation
    show_post_install_notice
}

main "$@"
exit 0