#!/usr/bin/env bash
# =============================================================================
# scripts/upload-gitcode-release.sh - 将构建产物上传到 GitCode Release
#
# 功能:
#   1. 在 GitCode 仓库创建 Release（若已存在则复用）
#   2. 重新烤制 install.sh / install.ps1，将下载源替换为 GitCode Release 地址
#   3. 将指定目录下的所有文件作为附件上传到该 Release
#   4. 同步更新 "latest" 滚动 Release（只放 install 脚本），实现 /releases/download/latest/ 一键安装
#
# 用法:
#   ./scripts/upload-gitcode-release.sh \
#     -t <gitcode_token> \
#     -o <owner>          # GitCode 仓库所属空间（组织或个人 path）
#     -r <repo>           # GitCode 仓库路径
#     -v <version>        # 版本号 / tag name
#     -d <dir>            # 产物目录，该目录下所有文件都会被上传
#     [-n <release_name>] # Release 名称（默认同 version）
#     [-b <release_body>] # Release 描述（默认空）
#
# 示例:
#   ./scripts/upload-gitcode-release.sh \
#     -t "$GITCODE_TOKEN" \
#     -o CloudDeveloperDepartment \
#     -r devbrige \
#     -v 1.0.0 \
#     -d ./bin
#
# 依赖: curl, jq
# =============================================================================
set -euo pipefail

# ---------------------------------------------------------------------------
# 默认配置
# ---------------------------------------------------------------------------
TOKEN=""
OWNER=""
REPO=""
VERSION=""
DIR=""
RELEASE_NAME=""
RELEASE_BODY=""
LATEST_TAG="latest"

API_BASE="https://api.gitcode.com/api/v5"
GITCODE_BASE="https://gitcode.com"

# ---------------------------------------------------------------------------
# 日志
# ---------------------------------------------------------------------------
log_info()  { echo -e "\033[0;32m[INFO]\033[0m  $*"; }
log_warn()  { echo -e "\033[1;33m[WARN]\033[0m  $*"; }
log_error() { echo -e "\033[0;31m[ERROR]\033[0m $*" >&2; exit 1; }

# ---------------------------------------------------------------------------
# 用法
# ---------------------------------------------------------------------------
usage() {
  cat <<EOF
上传构建产物到 GitCode Release

用法:
  $0 -t <token> -o <owner> -r <repo> -v <version> -d <dir> [-n <name>] [-b <body>]

选项:
  -t, --token TOKEN       GitCode 个人访问令牌（必填）
  -o, --owner OWNER       仓库所属空间地址（必填）
  -r, --repo REPO         仓库路径（必填）
  -v, --version VERSION   版本号 / tag name（必填）
  -d, --dir DIR           产物目录，其下所有文件都会上传（必填）
  -n, --name NAME         Release 名称（默认同 version）
  -b, --body BODY         Release 描述（默认空）
  -h, --help              显示帮助
EOF
  exit 0
}

# ---------------------------------------------------------------------------
# 解析参数
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
  case "$1" in
    -t|--token)   TOKEN="$2"; shift 2 ;;
    -o|--owner)   OWNER="$2"; shift 2 ;;
    -r|--repo)    REPO="$2"; shift 2 ;;
    -v|--version) VERSION="$2"; shift 2 ;;
    -d|--dir)     DIR="$2"; shift 2 ;;
    -n|--name)    RELEASE_NAME="$2"; shift 2 ;;
    -b|--body)    RELEASE_BODY="$2"; shift 2 ;;
    -h|--help)    usage ;;
    *)            log_error "未知参数: $1" ;;
  esac
done

# ---------------------------------------------------------------------------
# 校验必填参数
# ---------------------------------------------------------------------------
[[ -n "$TOKEN" ]]   || log_error "缺少 -t/--token"
[[ -n "$OWNER" ]]   || log_error "缺少 -o/--owner"
[[ -n "$REPO" ]]    || log_error "缺少 -r/--repo"
[[ -n "$VERSION" ]] || log_error "缺少 -v/--version"
[[ -n "$DIR" ]]     || log_error "缺少 -d/--dir"
[[ -d "$DIR" ]]     || log_error "目录不存在: $DIR"

# 检查依赖
command -v curl >/dev/null || log_error "需要 curl"
command -v jq   >/dev/null || log_error "需要 jq（JSON 解析工具）"

RELEASE_NAME="${RELEASE_NAME:-$VERSION}"

