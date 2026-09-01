#!/usr/bin/env bash
# =============================================================================
# scripts/bake-install.sh - 烤制安装脚本
#
# 将版本号和下载地址注入 install.sh / install.ps1，输出到 dist-extra/
#
# 用法:
#   ./scripts/bake-install.sh <version> <release_url>
#
# 示例:
#   ./scripts/bake-install.sh 1.0.0-release "https://github.com/huaweicloud/devspace-devbridge/releases/download/1.0.0-release"
#
# 产物:
#   dist-extra/install.sh    — 版本号和下载地址已注入
#   dist-extra/install.ps1   — 版本号和下载地址已注入
# =============================================================================
set -euo pipefail

VERSION="${1:?用法: bake-install.sh <version> <release_url>}"
RELEASE_URL="${2:?用法: bake-install.sh <version> <release_url>}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
OUTPUT_DIR="${PROJECT_ROOT}/dist-extra"

mkdir -p "${OUTPUT_DIR}"

# ---- 烤制 install.sh ----
sed -e "s|^DEFAULT_VERSION=.*|DEFAULT_VERSION=\"${VERSION}\"|" \
    -e "s|^DEFAULT_ARTIFACT_URL=.*|DEFAULT_ARTIFACT_URL=\"${RELEASE_URL}\"|" \
    "${PROJECT_ROOT}/install.sh" > "${OUTPUT_DIR}/install.sh"
chmod +x "${OUTPUT_DIR}/install.sh"

# ---- 烤制 install.ps1 ----
sed -e "s|DEFAULT_VERSION = \".*\"|DEFAULT_VERSION = \"${VERSION}\"|" \
    -e "s|DEFAULT_ARTIFACT_URL = \".*\"|DEFAULT_ARTIFACT_URL = \"${RELEASE_URL}\"|" \
    "${PROJECT_ROOT}/install.ps1" > "${OUTPUT_DIR}/install.ps1"

echo "=== baked install.sh ==="
grep -E '^DEFAULT_ARTIFACT_URL=|^DEFAULT_VERSION=' "${OUTPUT_DIR}/install.sh"
echo "=== baked install.ps1 ==="
grep -E 'DEFAULT_ARTIFACT_URL = "|DEFAULT_VERSION = "' "${OUTPUT_DIR}/install.ps1" | head -2
