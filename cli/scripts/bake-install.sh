#!/usr/bin/env bash
# =============================================================================
# scripts/bake-install.sh - 烤制安装脚本
#
# 将版本号和下载地址注入 install.sh / install.ps1，按渠道输出到 dist-extra/：
#
#   dist-extra/install.sh / install.ps1      — GitHub 渠道（指向 GitHub Release）
#   dist-extra/obs/install.sh / install.ps1  — OBS 渠道（指向 OBS 版本化目录）
#
# GitCode 渠道无需单独烤制：post-release.sh 上传时由
# upload-gitcode-release.sh 重烤为 GitCode 地址。
#
# 用法:
#   ./scripts/bake-install.sh <version> <github_release_url> [obs_release_url]
#
# 示例:
#   ./scripts/bake-install.sh 1.0.0-release \
#     "https://github.com/huaweicloud/devspace-devbridge/releases/download/1.0.0-release" \
#     "https://tools-artifact.developer.huaweicloud.com/sharedata/devbridge/releases/download/1.0.0-release"
#
# 产物:
#   dist-extra/install.sh / install.ps1   — GitHub 渠道
#   dist-extra/obs/install.sh / .ps1      — OBS 渠道（传入 obs_release_url 时生成）
# =============================================================================
set -euo pipefail

VERSION="${1:?用法: bake-install.sh <version> <github_release_url> [obs_release_url]}"
GITHUB_RELEASE_URL="${2:?用法: bake-install.sh <version> <github_release_url> [obs_release_url]}"
OBS_RELEASE_URL="${3:-}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
OUTPUT_DIR="${PROJECT_ROOT}/dist-extra"

mkdir -p "${OUTPUT_DIR}"

bake() {
    local src="$1" dst_dir="$2" artifact_url="$3"
    mkdir -p "${dst_dir}"
    sed -e "s|^DEFAULT_VERSION=.*|DEFAULT_VERSION=\"${VERSION}\"|" \
        -e "s|^DEFAULT_ARTIFACT_URL=.*|DEFAULT_ARTIFACT_URL=\"${artifact_url}\"|" \
        "${src}/install.sh" > "${dst_dir}/install.sh"
    chmod +x "${dst_dir}/install.sh"
    sed -e "s|DEFAULT_VERSION = \".*\"|DEFAULT_VERSION = \"${VERSION}\"|" \
        -e "s|DEFAULT_ARTIFACT_URL = \".*\"|DEFAULT_ARTIFACT_URL = \"${artifact_url}\"|" \
        "${src}/install.ps1" > "${dst_dir}/install.ps1"
    echo "=== baked ${dst_dir} ==="
    grep -E '^DEFAULT_ARTIFACT_URL=|^DEFAULT_VERSION=' "${dst_dir}/install.sh"
    grep -E 'DEFAULT_ARTIFACT_URL = "|DEFAULT_VERSION = "' "${dst_dir}/install.ps1" | head -2
}

# ---- GitHub 渠道 ----
bake "${PROJECT_ROOT}" "${OUTPUT_DIR}" "${GITHUB_RELEASE_URL}"

# ---- OBS 渠道（可选）----
if [[ -n "${OBS_RELEASE_URL}" ]]; then
    bake "${PROJECT_ROOT}" "${OUTPUT_DIR}/obs" "${OBS_RELEASE_URL}"
fi
