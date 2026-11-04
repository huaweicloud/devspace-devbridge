#!/bin/bash
#=============================================================================
# DevBridge SSH Tunnel Auto-Setup Script for ECS
#
# 功能：
#   1. 安装 DevBridge CLI
#   2. 用 API Key 登录
#   3. 创建持久隧道 + 添加 22 端口
#   4. 开机自启（systemd 优先，无 systemd 则 crontab @reboot 兜底）
#   5. 每次启动自动检查登录状态，过期则用 API Key 重新登录
#   6. 自动续期：剩余 < 24 小时时自动续到 720 小时
#   7. API Key 加密存储（AES-256-GCM + PBKDF2-SHA256，密钥从 machine-id 派生）
#
# 用法：
#   ./setup-devbridge-ssh.sh --api-key <YOUR_API_KEY>
#   ./setup-devbridge-ssh.sh --api-key <KEY> --plaintext   # 明文存储 API Key
#   HW_API_KEY=<key> ./setup-devbridge-ssh.sh
#   ./setup-devbridge-ssh.sh    # 交互式输入 API Key
#=============================================================================
set -euo pipefail

# ---------- 失败清理 ----------
cleanup_on_error() {
    log_error "脚本执行失败！可能留下残留资源："
    log_error "  - 隧道: devbridge list 查看并 devbridge delete <id> 清理"
    log_error "  - 配置: $CONFIG_DIR/ssh-tunnel.conf"
    log_error "  - Wrapper: $CONFIG_DIR/devbridge-ssh-wrapper.sh"
    log_error "  - 服务: /etc/systemd/system/devbridge-ssh.service"
}
trap cleanup_on_error ERR

# ---------- 常量 ----------
INSTALL_URL="https://tools-artifact.developer.huaweicloud.com/sharedata/devbridge/install.sh"
DEVBRIDGE_BIN="$HOME/.huawei/bin/devbridge"
CONFIG_DIR="$HOME/.huawei/devbridge"
CONFIG_FILE="$CONFIG_DIR/ssh-tunnel.conf"
WRAPPER_SCRIPT="$CONFIG_DIR/devbridge-ssh-wrapper.sh"
SERVICE_NAME="devbridge-ssh"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
PID_FILE="$CONFIG_DIR/devbridge-ssh.pid"
LOG_FILE="$CONFIG_DIR/devbridge-ssh.log"

TUNNEL_EXPIRATION=720
RENEW_THRESHOLD=24
SSH_PORT=22
ENCRYPTION_SALT="devbridge-ssh-tunnel-v1"

# ---------- 颜色输出 ----------
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'
log_info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
log_step()  { echo -e "${BLUE}[STEP]${NC} $*"; }

# ---------- 加解密函数 ----------
get_machine_pass() {
    local mid
    mid=$(cat /etc/machine-id 2>/dev/null || cat /var/lib/dbus/machine-id 2>/dev/null || uname -n)
    echo -n "${mid}${ENCRYPTION_SALT}"
}

encrypt_secret() {
    local plain="$1" pass
    pass=$(get_machine_pass)
    _DB_PASS="$pass" _DB_PLAIN="$plain" python3 -c "
import os, base64
from cryptography.hazmat.primitives.ciphers.aead import AESGCM
from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2HMAC
from cryptography.hazmat.primitives import hashes
pw = os.environ['_DB_PASS'].encode()
pt = os.environ['_DB_PLAIN'].encode()
salt = os.urandom(16)
kdf = PBKDF2HMAC(algorithm=hashes.SHA256(), length=32, salt=salt, iterations=100000)
key = kdf.derive(pw)
nonce = os.urandom(12)
ct = AESGCM(key).encrypt(nonce, pt, None)
print(base64.b64encode(salt + nonce + ct).decode(), end='')
"
}

