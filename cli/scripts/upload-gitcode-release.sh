#!/usr/bin/env bash
# =============================================================================
# scripts/upload-gitcode-release.sh - upload build artifacts to GitCode Release
#
# What it does:
#   1. Create a Release in the GitCode repo (reuse it if it already exists)
#   2. Re-bake install.sh / install.ps1 to point the download source at the GitCode Release
#   3. Upload every file in the given directory as an asset to that Release
#   4. Sync the rolling "latest" Release (holds only the install scripts) for
#      one-click installs via /releases/download/latest/
#
# Usage:
#   ./scripts/upload-gitcode-release.sh \
#     -t <gitcode_token> \
#     -o <owner>          # GitCode repo namespace (org or personal path)
#     -r <repo>           # GitCode repo path
#     -v <version>        # version / tag name
#     -d <dir>            # artifact directory; all files under it are uploaded
#     [-n <release_name>] # Release name (defaults to version)
#     [-b <release_body>] # Release description (defaults to empty)
#
# Example:
#   ./scripts/upload-gitcode-release.sh \
#     -t "$GITCODE_TOKEN" \
#     -o CloudDeveloperDepartment \
#     -r devbrige \
#     -v 1.0.0 \
#     -d ./bin
#
# Dependencies: curl, jq
# =============================================================================
set -euo pipefail

# ---------------------------------------------------------------------------
# Defaults
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
# Logging
# ---------------------------------------------------------------------------
log_info()  { echo -e "\033[0;32m[INFO]\033[0m  $*"; }
log_warn()  { echo -e "\033[1;33m[WARN]\033[0m  $*"; }
log_error() { echo -e "\033[0;31m[ERROR]\033[0m $*" >&2; exit 1; }

# ---------------------------------------------------------------------------
# Usage
# ---------------------------------------------------------------------------
usage() {
  cat <<EOF
Upload build artifacts to GitCode Release

Usage:
  $0 -t <token> -o <owner> -r <repo> -v <version> -d <dir> [-n <name>] [-b <body>]

Options:
  -t, --token TOKEN       GitCode personal access token (required)
  -o, --owner OWNER       repo namespace path (required)
  -r, --repo REPO         repo path (required)
  -v, --version VERSION   version / tag name (required)
  -d, --dir DIR           artifact directory; all files under it are uploaded (required)
  -n, --name NAME         Release name (defaults to version)
  -b, --body BODY         Release description (defaults to empty)
  -h, --help              show help
EOF
  exit 0
}

# ---------------------------------------------------------------------------
# Parse arguments
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
    *)            log_error "unknown option: $1" ;;
  esac
done

# ---------------------------------------------------------------------------
# Validate required arguments
# ---------------------------------------------------------------------------
[[ -n "$TOKEN" ]]   || log_error "missing -t/--token"
[[ -n "$OWNER" ]]   || log_error "missing -o/--owner"
[[ -n "$REPO" ]]    || log_error "missing -r/--repo"
[[ -n "$VERSION" ]] || log_error "missing -v/--version"
[[ -n "$DIR" ]]     || log_error "missing -d/--dir"
[[ -d "$DIR" ]]     || log_error "directory not found: $DIR"

# Check dependencies
command -v curl >/dev/null || log_error "curl is required"
command -v jq   >/dev/null || log_error "jq (JSON parser) is required"

RELEASE_NAME="${RELEASE_NAME:-$VERSION}"

# ---------------------------------------------------------------------------
# API helpers
# ---------------------------------------------------------------------------
api_url() {
  echo "${API_BASE}/repos/${OWNER}/${REPO}$1"
}

