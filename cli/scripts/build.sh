#!/usr/bin/env bash
# =============================================================================
# scripts/build.sh - DevBridge 统一跨平台构建脚本
#
# 功能:
#   1. 基于 git tag 自动推导版本号（可用 -v 覆盖）
#   2. 矩阵构建 6 个平台（linux/darwin/windows x amd64/arm64）
#   3. 产物命名: devbridge_{OS}_{Arch}_{Version}[.exe]（与 install.sh 对齐）
#   4. 为每个产物生成 .sha256 校验文件
#   5. 可选 RSA-PSS 签名（-s 指定私钥路径）
#
# 用法:
#   ./scripts/build.sh                          # 构建全部平台，版本号从 git tag 推导
#   ./scripts/build.sh -v 1.0.0                 # 指定版本号
#   ./scripts/build.sh -p linux/amd64           # 只构建指定平台
#   ./scripts/build.sh -e prod                  # 使用 prod 环境配置
#   ./scripts/build.sh -o /tmp/out              # 指定输出目录
#   ./scripts/build.sh -s /path/to/private.pem  # 签名产物
#   ./scripts/build.sh --all-platforms          # 显式构建全部 6 平台（默认行为）
# =============================================================================
set -euo pipefail

# ---------------------------------------------------------------------------
# 默认配置
# ---------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
OUTPUT_DIR="${PROJECT_ROOT}/bin"
ENV="dev"
VERSION=""
PLATFORMS=()
SIGN_KEY=""

# 环境配置（与 Makefile/run.sh 保持一致）
declare -A ENV_CONFIG
ENV_CONFIG[SERVER_DOMAIN,dev]="https://devstation-desktop-dev.cn-north-7.myhuaweicloud.com"
ENV_CONFIG[SERVER_DOMAIN,test]="https://devstation-desktop.cn-north-7.myhuaweicloud.com"
ENV_CONFIG[SERVER_DOMAIN,prod]="https://devstation.myhuaweicloud.com"
ENV_CONFIG[LOGIN_URL,dev]="https://devstation.ulanqab.huawei.com"
ENV_CONFIG[LOGIN_URL,test]="https://devstation.ulanqab.huawei.com"
ENV_CONFIG[LOGIN_URL,prod]="https://devstation.connect.huaweicloud.com"
ENV_CONFIG[GATEWAY_ADDR,dev]="100.85.218.138:443"
ENV_CONFIG[GATEWAY_ADDR,test]="100.85.218.138:443"
ENV_CONFIG[GATEWAY_ADDR,prod]="gateway.cn-north-4-bridge.huaweicloud.com:443"
ENV_CONFIG[CLUSTER_DOMAIN,dev]="cn-north-4-bridge.myhuaweicloud.com"
ENV_CONFIG[CLUSTER_DOMAIN,test]="cn-north-4-bridge.myhuaweicloud.com"
ENV_CONFIG[CLUSTER_DOMAIN,prod]="cn-north-4-bridge.myhuaweicloud.com"

ALL_PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/arm64"
)

# ---------------------------------------------------------------------------
# 日志
# ---------------------------------------------------------------------------
log_info()    { echo -e "\033[0;32m[INFO]\033[0m  $*"; }
log_warn()    { echo -e "\033[1;33m[WARN]\033[0m  $*"; }
log_error()   { echo -e "\033[0;31m[ERROR]\033[0m $*" >&2; exit 1; }
log_step()    { echo -e "\033[0;2m$*\033[0m"; }

# ---------------------------------------------------------------------------
# 用法
# ---------------------------------------------------------------------------
usage() {
    cat <<EOF
DevBridge 统一跨平台构建脚本

用法:
  ./scripts/build.sh [选项]

选项:
  -v, --version VERSION    版本号（默认从 git tag 推导: git describe --tags --always）
  -e, --env ENV            环境配置: dev|test|prod（默认: dev）
  -p, --platform PLAT      只构建指定平台（格式: os/arch，如 linux/amd64），可多次指定
  -o, --output DIR         输出目录（默认: bin/）
  -s, --sign-key FILE      RSA 私钥路径，用于签名产物
  --all-platforms          构建全部 6 平台（默认行为，显式声明）
  -h, --help               显示帮助

示例:
  ./scripts/build.sh                              # 全平台构建，版本号从 git tag
  ./scripts/build.sh -v 1.0.0 -e prod             # prod 环境，版本 1.0.0
  ./scripts/build.sh -p linux/amd64 -p darwin/arm64  # 只构建两个平台
EOF
    exit 0
}