decrypt_secret() {
    local enc="$1" pass
    pass=$(get_machine_pass)
    _DB_PASS="$pass" _DB_ENC="$enc" python3 -c "
import os, base64
from cryptography.hazmat.primitives.ciphers.aead import AESGCM
from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2HMAC
from cryptography.hazmat.primitives import hashes
pw = os.environ['_DB_PASS'].encode()
raw = base64.b64decode(os.environ['_DB_ENC'])
salt, nonce, ct = raw[:16], raw[16:28], raw[28:]
kdf = PBKDF2HMAC(algorithm=hashes.SHA256(), length=32, salt=salt, iterations=100000)
key = kdf.derive(pw)
pt = AESGCM(key).decrypt(nonce, ct, None)
print(pt.decode(), end='')
" 2>/dev/null
}

# ========== Step 0: 解析参数 ==========
log_step "0/7 解析参数"
API_KEY=""
PLAINTEXT=false
while [[ $# -gt 0 ]]; do
    case $1 in
        --api-key)        API_KEY="$2"; shift 2 ;;
        --api-key=*)      API_KEY="${1#*=}"; shift ;;
        --plaintext)      PLAINTEXT=true; shift ;;
        -h|--help)        sed -n '2,19p' "$0"; exit 0 ;;
        *) log_error "未知参数: $1"; exit 1 ;;
    esac
done
[[ -z "$API_KEY" ]] && API_KEY="${HW_API_KEY:-}"
if [[ -z "$API_KEY" ]]; then
    read -sp "请输入 DevBridge API Key: " API_KEY; echo
fi
[[ -z "$API_KEY" ]] && { log_error "API Key 不能为空"; exit 1; }
log_info "API Key 已获取"

# ========== Step 1: 前置检查 ==========
log_step "1/7 前置检查"
if [[ $EUID -ne 0 ]]; then
    log_error "需要 root 权限运行。请使用 sudo 或 root 用户执行。"
    exit 1
fi
command -v curl &>/dev/null || { log_error "需要 curl"; exit 1; }

# systemd 检查（非 systemd 系统使用 crontab @reboot 兜底）
HAS_SYSTEMD=false
if [[ -d /run/systemd/system ]] && command -v systemctl >/dev/null 2>&1; then
    HAS_SYSTEMD=true
fi

# python3 检查
if ! command -v python3 &>/dev/null; then
    log_info "python3 未安装，尝试自动安装..."
    if command -v yum &>/dev/null; then yum install -y python3
    elif command -v apt-get &>/dev/null; then apt-get update -qq && apt-get install -y python3
    elif command -v apk &>/dev/null; then apk add --no-cache python3
    else log_error "需要 python3，且无法自动安装（找不到 yum/apt-get/apk）"; exit 1; fi
    command -v python3 &>/dev/null || { log_error "python3 安装失败"; exit 1; }
    log_info "python3 安装成功"
fi

# pip3 检查，缺失则自动安装（部分镜像装完 python3 不带 pip）
if ! python3 -m pip --version &>/dev/null; then
    log_info "pip 未安装，尝试自动安装..."
    if command -v yum &>/dev/null; then yum install -y python3-pip 2>/dev/null || yum install -y python3-setuptools
    elif command -v apt-get &>/dev/null; then apt-get install -y python3-pip
    elif command -v apk &>/dev/null; then apk add --no-cache py3-pip
    else
        # 兜底：用 ensurepip 安装
        python3 -m ensurepip 2>/dev/null || { log_error "pip 安装失败，请手动执行: yum install python3-pip 或 apt-get install python3-pip"; exit 1; }
    fi
    python3 -m pip --version &>/dev/null || { log_error "pip 安装失败"; exit 1; }
    log_info "pip 安装成功"
fi

