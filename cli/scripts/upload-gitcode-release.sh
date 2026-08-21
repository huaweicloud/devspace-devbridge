#!/usr/bin/env bash
# =============================================================================
# scripts/upload-gitcode-release.sh - 将构建产物上传到 GitCode Release
#
# 功能:
#   1. 在 GitCode 仓库创建 Release（若已存在则复用）
#   2. 将指定目录下的所有文件作为附件上传到该 Release
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

API_BASE="https://api.gitcode.com/api/v5"

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
# 1. 创建或获取 Release
# ---------------------------------------------------------------------------
log_info "准备 GitCode Release: ${OWNER}/${REPO}, tag=${VERSION}"

# 先尝试按 tag 查询是否已存在
RELEASE_ID=""
HTTP_CODE=$(curl -s -o /tmp/gc_release_check.json -w "%{http_code}" \
  -H "Authorization: Bearer ${TOKEN}" \
  "$(api_url "/releases/tags/${VERSION}")")

if [[ "$HTTP_CODE" == "200" ]]; then
  RELEASE_ID=$(jq -r '.id' /tmp/gc_release_check.json)
  log_info "Release 已存在（tag=${VERSION}），复用 release_id=${RELEASE_ID}"
elif [[ "$HTTP_CODE" == "404" ]]; then
  log_info "Release 不存在，开始创建..."

  # 创建 Release
  HTTP_CODE=$(curl -s -o /tmp/gc_release_create.json -w "%{http_code}" \
    -X POST \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    "$(api_url "/releases")" \
    -d "$(jq -n \
      --arg tag "$VERSION" \
      --arg name "$RELEASE_NAME" \
      --arg body "$RELEASE_BODY" \
      '{tag_name: $tag, name: $name, body: $body}')")

  if [[ "$HTTP_CODE" == "201" ]]; then
    RELEASE_ID=$(jq -r '.id' /tmp/gc_release_create.json)
    log_info "Release 创建成功，release_id=${RELEASE_ID}"
  else
    log_error "创建 Release 失败 (HTTP ${HTTP_CODE}): $(cat /tmp/gc_release_create.json)"
  fi
else
  log_error "查询 Release 失败 (HTTP ${HTTP_CODE}): $(cat /tmp/gc_release_check.json)"
fi

[[ -n "$RELEASE_ID" ]] || log_error "未能获取 release_id"

# ---------------------------------------------------------------------------
# 2. 上传所有产物文件
# ---------------------------------------------------------------------------
# 收集目录下所有文件（按文件名排序，输出可复现）
mapfile -t FILES < <(find "$DIR" -maxdepth 1 -type f | sort)

if [[ ${#FILES[@]} -eq 0 ]]; then
  log_error "目录 ${DIR} 下没有文件"
fi

log_info "共 ${#FILES[@]} 个文件待上传"

SUCCESS=0
FAILED=0

for FILE in "${FILES[@]}"; do
  FILENAME=$(basename "$FILE")
  FILESIZE=$(stat -c%s "$FILE" 2>/dev/null || stat -f%z "$FILE" 2>/dev/null)

  log_info "上传: ${FILENAME} (${FILESIZE} bytes)"

  HTTP_CODE=$(curl -s -o /tmp/gc_upload_resp.json -w "%{http_code}" \
    -X POST \
    -H "Authorization: Bearer ${TOKEN}" \
    -F "attachment=@${FILE}" \
    "$(api_url "/releases/${RELEASE_ID}/assets")")

  if [[ "$HTTP_CODE" == "201" || "$HTTP_CODE" == "200" ]]; then
    log_info "  ✅ ${FILENAME} 上传成功"
    ((SUCCESS++))
  else
    log_warn "  ❌ ${FILENAME} 上传失败 (HTTP ${HTTP_CODE}): $(cat /tmp/gc_upload_resp.json)"
    ((FAILED++))
  fi
done

# ---------------------------------------------------------------------------
# 3. 汇总
# ---------------------------------------------------------------------------
echo ""
log_info "上传完成: 成功 ${SUCCESS}/${#FILES[@]}, 失败 ${FAILED}"

if [[ "$FAILED" -gt 0 ]]; then
  log_error "有 ${FAILED} 个文件上传失败"
fi

log_info "GitCode Release: https://gitcode.com/${OWNER}/${REPO}/releases/${VERSION}"