# ---------------------------------------------------------------------------
# API 辅助函数
# ---------------------------------------------------------------------------
api_url() {
  echo "${API_BASE}/repos/${OWNER}/${REPO}$1"
}

# ---------------------------------------------------------------------------
# find_or_create_release - 查找或创建 Release，返回 tag（GitCode 用 tag 标识 Release）
#
# GitCode API 与标准 Gitea 有差异：
#   - Release 响应体没有顶层 id 字段，所有 API 用 tag name 标识 Release
#   - 创建 Release 返回 HTTP 200 而非 201
# ---------------------------------------------------------------------------
find_or_create_release() {
  local tag="$1" name="$2" body="${3:-}"

  # 先按 tag 查询是否已存在
  local code
  code=$(curl -s --connect-timeout 10 --max-time 30 -o /tmp/gc_rel_find.json -w "%{http_code}" \
    -H "Authorization: Bearer ${TOKEN}" \
    "$(api_url "/releases/tags/${tag}")")

  if [[ "$code" == "200" ]]; then
    log_info "Release 已存在（tag=${tag}）" >&2
    echo "$tag"
    return 0
  fi

  if [[ "$code" != "404" ]]; then
    log_error "查询 Release 失败 (HTTP ${code}): $(cat /tmp/gc_rel_find.json)"
  fi

  # 404 → 创建
  log_info "Release 不存在（tag=${tag}），创建..." >&2

  code=$(curl -s --connect-timeout 10 --max-time 30 -o /tmp/gc_rel_create.json -w "%{http_code}" \
    -X POST \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    "$(api_url "/releases")" \
    -d "$(jq -n \
      --arg t "$tag" \
      --arg n "$name" \
      --arg b "$body" \
      '{tag_name: $t, name: $n, body: $b}')")

  # GitCode 返回 200 或 201 均为成功
  if [[ "$code" != "200" && "$code" != "201" ]]; then
    log_error "创建 Release 失败 (HTTP ${code}): $(cat /tmp/gc_rel_create.json)"
  fi

  log_info "Release 创建成功（tag=${tag}）" >&2
  echo "$tag"
}