# ---------------------------------------------------------------------------
# find_or_create_release - find or create a Release, returning the tag.
#
# GitCode API differs from stock Gitea:
#   - the Release response has no top-level id field; all APIs use the tag name
#   - creating a Release returns HTTP 200 rather than 201
# ---------------------------------------------------------------------------
find_or_create_release() {
  local tag="$1" name="$2" body="${3:-}"

  # First query by tag to see whether it already exists
  local code
  code=$(curl -s --connect-timeout 30 --max-time 60 -o /tmp/gc_rel_find.json -w "%{http_code}" \
    -H "Authorization: Bearer ${TOKEN}" \
    "$(api_url "/releases/tags/${tag}")")

  if [[ "$code" == "200" ]]; then
    log_info "Release already exists (tag=${tag})" >&2
    echo "$tag"
    return 0
  fi

  if [[ "$code" != "404" ]]; then
    log_error "failed to query Release (HTTP ${code}): $(cat /tmp/gc_rel_find.json)"
  fi

  # 404 -> create
  log_info "Release not found (tag=${tag}), creating..." >&2

  code=$(curl -s --connect-timeout 30 --max-time 60 -o /tmp/gc_rel_create.json -w "%{http_code}" \
    -X POST \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    "$(api_url "/releases")" \
    -d "$(jq -n \
      --arg t "$tag" \
      --arg n "$name" \
      --arg b "$body" \
      '{tag_name: $t, name: $n, body: $b}')")

  # GitCode returns either 200 or 201 on success
  if [[ "$code" != "200" && "$code" != "201" ]]; then
    log_error "failed to create Release (HTTP ${code}): $(cat /tmp/gc_rel_create.json)"
  fi

  log_info "Release created (tag=${tag})" >&2
  echo "$tag"
}