# cryptography 库检查，缺失则自动安装
if ! python3 -c "from cryptography.hazmat.primitives.ciphers.aead import AESGCM" 2>/dev/null; then
    log_info "cryptography 库未安装，尝试自动安装..."
    # 先装编译依赖（CentOS 7 等 old distro 从源码编译需要）
    if command -v yum &>/dev/null; then
        yum install -y gcc openssl-devel libffi-devel python3-devel 2>/dev/null
    elif command -v apt-get &>/dev/null; then
        apt-get install -y gcc libssl-dev libffi-dev python3-dev 2>/dev/null
    fi
    # 升级 pip 以支持 wheel 安装
    python3 -m pip install --upgrade pip 2>/dev/null || true
    # 尝试安装 cryptography：先试最新版 wheel，失败则试不需要 Rust 的旧版
    if python3 -m pip install cryptography 2>/dev/null; then
        log_info "cryptography 库安装成功"
    elif python3 -m pip install "cryptography==3.4.8" 2>/dev/null; then
        log_info "cryptography 库安装成功 (fallback v3.4.8)"
    elif python3 -m pip install "cryptography==2.9.2" 2>/dev/null; then
        log_info "cryptography 库安装成功 (fallback v2.9.2)"
    else
        log_error "cryptography 库安装失败，请手动执行:"
        log_error "  yum install -y gcc openssl-devel libffi-devel python3-devel"
        log_error "  pip3 install cryptography"
        exit 1
    fi
fi

curl -fsSL --max-time 10 "$INSTALL_URL" &>/dev/null || {
    log_error "无法访问 DevBridge 安装源 ($INSTALL_URL)"; exit 1
}
if [[ "$HAS_SYSTEMD" == "true" ]]; then
    log_info "前置检查通过 (init: systemd)"
else
    log_info "前置检查通过 (init: crontab 兜底，无 systemd)"
fi

# ========== Step 2: 安装 DevBridge CLI ==========
log_step "2/7 安装 DevBridge CLI"
export PATH="$HOME/.huawei/bin:$PATH"
if command -v devbridge &>/dev/null && devbridge version &>/dev/null 2>&1; then
    log_info "DevBridge CLI 已安装: $(devbridge version 2>/dev/null || echo 'unknown')"
else
    log_info "从 OBS 源安装 DevBridge CLI..."
    curl -fsSL "$INSTALL_URL" -o /tmp/devbridge_install.sh && bash /tmp/devbridge_install.sh 2>&1 || true
    export PATH="$HOME/.huawei/bin:$PATH"
    if ! command -v devbridge &>/dev/null; then
        log_error "安装失败，devbridge 命令未找到"; exit 1
    fi
    if ! devbridge version &>/dev/null 2>&1; then
        log_error "devbridge version 命令执行失败"; exit 1
    fi
    log_info "安装成功: $(devbridge version 2>/dev/null || echo 'installed')"
fi

# 创建软链接到 /usr/local/bin，确保任何终端都能直接使用 devbridge 命令
if [ -x "$HOME/.huawei/bin/devbridge" ] && [ ! -f /usr/local/bin/devbridge ]; then
    ln -sf "$HOME/.huawei/bin/devbridge" /usr/local/bin/devbridge 2>/dev/null || true
    log_info "已创建软链接 /usr/local/bin/devbridge"
fi

# ========== Step 3: 登录 ==========
log_step "3/7 登录 DevBridge"

# 检测当前是否已登录（不因 set -e 退出）
ALREADY_LOGGED_IN=false
if ! devbridge auth status 2>&1 | grep -qi "not logged in\|未登录"; then
    ALREADY_LOGGED_IN=true
fi

# 读取已落库的 key，与传入 key 比较是否变更
KEY_CHANGED=false
OLD_TUNNEL_ID=""
if [[ -f "$CONFIG_FILE" ]]; then
    STORED_ENC=$(grep '^API_KEY_ENC=' "$CONFIG_FILE" 2>/dev/null | sed 's/^API_KEY_ENC="//;s/"$//' || true)
    STORED_PLAIN=$(grep '^API_KEY_PLAIN=' "$CONFIG_FILE" 2>/dev/null | sed 's/^API_KEY_PLAIN="//;s/"$//' || true)
    STORED_KEY=""
    if [[ -n "$STORED_ENC" ]]; then
        STORED_KEY=$(decrypt_secret "$STORED_ENC" 2>/dev/null || true)
        if [[ -z "$STORED_KEY" ]]; then
            log_warn "旧 key 解密失败，按 key 变更处理"
            KEY_CHANGED=true
        fi
    elif [[ -n "$STORED_PLAIN" ]]; then
        STORED_KEY="$STORED_PLAIN"
    fi
    if [[ -n "$STORED_KEY" && "$STORED_KEY" != "$API_KEY" ]]; then
        KEY_CHANGED=true
    fi
    OLD_TUNNEL_ID=$(grep '^TUNNEL_ID=' "$CONFIG_FILE" 2>/dev/null | sed 's/^TUNNEL_ID="//;s/"$//' || true)