# ---------------------------------------------------------------------------
# delete_all_assets - 删除 Release 的所有现有附件（attach 类型）
#
# GitCode API：DELETE /repos/:owner/:repo/releases/:tag/attach_files/:attach_file_id
# 只有 type=="attach" 的附件有 id 字段，type=="source" 的是自动生成的源码包，不可删除。
# ---------------------------------------------------------------------------
delete_all_assets() {
  local tag="$1"

  # 获取 Release 详情（含 assets 列表）
  local code
  code=$(curl -s --connect-timeout 10 --max-time 30 -o /tmp/gc_rel_assets.json -w "%{http_code}" \
    -H "Authorization: Bearer ${TOKEN}" \
    "$(api_url "/releases/tags/${tag}")")

  if [[ "$code" != "200" ]]; then
    log_warn "无法获取 Release 附件列表 (HTTP ${code})，跳过清理"
    return 0
  fi

  # 只删除 type=="attach" 的附件（上传的文件），保留 type=="source"（自动生成的源码包）
  local asset_ids
  mapfile -t asset_ids < <(jq -r '.assets[] | select(.type == "attach") | .id // empty' /tmp/gc_rel_assets.json)

  if [[ ${#asset_ids[@]} -eq 0 ]]; then
    log_info "  无现有附件，无需清理"
    return 0
  fi

  log_info "  清理 ${#asset_ids[@]} 个旧附件..."
  for aid in "${asset_ids[@]}"; do
    curl -s --connect-timeout 10 --max-time 30 -o /dev/null \
      -X DELETE \
      -H "Authorization: Bearer ${TOKEN}" \
      "$(api_url "/releases/${tag}/attach_files/${aid}")"
  done
  log_info "  旧附件已清理"
}

# ---------------------------------------------------------------------------
# upload_asset - 上传单个文件到 Release（GitCode 两步式上传）
#
# GitCode 的附件上传与 Gitea/GitHub 不同，不是 POST /releases/{id}/assets，
# 而是两步式：
#   1. GET /repos/:owner/:repo/releases/:tag/upload_url?file_name=<filename>
#      → 返回 OBS 预签名 URL 和所需 headers
#   2. PUT 文件内容到预签名 URL，带上指定的 headers
#      → 返回 "success" (HTTP 200)
# ---------------------------------------------------------------------------
upload_asset() {
  local tag="$1" file="$2"
  local filename
  filename=$(basename "$file")
  local filesize
  filesize=$(stat -c%s "$file" 2>/dev/null || stat -f%z "$file" 2>/dev/null)

  log_info "  上传: ${filename} (${filesize} bytes)"

  local max_retries=3
  local attempt

  for attempt in $(seq 1 "$max_retries"); do
    # Step 1: 获取 OBS 预签名上传地址
    local resp code
    resp=$(curl -s --connect-timeout 10 --max-time 30 -w "\n%{http_code}" \
      -H "Authorization: Bearer ${TOKEN}" \
      "$(api_url "/releases/${tag}/upload_url")?file_name=$(python3 -c "import urllib.parse; print(urllib.parse.quote('${filename}'))")")

    code=$(echo "$resp" | tail -1)
    local body
    body=$(echo "$resp" | sed '$d')

    if [[ "$code" != "200" ]]; then
      log_warn "    ❌ ${filename} 获取上传地址失败 (HTTP ${code}): ${body}"
      # 5xx 错误重试，4xx 不重试
      if [[ "${code:0:1}" == "5" && "$attempt" -lt "$max_retries" ]]; then
        log_warn "    第 ${attempt}/${max_retries} 次重试（等待 5 秒）..."
        sleep 5
        continue
      fi
      return 1
    fi

    # 解析预签名 URL 和 headers
    local upload_url content_type obs_meta obs_acl obs_callback
    upload_url=$(echo "$body" | jq -r '.url')
    content_type=$(echo "$body" | jq -r '.headers["Content-Type"] // "application/octet-stream"')
    obs_meta=$(echo "$body" | jq -r '.headers["x-obs-meta-project-id"] // empty')
    obs_acl=$(echo "$body" | jq -r '.headers["x-obs-acl"] // empty')
    obs_callback=$(echo "$body" | jq -r '.headers["x-obs-callback"] // empty')

    # Step 2: PUT 文件到预签名 URL
    local put_code
    put_code=$(curl -s --connect-timeout 10 --max-time 120 -o /tmp/gc_upload_resp.txt -w "%{http_code}" \
      -X PUT \
      -H "Content-Type: ${content_type}" \
      ${obs_meta:+-H "x-obs-meta-project-id: ${obs_meta}"} \
      ${obs_acl:+-H "x-obs-acl: ${obs_acl}"} \
      ${obs_callback:+-H "x-obs-callback: ${obs_callback}"} \
      --data-binary @"${file}" \
      "$upload_url")

    if [[ "$put_code" == "200" || "$put_code" == "201" ]]; then
      log_info "    ✅ ${filename} 上传成功"
      return 0
    else
      log_warn "    ❌ ${filename} 上传失败 (HTTP ${put_code}): $(cat /tmp/gc_upload_resp.txt 2>/dev/null)"
      # 5xx 错误重试，4xx 不重试
      if [[ "${put_code:0:1}" == "5" && "$attempt" -lt "$max_retries" ]]; then
        log_warn "    第 ${attempt}/${max_retries} 次重试（等待 5 秒）..."
        sleep 5
        continue
      fi
      return 1
    fi
  done

  return 1
}

# ===========================================================================
# 主流程
# ===========================================================================

# ---------------------------------------------------------------------------
# 1. 创建或获取版本 Release
# ---------------------------------------------------------------------------
log_info "===== 1/4 准备版本 Release: ${OWNER}/${REPO}, tag=${VERSION} ====="

RELEASE_TAG=$(find_or_create_release "$VERSION" "$RELEASE_NAME" "$RELEASE_BODY")
[[ -n "$RELEASE_TAG" ]] || log_error "未能获取 release tag"

# ---------------------------------------------------------------------------
# 2. 重新烤制 install 脚本，将下载源指向 GitCode Release
# ---------------------------------------------------------------------------
log_info "===== 2/4 重新烤制 install 脚本 ====="

# CI 烤制的 install.sh/install.ps1 里 DEFAULT_ARTIFACT_URL 指向 GitHub Release，
# 上传到 GitCode 前需要替换为 GitCode Release 下载地址，这样从 GitCode 拿到的
# install 脚本会从 GitCode 下载二进制，而非回退到 GitHub。
GITCODE_RELEASE_URL="${GITCODE_BASE}/${OWNER}/${REPO}/releases/download/${VERSION}"
log_info "下载源 → ${GITCODE_RELEASE_URL}"

INSTALL_SH="${DIR}/install.sh"
INSTALL_PS1="${DIR}/install.ps1"

if [[ -f "${INSTALL_SH}" ]]; then
  sed -i "s|^DEFAULT_ARTIFACT_URL=.*|DEFAULT_ARTIFACT_URL=\"${GITCODE_RELEASE_URL}\"|" "${INSTALL_SH}"
  log_info "  install.sh 已更新"
else
  log_warn "  install.sh 不存在，跳过"
fi

if [[ -f "${INSTALL_PS1}" ]]; then
  sed -i "s|DEFAULT_ARTIFACT_URL = \".*\"|DEFAULT_ARTIFACT_URL = \"${GITCODE_RELEASE_URL}\"|" "${INSTALL_PS1}"
  log_info "  install.ps1 已更新"
else
  log_warn "  install.ps1 不存在，跳过"
fi

# ---------------------------------------------------------------------------
# 3. 上传所有产物文件到版本 Release
# ---------------------------------------------------------------------------
log_info "===== 3/4 上传产物到版本 Release ====="

# 收集目录下所有文件（按文件名排序，输出可复现）
mapfile -t FILES < <(find "$DIR" -maxdepth 1 -type f | sort)

if [[ ${#FILES[@]} -eq 0 ]]; then
  log_error "目录 ${DIR} 下没有文件"
fi

log_info "共 ${#FILES[@]} 个文件待上传"

SUCCESS=0
FAILED=0

for FILE in "${FILES[@]}"; do
  if upload_asset "$RELEASE_TAG" "$FILE"; then
    SUCCESS=$((SUCCESS + 1))
  else
    FAILED=$((FAILED + 1))
  fi
done

echo ""
log_info "版本 Release 上传完成: 成功 ${SUCCESS}/${#FILES[@]}, 失败 ${FAILED}"

if [[ "$FAILED" -gt 0 ]]; then
  log_error "有 ${FAILED} 个文件上传失败"
fi

# ---------------------------------------------------------------------------
# 4. 同步 "latest" 滚动 Release（只放 install 脚本）
# ---------------------------------------------------------------------------
log_info "===== 4/4 同步 latest 滚动 Release ====="

# "latest" Release 里只放 install.sh 和 install.ps1，
# 脚本内部已烤制了真实版本号和版本 Release 的下载地址。
# 用户可通过 /releases/download/latest/install.sh 一键安装最新版。

LATEST_NAME="Latest (${VERSION})"
LATEST_BODY="自动维护的滚动 Release，始终指向最新版本 ${VERSION}。
实际下载地址: ${GITCODE_RELEASE_URL}"

LATEST_TAG_VALUE=$(find_or_create_release "$LATEST_TAG" "$LATEST_NAME" "$LATEST_BODY")
[[ -n "$LATEST_TAG_VALUE" ]] || log_error "未能获取 latest release tag"

# 清理旧附件
log_info "清理 latest Release 旧附件..."
delete_all_assets "$LATEST_TAG_VALUE"

# 上传 install 脚本到 latest Release
LATEST_SUCCESS=0
LATEST_FAILED=0

if [[ -f "${INSTALL_SH}" ]]; then
  if upload_asset "$LATEST_TAG_VALUE" "${INSTALL_SH}"; then
    LATEST_SUCCESS=$((LATEST_SUCCESS + 1))
  else
    LATEST_FAILED=$((LATEST_FAILED + 1))
  fi
fi

if [[ -f "${INSTALL_PS1}" ]]; then
  if upload_asset "$LATEST_TAG_VALUE" "${INSTALL_PS1}"; then
    LATEST_SUCCESS=$((LATEST_SUCCESS + 1))
  else
    LATEST_FAILED=$((LATEST_FAILED + 1))
  fi
fi

if [[ "$LATEST_FAILED" -gt 0 ]]; then
  log_warn "latest Release 有 ${LATEST_FAILED} 个文件上传失败"
fi

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------
echo ""
log_info "========================================"
log_info "全部完成!"
log_info "========================================"
log_info ""
log_info "版本 Release:"
log_info "  ${GITCODE_BASE}/${OWNER}/${REPO}/releases/${VERSION}"
log_info "  curl -fsSL ${GITCODE_BASE}/${OWNER}/${REPO}/releases/download/${VERSION}/install.sh | bash"
log_info ""
log_info "最新版（滚动 latest）:"
log_info "  ${GITCODE_BASE}/${OWNER}/${REPO}/releases/${LATEST_TAG}"
log_info "  curl -fsSL ${GITCODE_BASE}/${OWNER}/${REPO}/releases/download/${LATEST_TAG}/install.sh | bash"