# ---------------------------------------------------------------------------
# delete_all_assets - delete all existing Release assets (attach type).
#
# GitCode API: DELETE /repos/:owner/:repo/releases/:tag/attach_files/:attach_file_id
# Only type=="attach" assets have an id; type=="source" is an auto-generated
# source archive and must not be deleted.
# ---------------------------------------------------------------------------
delete_all_assets() {
  local tag="$1"

  # Fetch Release details (including the asset list)
  local code
  code=$(curl -s --connect-timeout 30 --max-time 60 -o /tmp/gc_rel_assets.json -w "%{http_code}" \
    -H "Authorization: Bearer ${TOKEN}" \
    "$(api_url "/releases/tags/${tag}")")

  if [[ "$code" != "200" ]]; then
    log_warn "failed to fetch Release asset list (HTTP ${code}), skipping cleanup"
    return 0
  fi

  # Delete only type=="attach" assets (uploaded files), keep type=="source" (auto source archive)
  local asset_ids
  mapfile -t asset_ids < <(jq -r '.assets[] | select(.type == "attach") | .id // empty' /tmp/gc_rel_assets.json)

  if [[ ${#asset_ids[@]} -eq 0 ]]; then
    log_info "  no existing assets, nothing to clean"
    return 0
  fi

  log_info "  cleaning ${#asset_ids[@]} old assets..."
  for aid in "${asset_ids[@]}"; do
    curl -s --connect-timeout 30 --max-time 60 -o /dev/null \
      -X DELETE \
      -H "Authorization: Bearer ${TOKEN}" \
      "$(api_url "/releases/${tag}/attach_files/${aid}")"
  done
  log_info "  old assets cleaned"
}

# ---------------------------------------------------------------------------
# upload_asset - upload a single file to a Release (two-step GitCode upload).
#
# GitCode asset upload differs from Gitea/GitHub; it is not
# POST /releases/{id}/assets but a two-step flow:
#   1. GET /repos/:owner/:repo/releases/:tag/upload_url?file_name=<filename>
#      -> returns an OBS presigned URL and required headers
#   2. PUT the file content to the presigned URL with the given headers
#      -> returns "success" (HTTP 200)
# ---------------------------------------------------------------------------
upload_asset() {
  local tag="$1" file="$2"
  local filename
  filename=$(basename "$file")
  local filesize
  filesize=$(stat -c%s "$file" 2>/dev/null || stat -f%z "$file" 2>/dev/null)

  log_info "  uploading: ${filename} (${filesize} bytes)"

  local max_retries=10
  local attempt

  for attempt in $(seq 1 "$max_retries"); do
    # Step 1: get the OBS presigned upload URL
    local resp code
    resp=$(curl -s --connect-timeout 30 --max-time 60 -w "\n%{http_code}" \
      -H "Authorization: Bearer ${TOKEN}" \
      "$(api_url "/releases/${tag}/upload_url")?file_name=$(python3 -c "import urllib.parse; print(urllib.parse.quote('${filename}'))")")

    code=$(echo "$resp" | tail -1)
    local body
    body=$(echo "$resp" | sed '$d')

    if [[ "$code" != "200" ]]; then
      log_warn "    ❌ ${filename} failed to get upload URL (HTTP ${code}): ${body}"
      # Retry on 5xx, not 4xx
      if [[ "${code:0:1}" == "5" && "$attempt" -lt "$max_retries" ]]; then
        log_warn "    retry ${attempt}/${max_retries} (waiting 5s)..."
        sleep 5
        continue
      fi
      return 1
    fi

    # Parse the presigned URL and headers
    local upload_url content_type obs_meta obs_acl obs_callback
    upload_url=$(echo "$body" | jq -r '.url')
    content_type=$(echo "$body" | jq -r '.headers["Content-Type"] // "application/octet-stream"')
    obs_meta=$(echo "$body" | jq -r '.headers["x-obs-meta-project-id"] // empty')
    obs_acl=$(echo "$body" | jq -r '.headers["x-obs-acl"] // empty')
    obs_callback=$(echo "$body" | jq -r '.headers["x-obs-callback"] // empty')

    # Step 2: PUT the file to the presigned URL.
    # Clear the response file first so a leftover "success" body can't be misread.
    : > /tmp/gc_upload_resp.txt
    local put_code
    put_code=$(curl -s --connect-timeout 30 --max-time 600 -o /tmp/gc_upload_resp.txt -w "%{http_code}" \
      -X PUT \
      -H "Content-Type: ${content_type}" \
      ${obs_meta:+-H "x-obs-meta-project-id: ${obs_meta}"} \
      ${obs_acl:+-H "x-obs-acl: ${obs_acl}"} \
      ${obs_callback:+-H "x-obs-callback: ${obs_callback}"} \
      -H "Expect:" \
      --data-binary @"${file}" \
      "$upload_url")

    local put_body
    put_body=$(cat /tmp/gc_upload_resp.txt 2>/dev/null)

    # Success criteria:
    #   - HTTP 200/201: normal success
    #   - HTTP 100 + body "success": curl misread 100 Continue as the final code
    #     (Expect:100-continue issue on slow links), but OBS actually uploaded.
    if [[ "$put_code" == "200" || "$put_code" == "201" ]]; then
      log_info "    ✅ ${filename} uploaded"
      return 0
    elif [[ "$put_code" == "100" && "$put_body" == "success" ]]; then
      log_warn "    ⚠️ ${filename} got HTTP 100 (curl 100-continue false positive), body=success, treating as success"
      return 0
    else
      log_warn "    ❌ ${filename} upload failed (HTTP ${put_code}): ${put_body}"
      # Retry on 5xx server errors, or 203 callback errors (GitCode obs_callback intermittently 400)
      if [[ "${put_code:0:1}" == "5" || "$put_code" == "203" || "$put_code" == "000" ]] && [[ "$attempt" -lt "$max_retries" ]]; then
        log_warn "    retry ${attempt}/${max_retries} (waiting 5s)..."
        sleep 5
        continue
      fi
      return 1
    fi
  done

  return 1
}

# ===========================================================================
# Main
# ===========================================================================

# ---------------------------------------------------------------------------
# 1. Create or fetch the version Release
# ---------------------------------------------------------------------------
log_info "===== 1/4 preparing version Release: ${OWNER}/${REPO}, tag=${VERSION} ====="

RELEASE_TAG=$(find_or_create_release "$VERSION" "$RELEASE_NAME" "$RELEASE_BODY")
[[ -n "$RELEASE_TAG" ]] || log_error "failed to obtain release tag"

# ---------------------------------------------------------------------------
# 2. Re-bake the install scripts to point at the GitCode Release
# ---------------------------------------------------------------------------
log_info "===== 2/4 re-baking install scripts ====="

# CI-baked install.sh/install.ps1 point DEFAULT_ARTIFACT_URL at GitHub Release;
# before uploading to GitCode, replace it with the GitCode Release URL so the
# scripts fetched from GitCode download the binary from GitCode rather than GitHub.
GITCODE_RELEASE_URL="${GITCODE_BASE}/${OWNER}/${REPO}/releases/download/${VERSION}"
log_info "download source -> ${GITCODE_RELEASE_URL}"

INSTALL_SH="${DIR}/install.sh"
INSTALL_PS1="${DIR}/install.ps1"

if [[ -f "${INSTALL_SH}" ]]; then
  sed -i "s|^DEFAULT_ARTIFACT_URL=.*|DEFAULT_ARTIFACT_URL=\"${GITCODE_RELEASE_URL}\"|" "${INSTALL_SH}"
  log_info "  install.sh updated"
else
  log_warn "  install.sh not found, skipping"
fi

if [[ -f "${INSTALL_PS1}" ]]; then
  sed -i "s|DEFAULT_ARTIFACT_URL = \".*\"|DEFAULT_ARTIFACT_URL = \"${GITCODE_RELEASE_URL}\"|" "${INSTALL_PS1}"
  log_info "  install.ps1 updated"
else
  log_warn "  install.ps1 not found, skipping"
fi

# ---------------------------------------------------------------------------
# 3. Upload all artifact files to the version Release
# ---------------------------------------------------------------------------
log_info "===== 3/4 uploading artifacts to the version Release ====="

# Clean old assets (avoids same-name file conflicts on re-runs)
log_info "cleaning old assets of the version Release..."
delete_all_assets "$RELEASE_TAG"

# Collect all files under the directory (sorted for reproducible output)
mapfile -t FILES < <(find "$DIR" -maxdepth 1 -type f | sort)

if [[ ${#FILES[@]} -eq 0 ]]; then
  log_error "no files under directory ${DIR}"
fi

log_info "total ${#FILES[@]} files to upload"

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
log_info "version Release upload done: ${SUCCESS}/${#FILES[@]} succeeded, ${FAILED} failed"

if [[ "$FAILED" -gt 0 ]]; then
  log_error "${FAILED} files failed to upload"
fi

# ---------------------------------------------------------------------------
# 4. Sync the rolling "latest" Release (only the install scripts)
# ---------------------------------------------------------------------------
log_info "===== 4/4 syncing the rolling latest Release ====="

# The "latest" Release only holds install.sh and install.ps1; the scripts already
# have the real version and download URL baked in, so users can one-click install
# the latest via /releases/download/latest/install.sh.

LATEST_NAME="Latest (${VERSION})"
LATEST_BODY="Automatically maintained rolling Release, always pointing at ${VERSION}.
Download URL: ${GITCODE_RELEASE_URL}"

LATEST_TAG_VALUE=$(find_or_create_release "$LATEST_TAG" "$LATEST_NAME" "$LATEST_BODY")
[[ -n "$LATEST_TAG_VALUE" ]] || log_error "failed to obtain the latest release tag"

# Clean old assets
log_info "cleaning old assets of the latest Release..."
delete_all_assets "$LATEST_TAG_VALUE"

# Upload the install scripts to the latest Release
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
  log_warn "${LATEST_FAILED} files of the latest Release failed to upload"
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo ""
log_info "========================================"
log_info "All done!"
log_info "========================================"
log_info ""
log_info "Version Release:"
log_info "  ${GITCODE_BASE}/${OWNER}/${REPO}/releases/${VERSION}"
log_info "  curl -fsSL ${GITCODE_BASE}/${OWNER}/${REPO}/releases/download/${VERSION}/install.sh | bash"
log_info ""
log_info "Latest (rolling latest):"
log_info "  ${GITCODE_BASE}/${OWNER}/${REPO}/releases/${LATEST_TAG}"
log_info "  curl -fsSL ${GITCODE_BASE}/${OWNER}/${REPO}/releases/download/${LATEST_TAG}/install.sh | bash"