fi

# key 变更时提醒：旧隧道将成孤儿（不自动删，避免切断在用 SSH），30 天后自动过期
if [[ "$KEY_CHANGED" == "true" && -n "$OLD_TUNNEL_ID" ]]; then
    log_warn "检测到 API Key 变更，旧隧道 $OLD_TUNNEL_ID 属旧账号，将无法访问，30 天后自动过期"
    log_warn "如需立即清理：用旧账号登录后 devbridge delete $OLD_TUNNEL_ID"
fi

if [[ "$ALREADY_LOGGED_IN" == "true" && "$KEY_CHANGED" == "true" ]]; then
    log_info "检测到 API Key 变更，切换账号..."
    devbridge auth logout 2>/dev/null || true
    devbridge auth login --api-key "$API_KEY" || true
elif [[ "$ALREADY_LOGGED_IN" == "true" && "$KEY_CHANGED" == "false" ]]; then
    log_info "已登录且 key 未变，跳过登录"
else
    log_info "使用 API Key 登录..."
    devbridge auth login --api-key "$API_KEY" || true
fi

# 统一验证登录结果
if devbridge auth status 2>&1 | grep -qi "not logged in\|未登录"; then
    log_error "登录失败，请检查 API Key 是否有效"; exit 1
fi
log_info "登录成功"

# ========== Step 4: 创建/复用隧道 + 端口 ==========
log_step "4/7 创建隧道和端口"
mkdir -p "$CONFIG_DIR"

HOSTNAME_SHORT=$(uname -n | tr -cd 'a-zA-Z0-9-' | cut -c1-20)
MID_SHORT=$(cat /etc/machine-id 2>/dev/null | head -c 8)
TUNNEL_NAME="ecs-ssh-${HOSTNAME_SHORT}-${MID_SHORT}"
TUNNEL_DESC="ECSSSHtunnel"

TUNNEL_ID=""
if [[ -f "$CONFIG_FILE" ]]; then
    # shellcheck disable=SC1090
    source "$CONFIG_FILE"
    TUNNEL_ID="${TUNNEL_ID:-}"
fi

if [[ -n "$TUNNEL_ID" ]]; then
    if devbridge show "$TUNNEL_ID" &>/dev/null 2>&1; then
        log_info "复用已有隧道: $TUNNEL_ID"
    else
        log_warn "已保存的隧道 $TUNNEL_ID 已失效，将创建新隧道"
        TUNNEL_ID=""
    fi
fi

if [[ -z "$TUNNEL_ID" ]]; then
    log_info "创建隧道: $TUNNEL_NAME (有效期 ${TUNNEL_EXPIRATION}h)"
    TUNNEL_OUTPUT=$(devbridge create "$TUNNEL_NAME" -d "$TUNNEL_DESC" -e "$TUNNEL_EXPIRATION" 2>&1) || {
        TUNNEL_NAME="${TUNNEL_NAME}-$((RANDOM % 9000 + 1000))"
        TUNNEL_OUTPUT=$(devbridge create "$TUNNEL_NAME" -d "$TUNNEL_DESC" -e "$TUNNEL_EXPIRATION" 2>&1) || {
            log_error "创建隧道失败: $TUNNEL_OUTPUT"; exit 1
        }
    }
    TUNNEL_ID=$(echo "$TUNNEL_OUTPUT" | grep -i "Tunnel ID" | awk '{print $NF}')
    if [[ -z "$TUNNEL_ID" ]]; then
        log_error "无法提取 Tunnel ID: $TUNNEL_OUTPUT"; exit 1
    fi
    log_info "隧道创建成功: $TUNNEL_ID"
fi

if devbridge port list "$TUNNEL_ID" 2>/dev/null | grep -qw "$SSH_PORT"; then
    log_info "端口 $SSH_PORT 已配置"
