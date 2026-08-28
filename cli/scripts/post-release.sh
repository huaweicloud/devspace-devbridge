#!/usr/bin/env bash
# =============================================================================
# scripts/post-release.sh - Release 后处理：上传到 GitCode Release
#
# GoReleaser after.hooks 调用。从 dist/ 收集产物 + dist-extra/ 收集安装脚本，
# 组装到 dist/gitcode/ 目录，然后调 upload-gitcode-release.sh 上传。
#
# 用法（由 GoReleaser after.hooks 自动调用）:
#   ./scripts/post-release.sh <version>
#
# 环境变量:
#   GITCODE_TOKEN  GitCode 个人访问令牌（留空则跳过）
# =============================================================================
set -euo pipefail

VERSION="${1:?用法: post-release.sh <version>}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DIST_DIR="${PROJECT_ROOT}/dist"
EXTRA_DIR="${PROJECT_ROOT}/dist-extra"
GITCODE_DIR="${DIST_DIR}/gitcode"

# ---- 未传令牌则跳过 ----
if [[ -z "${GITCODE_TOKEN:-}" ]]; then
    echo "⚠️  未设置 GITCODE_TOKEN，跳过 GitCode Release 上传"
    exit 0
fi

# ---- 收集产物到 dist/gitcode/ ----
mkdir -p "${GITCODE_DIR}"

# 压缩包 + 校验文件
cp "${DIST_DIR}"/devbridge_*.tar.gz "${GITCODE_DIR}/"
cp "${DIST_DIR}/checksums.txt" "${GITCODE_DIR}/"

# 烤制的安装脚本
if [[ -f "${EXTRA_DIR}/install.sh" ]]; then
    cp "${EXTRA_DIR}/install.sh" "${GITCODE_DIR}/"
fi
if [[ -f "${EXTRA_DIR}/install.ps1" ]]; then
    cp "${EXTRA_DIR}/install.ps1" "${GITCODE_DIR}/"
fi

echo "GitCode 上传目录内容:"
ls -la "${GITCODE_DIR}"/

# ---- 上传到 GitCode Release ----
# upload-gitcode-release.sh 会重新烤制 install 脚本，将下载源指向 GitCode 地址
"${SCRIPT_DIR}/upload-gitcode-release.sh" \
    -t "${GITCODE_TOKEN}" \
    -o CloudDeveloperDepartment \
    -r devbrige \
    -v "${VERSION}" \
    -d "${GITCODE_DIR}" \
    -n "${VERSION}" \
    -b "DevBridge CLI ${VERSION} - 同步自 GitHub Release"