# ---------------------------------------------------------------------------
# 参数解析
# ---------------------------------------------------------------------------
parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -v|--version)       VERSION="$2"; shift 2 ;;
            -e|--env)           ENV="$2"; shift 2 ;;
            -p|--platform)      PLATFORMS+=("$2"); shift 2 ;;
            -o|--output)        OUTPUT_DIR="$2"; shift 2 ;;
            -s|--sign-key)      SIGN_KEY="$2"; shift 2 ;;
            --all-platforms)    shift ;;
            -h|--help)          usage ;;
            *)                  log_error "未知选项: $1" ;;
        esac
    done

    # 校验环境
    if [[ -z "${ENV_CONFIG[SERVER_DOMAIN,${ENV}]:-}" ]]; then
        log_error "无效环境: ${ENV}（可选: dev|test|prod）"
    fi

    # 默认全平台
    if [[ ${#PLATFORMS[@]} -eq 0 ]]; then
        PLATFORMS=("${ALL_PLATFORMS[@]}")
    fi
}

# ---------------------------------------------------------------------------
# 版本号推导
# ---------------------------------------------------------------------------
resolve_version() {
    if [[ -z "${VERSION}" ]]; then
        if command -v git &>/dev/null && git -C "${PROJECT_ROOT}" rev-parse --is-inside-work-tree &>/dev/null; then
            VERSION=$(git -C "${PROJECT_ROOT}" describe --tags --always --dirty 2>/dev/null || echo "dev")
        else
            VERSION="dev"
        fi
    fi
    log_info "版本号: ${VERSION}"
}

# ---------------------------------------------------------------------------
# 单平台构建
# ---------------------------------------------------------------------------
build_one() {
    local platform="$1"
    local goos goarch

    IFS='/' read -r goos goarch <<< "${platform}"

    # 产物文件名（与 install.sh 对齐: devbridge_{OS}_{Arch}_{Version}）
    local exe_suffix=""
    [[ "${goos}" == "windows" ]] && exe_suffix=".exe"
    local binary_name="devbridge_${goos}_${goarch}_${VERSION}${exe_suffix}"
    local output_path="${OUTPUT_DIR}/${binary_name}"

    # ldflags 注入
    local server_domain="${ENV_CONFIG[SERVER_DOMAIN,${ENV}]}"
    local login_url="${ENV_CONFIG[LOGIN_URL,${ENV}]}"
    local gateway_addr="${ENV_CONFIG[GATEWAY_ADDR,${ENV}]}"
    local cluster_domain="${ENV_CONFIG[CLUSTER_DOMAIN,${ENV}]}"

    local ldflags="-s -w \
-X huawei.com/devbridge/cmd.version=${VERSION} \
-X huawei.com/devbridge/internal/auth.LoginURL=${login_url} \
-X huawei.com/devbridge/internal/config.DefaultServerDomain=${server_domain} \
-X huawei.com/devbridge/internal/connect.ServerAddr=${gateway_addr} \
-X huawei.com/devbridge/internal/connect.ServerHost=${cluster_domain}"

    log_step "构建 ${goos}/${goarch} -> ${binary_name}"

    CGO_ENABLED=0 GOOS="${goos}" GOARCH="${goarch}" \
        go build -ldflags "${ldflags}" -buildmode=pie -trimpath \
        -o "${output_path}" ./cmd/cli

    if [[ ! -f "${output_path}" ]]; then
        log_error "构建失败: ${output_path}"
    fi

    # 生成 SHA256 校验文件
    generate_checksum "${output_path}"

    log_info "✓ ${binary_name} ($(du -h "${output_path}" | cut -f1))"
}

# ---------------------------------------------------------------------------
# SHA256 校验文件
# ---------------------------------------------------------------------------
generate_checksum() {
    local file="$1"
    local sha_file="${file}.sha256"

    if command -v sha256sum &>/dev/null; then
        local hash
        hash=$(sha256sum "${file}" | awk '{print $1}')
        echo "${hash}" > "${sha_file}"
    elif command -v shasum &>/dev/null; then
        local hash
        hash=$(shasum -a 256 "${file}" | awk '{print $1}')
        echo "${hash}" > "${sha_file}"
    else
        log_warn "无 sha256sum/shasum，跳过校验文件生成: ${file}"
        return
    fi
}

# ---------------------------------------------------------------------------
# RSA-PSS 签名
# ---------------------------------------------------------------------------
sign_binary() {
    local file="$1"

    if [[ -z "${SIGN_KEY}" ]] || [[ ! -f "${SIGN_KEY}" ]]; then
        return
    fi

    if ! command -v openssl &>/dev/null; then
        log_warn "无 openssl，跳过签名: ${file}"
        return
    fi

    local sig_file="${file}.sig"
    log_step "签名 $(basename "${file}")"
    openssl dgst -sha256 -sign "${SIGN_KEY}" -out "${sig_file}" "${file}"
    log_info "✓ $(basename "${sig_file}")"
}

# ---------------------------------------------------------------------------
# 主流程
# ---------------------------------------------------------------------------
main() {
    parse_args "$@"
    resolve_version

    mkdir -p "${OUTPUT_DIR}"

    log_info "环境: ${ENV}"
    log_info "输出目录: ${OUTPUT_DIR}"
    log_info "平台: ${PLATFORMS[*]}"
    echo ""

    for platform in "${PLATFORMS[@]}"; do
        build_one "${platform}"
        local goos goarch exe_suffix=""
        IFS='/' read -r goos goarch <<< "${platform}"
        [[ "${goos}" == "windows" ]] && exe_suffix=".exe"
        sign_binary "${OUTPUT_DIR}/devbridge_${goos}_${goarch}_${VERSION}${exe_suffix}"
    done

    echo ""
    log_info "构建完成。产物列表:"
    ls -la "${OUTPUT_DIR}"/devbridge_*_${VERSION}* 2>/dev/null || true
}

main "$@"