else
    log_info "添加端口 $SSH_PORT (协议 auto, 禁止匿名访问)..."
    devbridge port create "$TUNNEL_ID" -p "$SSH_PORT" --protocol auto --deny-anonymous
    log_info "端口 $SSH_PORT 添加成功"
fi

if [[ "$PLAINTEXT" == "true" ]]; then
    API_KEY_PLAIN="$API_KEY"
    API_KEY_ENC=""
    log_warn "API Key 以明文存储（用户选择 --plaintext）"
    CONFIG_COMMENT="API Key 明文存储（用户选择 --plaintext）"
    STORAGE_DESC="明文存储"
else
    API_KEY_ENC=$(encrypt_secret "$API_KEY")
    API_KEY_PLAIN=""
    if [[ -z "$API_KEY_ENC" ]]; then
        log_error "API Key 加密失败"; exit 1
    fi
    log_info "API Key 已加密 (AES-256-GCM + PBKDF2-SHA256, machine-id 派生密钥)"
    CONFIG_COMMENT="API Key 以 AES-256-GCM + PBKDF2 加密存储，密钥从 /etc/machine-id 派生"
    STORAGE_DESC="密文存储"
fi

# 备份旧配置，便于换 key 出错时回滚
if [[ -f "$CONFIG_FILE" ]]; then
    cp "$CONFIG_FILE" "${CONFIG_FILE}.bak"
    log_info "已备份旧配置: ${CONFIG_FILE}.bak"
fi

cat > "$CONFIG_FILE" << EOF
# DevBridge SSH Tunnel Configuration
# 由 setup-devbridge-ssh.sh 自动生成，请勿手动编辑
# $CONFIG_COMMENT
TUNNEL_ID="$TUNNEL_ID"
TUNNEL_NAME="$TUNNEL_NAME"
TUNNEL_EXPIRATION=$TUNNEL_EXPIRATION
RENEW_THRESHOLD=$RENEW_THRESHOLD
SSH_PORT=$SSH_PORT
API_KEY_ENC="$API_KEY_ENC"
API_KEY_PLAIN="$API_KEY_PLAIN"
LAST_RENEWED="$(date -Iseconds)"
EOF
chmod 600 "$CONFIG_FILE"
log_info "配置已保存: $CONFIG_FILE (权限 600, API Key ${STORAGE_DESC})"

# ========== Step 5: 创建 wrapper 脚本 ==========
log_step "5/7 创建 wrapper 脚本"

cat > "$WRAPPER_SCRIPT" << 'WRAPPER_EOF'
#!/bin/bash
#=============================================================================
# DevBridge SSH Tunnel Wrapper (由 systemd 或 crontab 调用)
# 职责：解密API Key → 检查登录 → 检查隧道 → 自动续期 → 启动 host
#=============================================================================
set -euo pipefail
export PATH="$HOME/.huawei/bin:$PATH"

CONFIG_FILE="$HOME/.huawei/devbridge/ssh-tunnel.conf"
LOG_PREFIX="[devbridge-ssh]"
ENCRYPTION_SALT="devbridge-ssh-tunnel-v1"

get_machine_pass() {
    local mid
    mid=$(cat /etc/machine-id 2>/dev/null || cat /var/lib/dbus/machine-id 2>/dev/null || uname -n)
    echo -n "${mid}${ENCRYPTION_SALT}"
}
decrypt_secret() {
    local enc="$1" pass
    pass=$(get_machine_pass)
    _DB_PASS="$pass" _DB_ENC="$enc" python3 -c "
import os, base64
from cryptography.hazmat.primitives.ciphers.aead import AESGCM
from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2HMAC
from cryptography.hazmat.primitives import hashes
pw = os.environ['_DB_PASS'].encode()
raw = base64.b64decode(os.environ['_DB_ENC'])
salt, nonce, ct = raw[:16], raw[16:28], raw[28:]
kdf = PBKDF2HMAC(algorithm=hashes.SHA256(), length=32, salt=salt, iterations=100000)
key = kdf.derive(pw)
pt = AESGCM(key).decrypt(nonce, ct, None)
print(pt.decode(), end='')
" 2>/dev/null
}

if [[ ! -f "$CONFIG_FILE" ]]; then
    echo "$LOG_PREFIX 配置文件不存在: $CONFIG_FILE" >&2; exit 1
fi
# shellcheck disable=SC1090
source "$CONFIG_FILE"

TUNNEL_ID="${TUNNEL_ID:-}"
TUNNEL_EXPIRATION="${TUNNEL_EXPIRATION:-720}"
RENEW_THRESHOLD="${RENEW_THRESHOLD:-24}"
SSH_PORT="${SSH_PORT:-22}"
API_KEY_ENC="${API_KEY_ENC:-}"
API_KEY_PLAIN="${API_KEY_PLAIN:-}"
LAST_RENEWED="${LAST_RENEWED:-}"

if [[ -z "$TUNNEL_ID" ]]; then
    echo "$LOG_PREFIX 配置中无 TUNNEL_ID" >&2; exit 1
fi

API_KEY=""
if [[ -n "$API_KEY_ENC" ]]; then
    API_KEY=$(decrypt_secret "$API_KEY_ENC")
    if [[ -z "$API_KEY" ]]; then
        echo "$LOG_PREFIX API Key 解密失败（machine-id 不匹配？）" >&2; exit 1
    fi
elif [[ -n "$API_KEY_PLAIN" ]]; then
    API_KEY="$API_KEY_PLAIN"
fi

# ---- Step A: 检查登录 ----
echo "$LOG_PREFIX 检查登录状态..." >&2
if devbridge auth status 2>&1 | grep -qi "not logged in\|未登录"; then
    echo "$LOG_PREFIX 登录已过期，用 API Key 重新登录..." >&2
    if [[ -z "$API_KEY" ]]; then
        echo "$LOG_PREFIX 无 API Key，无法登录" >&2; exit 1
    fi
    if ! devbridge auth login --api-key "$API_KEY" 2>/dev/null; then
        echo "$LOG_PREFIX API Key 登录失败" >&2; exit 1
    fi
    echo "$LOG_PREFIX 重新登录成功" >&2
else
    echo "$LOG_PREFIX 登录正常" >&2
fi

# ---- Step B: 检查隧道是否存在，不存在则重建 ----
if ! devbridge show "$TUNNEL_ID" &>/dev/null 2>&1; then
    echo "$LOG_PREFIX 隧道 $TUNNEL_ID 不存在，重建..." >&2
    HOSTNAME_SHORT=$(uname -n | tr -cd 'a-zA-Z0-9-' | cut -c1-20)
    MID_SHORT=$(cat /etc/machine-id 2>/dev/null | head -c 8)
    TUNNEL_NAME="ecs-ssh-${HOSTNAME_SHORT}-${MID_SHORT}"
    TUNNEL_OUTPUT=$(devbridge create "$TUNNEL_NAME" -d "ECSSSHtunnel" -e "$TUNNEL_EXPIRATION" 2>&1) || {
        TUNNEL_NAME="${TUNNEL_NAME}-$((RANDOM % 9000 + 1000))"
        TUNNEL_OUTPUT=$(devbridge create "$TUNNEL_NAME" -d "ECSSSHtunnel" -e "$TUNNEL_EXPIRATION" 2>&1)
    }
    NEW_ID=$(echo "$TUNNEL_OUTPUT" | grep -i "Tunnel ID" | awk '{print $NF}')
    if [[ -z "$NEW_ID" ]]; then
        echo "$LOG_PREFIX 重建失败: $TUNNEL_OUTPUT" >&2; exit 1
    fi
    TUNNEL_ID="$NEW_ID"
    devbridge port create "$TUNNEL_ID" -p "$SSH_PORT" --protocol auto --deny-anonymous 2>/dev/null || true
    cat > "$CONFIG_FILE" << EOF
# DevBridge SSH Tunnel Configuration
# API Key 存储（加密或明文，取决于配置）
TUNNEL_ID="$TUNNEL_ID"
TUNNEL_NAME="$TUNNEL_NAME"
TUNNEL_EXPIRATION=$TUNNEL_EXPIRATION
RENEW_THRESHOLD=$RENEW_THRESHOLD
SSH_PORT=$SSH_PORT
API_KEY_ENC="$API_KEY_ENC"
API_KEY_PLAIN="$API_KEY_PLAIN"
LAST_RENEWED="$(date -Iseconds)"
EOF
    chmod 600 "$CONFIG_FILE"
    echo "$LOG_PREFIX 隧道已重建: $TUNNEL_ID" >&2
else
    # ---- Step C: 检查续期 ----
    NEED_RENEW=false
    if [[ -n "$LAST_RENEWED" ]]; then
        NOW_EPOCH=$(date +%s)
        LAST_EPOCH=$(date -d "$LAST_RENEWED" +%s 2>/dev/null || echo 0)
        if [[ "$LAST_EPOCH" -gt 0 ]]; then
            ELAPSED_HOURS=$(( (NOW_EPOCH - LAST_EPOCH) / 3600 ))
            REMAINING_HOURS=$(( TUNNEL_EXPIRATION - ELAPSED_HOURS ))
            echo "$LOG_PREFIX 剩余约 ${REMAINING_HOURS}h" >&2
            [[ "$REMAINING_HOURS" -lt "$RENEW_THRESHOLD" ]] && NEED_RENEW=true
        else
            NEED_RENEW=true
        fi
    else
        NEED_RENEW=true
    fi
    if [[ "$NEED_RENEW" == "true" ]]; then
        echo "$LOG_PREFIX 续期到 ${TUNNEL_EXPIRATION}h..." >&2
        if devbridge update "$TUNNEL_ID" -e "$TUNNEL_EXPIRATION" 2>/dev/null; then
            sed -i "s|^LAST_RENEWED=.*|LAST_RENEWED=\"$(date -Iseconds)\"|" "$CONFIG_FILE"
            echo "$LOG_PREFIX 续期成功" >&2
        else
            echo "$LOG_PREFIX 续期失败，继续 host..." >&2
        fi
    fi
fi

# ---- Step D: 启动 host（阻塞）----
echo "$LOG_PREFIX 启动 host，隧道: $TUNNEL_ID" >&2
exec devbridge host "$TUNNEL_ID"
WRAPPER_EOF
chmod +x "$WRAPPER_SCRIPT"
log_info "Wrapper 脚本已创建: $WRAPPER_SCRIPT"

# ========== Step 6: 设置开机自启 ==========
log_step "6/7 设置开机自启"
CURRENT_USER=$(whoami)

if [[ "$HAS_SYSTEMD" == "true" ]]; then
    # --- systemd 方式 ---
    cat > "$SERVICE_FILE" << EOF
[Unit]
Description=DevBridge SSH Tunnel for ECS
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${CURRENT_USER}
Environment=PATH=${HOME}/.huawei/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
Environment=HOME=${HOME}
ExecStart=${WRAPPER_SCRIPT}
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME" 2>/dev/null
    log_info "systemd 服务已创建并设置开机自启: $SERVICE_NAME"
else
    # --- crontab @reboot 兜底 ---
    if command -v crontab >/dev/null 2>&1; then
        CRON_LINE="@reboot ${WRAPPER_SCRIPT} >> ${LOG_FILE} 2>&1"
        # 移除旧条目
        (crontab -l 2>/dev/null | grep -v "$WRAPPER_SCRIPT"; echo "$CRON_LINE") | crontab -
        log_info "crontab @reboot 已设置开机自启"
        log_info "日志文件: $LOG_FILE"
    else
        log_warn "systemd 和 crontab 均不可用，跳过开机自启设置"
        log_warn "当前会话仍会启动 host，但重启后需手动运行: $WRAPPER_SCRIPT"
    fi
fi

# ========== Step 7: 启动并验证 ==========
log_step "7/7 启动并验证"

# 先杀掉可能残留的 devbridge host 进程（隧道只允许一个 Host 连接）
pkill -f "devbridge host" 2>/dev/null || true
sleep 1

if [[ "$HAS_SYSTEMD" == "true" ]]; then
    systemctl stop "$SERVICE_NAME" 2>/dev/null || true
    systemctl start "$SERVICE_NAME"
    for i in $(seq 1 15); do
        if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
            break
        fi
        sleep 1
    done
    if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
        log_info "服务已启动并运行中"
    else
        log_error "服务启动失败"
        log_error "最近日志:"
        journalctl -u "$SERVICE_NAME" --no-pager -n 20 2>/dev/null || true
        exit 1
    fi
else
    # nohup 启动 wrapper
    if [[ -f "$PID_FILE" ]] && kill -0 "$(cat "$PID_FILE" 2>/dev/null)" 2>/dev/null; then
        kill "$(cat "$PID_FILE")" 2>/dev/null || true
        sleep 1
    fi
    nohup "$WRAPPER_SCRIPT" >> "$LOG_FILE" 2>&1 &
    echo $! > "$PID_FILE"
    sleep 2
    if kill -0 "$(cat "$PID_FILE" 2>/dev/null)" 2>/dev/null; then
        log_info "进程已启动 (PID: $(cat "$PID_FILE"))"
    else
        log_error "进程启动失败，查看日志: $LOG_FILE"
        tail -20 "$LOG_FILE" 2>/dev/null || true
        exit 1
    fi
fi

# ========== 输出汇总 ==========
echo ""
echo "============================================================"
echo "   DevBridge SSH Tunnel 安装完成"
echo "============================================================"
echo ""
echo " 隧道 ID:    $TUNNEL_ID"
echo " SSH 端口:   $SSH_PORT"
echo " 有效期:     ${TUNNEL_EXPIRATION}h (剩余 < ${RENEW_THRESHOLD}h 自动续期)"
echo " 运行用户:   $CURRENT_USER"
if [[ "$PLAINTEXT" == "true" ]]; then
echo " API Key:    明文存储 (--plaintext)"
else
echo " API Key:    已加密存储 (AES-256-GCM + PBKDF2)"
fi
if [[ "$HAS_SYSTEMD" == "true" ]]; then
echo " 自启方式:   systemd ($SERVICE_NAME)"
elif command -v crontab >/dev/null 2>&1; then
echo " 自启方式:   crontab @reboot"
else
echo " 自启方式:   无（systemd/crontab 均不可用，需手动启动）"
fi
echo ""
echo "------------------------------------------------------------"
echo " 从另一台设备连接此 ECS 的 SSH："
echo "------------------------------------------------------------"
echo ""
echo "   1. 在你的设备上安装 DevBridge CLI"
echo "      curl -fsSL https://tools-artifact.developer.huaweicloud.com/sharedata/devbridge/install.sh | bash"
echo ""
echo "   2. 登录"
echo "      devbridge auth login"
echo ""
echo "   3. 连接隧道"
echo "      devbridge connect $TUNNEL_ID"
echo "      # 注意输出中的端口映射: localhost:XXXXX -> tunnel port: 22"
echo ""
echo "   4. SSH 连接"
echo "      # 本地 22 端口空闲时:  ssh ${CURRENT_USER}@localhost"
echo "      # 本地 22 端口被占用时: ssh ${CURRENT_USER}@localhost -p <步骤3输出的端口号>"
echo ""
echo "------------------------------------------------------------"
echo " 服务管理："
echo "------------------------------------------------------------"
echo ""
if [[ "$HAS_SYSTEMD" == "true" ]]; then
echo "   查看状态:   systemctl status  $SERVICE_NAME"
echo "   停止:       systemctl stop     $SERVICE_NAME"
echo "   启动:       systemctl start    $SERVICE_NAME"
echo "   查看日志:   journalctl -u      $SERVICE_NAME -f"
else
echo "   查看状态:   kill -0 \$(cat $PID_FILE) && echo running || echo stopped"
echo "   停止:       kill \$(cat $PID_FILE)"
echo "   启动:       nohup $WRAPPER_SCRIPT >> $LOG_FILE 2>&1 &"
echo "   查看日志:   tail -f $LOG_FILE"
fi
echo "   查看隧道:   devbridge show $TUNNEL_ID"
echo "   回滚:       cp ~/.huawei/devbridge/ssh-tunnel.conf.bak ~/.huawei/devbridge/ssh-tunnel.conf"
echo ""
echo "============================================================"
